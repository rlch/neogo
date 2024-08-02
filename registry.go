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

func (r *registry) bindValue(from any, to reflect.Value) (err error) {
	toT := to.Type()
	if to.Kind() == reflect.Ptr && toT.Elem() == emptyInterface {
		to.Elem().Set(reflect.ValueOf(from))
		return nil
	} else if toT == emptyInterface && to.CanSet() {
		to.Set(reflect.ValueOf(from))
		return nil
	}

	// Valuer through Node / relationship
	switch fromVal := from.(type) {
	case neo4j.Node:
		ok, err := bindValuer(fromVal, to)
		if err != nil {
			return err
		}
		if ok {
			return nil
		}
		innerT := toT
		for innerT.Kind() == reflect.Ptr {
			innerT = innerT.Elem()
		}
		if (toT.Implements(rAbstract) ||
			toT.Elem().Implements(rAbstract)) &&
			// We enforce that abstract nodes must be interfaces. Some hacking could
			// relax this.
			innerT.Kind() == reflect.Interface {
			return r.bindAbstractNode(fromVal, to)
		}
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

	// Valuer throuh any other RecordValue
	var ok bool
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

	// Recurse into slices
	switch reflect.TypeOf(from).Kind() {
	case reflect.Slice:
		if to.Kind() == reflect.Ptr {
			to = to.Elem()
		}
		if to.Kind() != reflect.Slice {
			return errors.New("cannot bind slice to non-slice type")
		}
		fromV := reflect.ValueOf(from)
		n := fromV.Len()
		to.Set(reflect.MakeSlice(to.Type(), n, n))
		for i := 0; i < n; i++ {
			fromI := fromV.Index(i).Interface()
			err := r.bindValue(fromI, to.Index(i))
			if err != nil {
				return fmt.Errorf("error binding slice element %d: %w", i, err)
			}
		}
		return nil
	}

	// Primitive coercion.
	value := unwindValue(to)
	ok, err = func() (bool, error) {
		if !to.CanSet() {
			return false, nil
		}
		i := value.Interface()
		switch i.(type) {
		case bool:
			return true, bindCasted(cast.ToBoolE, from, value)
		case string:
			return true, bindCasted(cast.ToStringE, from, value)
		case int:
			return true, bindCasted(cast.ToIntE, from, value)
		case int8:
			return true, bindCasted(cast.ToInt8E, from, value)
		case int16:
			return true, bindCasted(cast.ToInt16E, from, value)
		case int32:
			return true, bindCasted(cast.ToInt32E, from, value)
		case int64:
			return true, bindCasted(cast.ToInt64E, from, value)
		case uint:
			return true, bindCasted(cast.ToUintE, from, value)
		case uint8:
			return true, bindCasted(cast.ToUint8E, from, value)
		case uint16:
			return true, bindCasted(cast.ToUint16E, from, value)
		case uint32:
			return true, bindCasted(cast.ToUint32E, from, value)
		case uint64:
			return true, bindCasted(cast.ToUint64E, from, value)
		case float32:
			return true, bindCasted(cast.ToFloat32E, from, value)
		case float64:
			return true, bindCasted(cast.ToFloat64E, from, value)
		case []int:
			return true, bindCasted(cast.ToIntSliceE, from, value)
		case []string:
			return true, bindCasted(cast.ToStringSliceE, from, value)
		case time.Time:
			return true, bindCasted(cast.ToTimeE, from, value)
		case time.Duration:
			return true, bindCasted(cast.ToDurationE, from, value)
		}
		return false, nil
	}()
	if ok && err == nil {
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

	var (
		abs                IAbstract
		inheritanceCounter int
	)
	ptrTo := false
	canBindSubtype := true
	if to.Type().Implements(rAbstract) {
		if !to.IsNil() {
			canBindSubtype = false
		}
	} else if to.Type().Elem().Implements(rAbstract) {
		ptrTo = true
		if !to.Elem().IsNil() {
			abs = to.Elem().Interface().(IAbstract)
		}
	} else {
		return errors.New("cannot bind abstract node to non-abstract type")
	}
	// We find the abstract node (or exact implementation if registered) that has
	// a inheritance chain closest to the database node we're extracting from.
	// i.e. If we have a concrete-node with inheritance chain A > B > C, we prefer
	// A > B as a potential subtype over A.
	if abs == nil {
	Bases:
		for _, base := range r.abstractNodes {
			labels := internal.ExtractNodeLabels(base)
			if len(labels) == 0 {
				continue
			}
			currentInheritanceCounter := 0
			for _, label := range labels {
				if _, ok := isNodeLabel[label]; !ok {
					continue Bases
				}
				currentInheritanceCounter++
			}
			if currentInheritanceCounter > inheritanceCounter {
				abs = base
				inheritanceCounter = currentInheritanceCounter
			}
		}
		if abs == nil {
			return fmt.Errorf(
				"no abstract node found for labels: %s\nDid you forget to register the base node using neogo.WithTypes(...)?",
				strings.Join(nodeLabels, ", "),
			)
		}
	}

	// We found our impl
	var impl IAbstract
	if inheritanceCounter == len(nodeLabels) {
		impl = abs
	} else {
		if !canBindSubtype {
			return fmt.Errorf(
				"cannot bind abstract subtype to non-nil abstract type, as value-types cannot be reassigned.\nTry using *%s",
				to.Type(),
			)
		}
	Impls:
		for _, nextImpl := range abs.Implementers() {
			for _, label := range internal.ExtractNodeLabels(nextImpl) {
				if _, ok := isNodeLabel[label]; !ok {
					continue Impls
				}
			}
			impl = nextImpl
			break
		}
	}
	if impl == nil {
		return fmt.Errorf(
			"no concrete implementation found for labels: %s\nDid you forget to register the base node using neogo.WithTypes(...)?",
			strings.Join(nodeLabels, ", "),
		)
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
