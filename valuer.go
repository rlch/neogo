package neogo

import (
	"errors"
	"fmt"
	"reflect"
	"strings"
	"time"

	"github.com/goccy/go-json"
	"github.com/neo4j/neo4j-go-driver/v5/neo4j"
	"github.com/spf13/cast"

	"github.com/rlch/neogo/internal"
)

// Valuer allows arbitrary types to be marshalled into and unmarshalled from
// Neo4J data types. This allows any type (as oppposed to stdlib types, [INode],
// [IAbstract], [IRelationship], and structs with json tags) to be used with
// [neogo]. The valid Neo4J data types are defined by [neo4j.RecordValue].
//
// For example, here we define a custom type MyString that marshals to and
// from a string, one of the types in the [neo4j.RecordValue] union:
//
//	type MyString string
//
//	var _ Valuer[string] = (*MyString)(nil)
//
//	func (s MyString) Marshal() (*string, error) {
//		return func(s string) *string {
//			return &s
//		}(string(s)), nil
//	}
//
//	func (s *MyString) Unmarshal(v *string) error {
//		*s = MyString(*v)
//		return nil
//	}
type Valuer[V neo4j.RecordValue] interface {
	Marshal() (*V, error)
	Unmarshal(*V) error
}

type registry struct {
	abstractNodes []IAbstract
	nodes         []INode
	relationships []IRelationship
}

// WithTypes is an option for [New] that allows you to register instances of
// [IAbstract], [INode] and [IRelationship] to be used with [neogo].
func WithTypes(types ...any) func(*driver) {
	return func(d *driver) {
		for _, t := range types {
			if v, ok := t.(IAbstract); ok {
				d.abstractNodes = append(d.abstractNodes, v)
				continue
			}
			if v, ok := t.(INode); ok {
				d.nodes = append(d.nodes, v)
				continue
			}
			if v, ok := t.(IRelationship); ok {
				d.relationships = append(d.relationships, v)
				continue
			}
		}
	}
}

func unwindValue(ptrTo reflect.Value) reflect.Value {
	for ptrTo.Kind() == reflect.Ptr {
		ptrTo = ptrTo.Elem()
	}
	return ptrTo
}

func bindValuer[V neo4j.RecordValue](value V, bindTo reflect.Value) (ok bool, err error) {
	i := bindTo.Interface()
	valuer, ok := i.(Valuer[V])
	if !ok {
		return false, nil
	}
	if err := valuer.Unmarshal(&value); err != nil {
		return false, err
	}
	return true, nil
}

func bindCasted[C any](
	cast func(any) (C, error),
	value any,
	bindTo reflect.Value,
) error {
	c, err := cast(value)
	if err != nil {
		return err
	}
	bindTo.Set(reflect.ValueOf(c))
	return nil
}

var emptyInterface = reflect.TypeOf((*any)(nil)).Elem()

func (r *registry) bindValue(from any, to reflect.Value) error {
	if to.Kind() == reflect.Ptr && to.Type().Elem() == emptyInterface {
		to.Elem().Set(reflect.ValueOf(from))
		return nil
	} else if to.Type() == emptyInterface && to.CanSet() {
		to.Set(reflect.ValueOf(from))
		return nil
	}

	switch fromVal := from.(type) {
	case neo4j.Node:
		ok, err := bindValuer(fromVal, to)
		if err != nil {
			return err
		}
		if ok {
			return nil
		}
		if to.Type().Implements(rAbstract) ||
			to.Elem().Type().Implements(rAbstract) {
			return r.bindAbstractNode(fromVal, to)
		}
		// TODO: Support abstract nodes
		return r.bindValue(fromVal.Props, to)
	case neo4j.Relationship:
		ok, err := bindValuer(fromVal, to)
		if err != nil {
			return err
		}
		if ok {
			return nil
		}
		return r.bindValue(fromVal.Props, to)
	}

	// First, check for primitive coercion.
	value := unwindValue(to)
	ok, err := func() (bool, error) {
		if !to.CanSet() {
			return false, nil
		}
		i := value.Interface()
		switch i.(type) {
		case bool:
			return true, bindCasted[bool](cast.ToBoolE, from, value)
		case string:
			return true, bindCasted[string](cast.ToStringE, from, value)
		case int:
			return true, bindCasted[int](cast.ToIntE, from, value)
		case int8:
			return true, bindCasted[int8](cast.ToInt8E, from, value)
		case int16:
			return true, bindCasted[int16](cast.ToInt16E, from, value)
		case int32:
			return true, bindCasted[int32](cast.ToInt32E, from, value)
		case int64:
			return true, bindCasted[int64](cast.ToInt64E, from, value)
		case uint:
			return true, bindCasted[uint](cast.ToUintE, from, value)
		case uint8:
			return true, bindCasted[uint8](cast.ToUint8E, from, value)
		case uint16:
			return true, bindCasted[uint16](cast.ToUint16E, from, value)
		case uint32:
			return true, bindCasted[uint32](cast.ToUint32E, from, value)
		case uint64:
			return true, bindCasted[uint64](cast.ToUint64E, from, value)
		case float32:
			return true, bindCasted[float32](cast.ToFloat32E, from, value)
		case float64:
			return true, bindCasted[float64](cast.ToFloat64E, from, value)
		case []int:
			return true, bindCasted[[]int](cast.ToIntSliceE, from, value)
		case []string:
			return true, bindCasted[[]string](cast.ToStringSliceE, from, value)
		case time.Time:
			return true, bindCasted[time.Time](cast.ToTimeE, from, value)
		case time.Duration:
			return true, bindCasted[time.Duration](cast.ToDurationE, from, value)
		}
		return false, nil
	}()
	if ok && err == nil {
		return nil
	}

	// Next, we check if to implements Valuer.
	ok, err = func() (bool, error) {
		switch fromVal := from.(type) {
		case bool:
			return bindValuer(fromVal, to)
		case int64:
			return bindValuer(fromVal, to)
		case float64:
			return bindValuer(fromVal, to)
		case string:
			return bindValuer(fromVal, to)
		case neo4j.Point2D:
			return bindValuer(fromVal, to)
		case neo4j.Point3D:
			return bindValuer(fromVal, to)
		case neo4j.Date:
			return bindValuer(fromVal, to)
		case neo4j.LocalTime:
			return bindValuer(fromVal, to)
		case neo4j.LocalDateTime:
			return bindValuer(fromVal, to)
		case neo4j.Time:
			return bindValuer(fromVal, to)
		case neo4j.Duration:
			return bindValuer(fromVal, to)
		case time.Time:
			return bindValuer(fromVal, to)
		case []byte:
			return bindValuer(fromVal, to)
		case []any:
			return bindValuer(fromVal, to)
		case map[string]any:
			return bindValuer(fromVal, to)
		}
		return false, nil
	}()
	if err != nil {
		return err
	}
	if ok {
		return nil
	}

	// PERF: Obviously huge performance hit here. Consider alternative ways of
	// coercing between types. Might just need to be imperative and verbose
	bytes, err := json.Marshal(from)
	if err != nil {
		return err
	}
	err = json.Unmarshal(bytes, to.Interface())
	if err != nil {
		return err
	}
	return nil
}

func (r *registry) bindAbstractNode(node neo4j.Node, to reflect.Value) error {
	nodeLabels := node.Labels
	isNodeLabel := make(map[string]struct{}, len(nodeLabels))
	for _, label := range nodeLabels {
		isNodeLabel[label] = struct{}{}
	}

	var abs IAbstract
	ptrTo := false
	if to.Type().Implements(rAbstract) {
		if !to.IsNil() {
			return fmt.Errorf(
				"cannot bind abstract node to non-nil abstract type, as the type should not be deterministic.\nTry using *%T",
				abs,
			)
		}
	} else if to.Type().Elem().Implements(rAbstract) {
		ptrTo = true
		if !to.Elem().IsNil() {
			abs = to.Elem().Interface().(IAbstract)
		}
	} else {
		return errors.New("cannot bind abstract node to non-abstract type")
	}
	if abs == nil {
	Bases:
		for _, base := range r.abstractNodes {
			labels := internal.ExtractNodeLabels(base)
			if len(labels) == 0 {
				continue
			}
			fmt.Println(labels)
			for _, label := range labels {
				if _, ok := isNodeLabel[label]; !ok {
					continue Bases
				}
			}
			abs = base
		}
		if abs == nil {
			return fmt.Errorf(
				"no abstract node found for labels: %s\nDid you forget to register the base node using neogo.WithTypes(...)?",
				strings.Join(nodeLabels, ", "),
			)
		}
	}

	abstractLabels := internal.ExtractNodeLabels(abs)
	isAbstractLabel := make(map[string]struct{}, len(abstractLabels))
	for _, label := range abstractLabels {
		isAbstractLabel[label] = struct{}{}
	}
	for _, impl := range abs.Implementers() {
		for _, label := range internal.ExtractNodeLabels(impl) {
			if _, ok := isAbstractLabel[label]; ok {
				continue
			}
			if _, ok := isNodeLabel[label]; !ok {
				continue
			}
			toImpl := reflect.New(reflect.TypeOf(impl).Elem())
			err := r.bindValue(node.Props, toImpl)
			if err != nil {
				return err
			}
			if ptrTo {
				to.Elem().Set(toImpl)
			} else {
				to.Set(toImpl)
			}
			return nil
		}
	}
	return nil
}
