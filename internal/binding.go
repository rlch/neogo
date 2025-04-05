package internal

import (
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"time"

	"github.com/neo4j/neo4j-go-driver/v5/neo4j"
	"github.com/spf13/cast"
)

var rAbstract = reflect.TypeOf((*IAbstract)(nil)).Elem()

type Valuer[V neo4j.RecordValue] interface {
	Marshal() (*V, error)
	Unmarshal(*V) error
}

func unwindValue(ptrTo reflect.Value) reflect.Value {
	for ptrTo.Kind() == reflect.Ptr {
		ptrTo = ptrTo.Elem()
	}
	return ptrTo
}

func unwindType(ptrTo reflect.Type) reflect.Type {
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

func (r *Registry) BindValue(from any, to reflect.Value) (err error) {
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
				return r.BindAbstractNode(fromVal, to)
			}
			return r.BindValue(fromVal.Props, to)
		case neo4j.Relationship:
			ok, err := bindValuer(fromVal, to)
			if err != nil {
				return err
			}
			if ok {
				return nil
			}
			return r.BindValue(fromVal.Props, to)
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
				toI := to.Index(i)
				if toI.CanAddr() {
					toI = toI.Addr()
				}
				err := r.BindValue(fromI, toI)
				if err != nil {
					return fmt.Errorf("error binding slice element %d: %w", i, err)
				}
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
	// NOTE: a nil record is considered an empty list!
	if from != nil && reflect.TypeOf(from).Kind() != reflect.Slice {
		sliceV := to
		for sliceV.Kind() == reflect.Ptr {
			sliceV = sliceV.Elem()
		}
		if sliceV.Kind() == reflect.Slice {
			sliceV.Set(reflect.MakeSlice(sliceV.Type(), 1, 1))
			return r.BindValue(from, sliceV.Index(0).Addr())
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

func (r *Registry) BindAbstractNode(node neo4j.Node, to reflect.Value) error {
	ptrTo := false
	if to.Type().Implements(rAbstract) {
		// if !to.IsNil() {
		// 	canBindSubtype = false
		// }
	} else if to.Type().Elem().Implements(rAbstract) {
		ptrTo = true
	} else {
		return errors.New("cannot bind abstract node to non-abstract type. Ensure your binding type or the value it references implements IAbstract")
	}

	implNode, err := r.GetConcreteImplementation(node.Labels)
	if err != nil {
		return err
	}
	impl := implNode.typ
	toImpl := reflect.New(reflect.TypeOf(impl).Elem())
	err = r.BindValue(node.Props, toImpl)
	if err != nil {
		return err
	}
	// if !canBindSubtype {
	// 	return fmt.Errorf(
	// 		"cannot bind subtype to non-nil abstract type, as value-types cannot be reassigned.\nTry using *%s",
	// 		to.Type(),
	// 	)
	if ptrTo {
		to.Elem().Set(toImpl)
	} else {
		to.Set(toImpl)
	}
	return nil
}
