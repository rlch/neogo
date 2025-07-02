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
	abstractNodes []any
	nodes         []any
	relationships []any
}

func (r *registry) registerTypes(types ...any) {
	if r.abstractNodes == nil {
		r.abstractNodes = []any{}
	}
	if r.nodes == nil {
		r.nodes = []any{}
	}
	if r.relationships == nil {
		r.relationships = []any{}
	}
	for _, t := range types {
		if _, ok := t.(IAbstract); ok {
			r.abstractNodes = append(r.abstractNodes, t)
			continue
		}
		if v, ok := t.(INode); ok {
			r.nodes = append(r.nodes, v)
			continue
		}
		if v, ok := t.(IRelationship); ok {
			r.relationships = append(r.relationships, v)
			continue
		}
	}
}

func unwindType(ptrTo reflect.Type) reflect.Type {
	for ptrTo.Kind() == reflect.Ptr {
		ptrTo = ptrTo.Elem()
	}
	return ptrTo
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

	var ok bool
	if from != nil {
		handleSingleRecordToSlice := func(fromVal any) error {
			sliceV := to
			for sliceV.Kind() == reflect.Ptr {
				sliceV = sliceV.Elem()
			}
			sliceV.Set(reflect.MakeSlice(sliceV.Type(), 1, 1))
			return r.bindValue(fromVal, sliceV.Index(0).Addr())
		}
		// Valuer through Node / relationship
		switch fromVal := from.(type) {
		case neo4j.Node:
			// Handle 1 record of an expected slice of nodes
			if unwindType(toT).Kind() == reflect.Slice {
				return handleSingleRecordToSlice(fromVal)
			}
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
			// Handle 1 record of an expected slice of relationships
			if unwindType(toT).Kind() == reflect.Slice {
				return handleSingleRecordToSlice(fromVal)
			}
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

		// Recursively deserialize slices
		fromT := reflect.TypeOf(from)
		switch fromT.Kind() {
		case reflect.Slice:
			if to.Kind() == reflect.Ptr {
				to = to.Elem()
			}
			if to.Kind() != reflect.Slice {
				return errors.New("cannot bind slice to non-slice type")
			}
			toT = to.Type()
			fromV := reflect.ValueOf(from)
			n := fromV.Len()
			// If the depth of from and to is equal, there's a 1:1 relationship between the record and the output type.
			// If the depth of from is 1 lower than that of to, we assume the result from neo4j is a single record representing the first
			// element of the slice of the output, to.
			fromDepth, toDepth := computeDepth(fromT), computeDepth(toT)
			if fromDepth == toDepth {
				to.Set(reflect.MakeSlice(toT, n, n))
				for i := range n {
					fromI := fromV.Index(i).Interface()
					toI := to.Index(i)
					if toI.CanAddr() {
						toI = toI.Addr()
					}
					err := r.bindValue(fromI, toI)
					if err != nil {
						return fmt.Errorf("error binding slice element %d: %w", i, err)
					}
				}
			} else if fromDepth+1 == toDepth {
				to.Set(reflect.MakeSlice(toT, 1, 1))
				err := r.bindValue(from, to.Index(0))
				if err != nil {
					return fmt.Errorf("error binding value to first index of slice: %w", err)
				}
			} else {
				return fmt.Errorf("cannot bind slice of depth %d to slice of depth %d", fromDepth, toDepth)
			}

			return nil
		}

		// Primitive coercion.
		value := unwindValue(to)
		ok, err = func() (bool, error) {
			if !to.CanSet() || !value.IsValid() || !value.CanInterface() {
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
	}

	// This handles a slice of length 1, treated as a single record value.
	// When binding a single value (including nil) to a slice, create a slice with one element.
	sliceV := to
	for sliceV.Kind() == reflect.Ptr {
		sliceV = sliceV.Elem()
	}
	if sliceV.Kind() == reflect.Slice {
		// Handle non-slice values (including nil) by creating a slice with one element
		if from == nil || reflect.TypeOf(from).Kind() != reflect.Slice {
			sliceV.Set(reflect.MakeSlice(sliceV.Type(), 1, 1))
			return r.bindValue(from, sliceV.Index(0).Addr())
		}
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
		abs                any
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
			abs = to.Elem().Interface()
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
			labels := internal.ExtractConcreteNodeLabels(base)
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
	var impl any
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
		for _, nextImpl := range abs.(IAbstract).Implementers() {
			for _, label := range internal.ExtractConcreteNodeLabels(nextImpl) {
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

func computeDepth(t reflect.Type) (depth int) {
	for t.Kind() == reflect.Slice {
		depth++
		t = t.Elem()
	}
	return
}
