package neogo

import (
	"errors"
	"reflect"
	"time"

	"github.com/goccy/go-json"
	"github.com/neo4j/neo4j-go-driver/v5/neo4j"
	"github.com/spf13/cast"
)

// bool | int64 | float64 | string |
// Point2D | Point3D |
// Date | LocalTime | LocalDateTime | Time | Duration | time.Time |
// []byte | []any | map[string]any |
// Node | Relationship | Path

type Valuer[V neo4j.RecordValue] interface {
	Marshal() (V, error)
	Unmarshal(V) error
}

// var (
// 	boolV          = (*Valuer[bool])(nil)
// 	intV           = (*Valuer[int64])(nil)
// 	floatV         = (*Valuer[float64])(nil)
// 	stringV        = (*Valuer[string])(nil)
// 	point2dV       = (*Valuer[neo4j.Point2D])(nil)
// 	point3dV       = (*Valuer[neo4j.Point3D])(nil)
// 	dateV          = (*Valuer[neo4j.Date])(nil)
// 	localTimeV     = (*Valuer[neo4j.LocalTime])(nil)
// 	localDateTimeV = (*Valuer[neo4j.LocalDateTime])(nil)
// 	timeV          = (*Valuer[neo4j.Time])(nil)
// 	durationV      = (*Valuer[neo4j.Duration])(nil)
// 	timeTimeV      = (*Valuer[time.Time])(nil)
// 	bytesV         = (*Valuer[[]byte])(nil)
// 	arrayV         = (*Valuer[[]any])(nil)
// 	mapV           = (*Valuer[map[string]any])(nil)
// 	nodeV          = (*Valuer[neo4j.Node])(nil)
// 	relationshipV  = (*Valuer[neo4j.Relationship])(nil)
// 	pathV          = (*Valuer[neo4j.Path])(nil)
// )

func unwindValue(ptrTo reflect.Value) reflect.Value {
	for ptrTo.Kind() == reflect.Ptr {
		ptrTo = ptrTo.Elem()
	}
	return ptrTo
}

func bindValuer[V neo4j.RecordValue](value V, val reflect.Value) (ok bool, err error) {
	i := val.Interface()
	valuer, ok := i.(Valuer[V])
	if !ok {
		return false, nil
	}
	if err := valuer.Unmarshal(value); err != nil {
		return false, err
	}
	return true, nil
}

func bindCasted[C any](
	cast func(any) (C, error),
	value any,
	ptrToVal reflect.Value,
) error {
	c, err := cast(value)
	if err != nil {
		return err
	}
	ptrToVal.Elem().Set(reflect.ValueOf(c))
	return nil
}

func bindValue(fromVal any, to reflect.Value) error {
	if !to.CanSet() {
		return errors.New("cannot set value")
	}

	// First, check for primitive coercion.
	val := unwindValue(to)
	_ = func() error {
		i := val.Interface()
		switch i.(type) {
		case bool:
			return bindCasted[bool](cast.ToBoolE, fromVal, val)
		case string:
			return bindCasted[string](cast.ToStringE, fromVal, val)
		case int:
			return bindCasted[int](cast.ToIntE, fromVal, val)
		case int8:
			return bindCasted[int8](cast.ToInt8E, fromVal, val)
		case int16:
			return bindCasted[int16](cast.ToInt16E, fromVal, val)
		case int32:
			return bindCasted[int32](cast.ToInt32E, fromVal, val)
		case int64:
			return bindCasted[int64](cast.ToInt64E, fromVal, val)
		case uint:
			return bindCasted[uint](cast.ToUintE, fromVal, val)
		case uint8:
			return bindCasted[uint8](cast.ToUint8E, fromVal, val)
		case uint16:
			return bindCasted[uint16](cast.ToUint16E, fromVal, val)
		case uint32:
			return bindCasted[uint32](cast.ToUint32E, fromVal, val)
		case uint64:
			return bindCasted[uint64](cast.ToUint64E, fromVal, val)
		case float32:
			return bindCasted[float32](cast.ToFloat32E, fromVal, val)
		case float64:
			return bindCasted[float64](cast.ToFloat64E, fromVal, val)
		case []int:
			return bindCasted[[]int](cast.ToIntSliceE, fromVal, val)
		case []string:
			return bindCasted[[]string](cast.ToStringSliceE, fromVal, val)
		case time.Time:
			return bindCasted[time.Time](cast.ToTimeE, fromVal, val)
		case time.Duration:
			return bindCasted[time.Duration](cast.ToDurationE, fromVal, val)
		}
		return nil
	}

	// Next, we check if to implements Valuer.
	ok, err := func() (bool, error) {
		switch fromVal := fromVal.(type) {
		case bool:
			return bindValuer(fromVal, val)
		case int64:
			return bindValuer(fromVal, val)
		case float64:
			return bindValuer(fromVal, val)
		case string:
			return bindValuer(fromVal, val)
		case neo4j.Point2D:
			return bindValuer(fromVal, val)
		case neo4j.Point3D:
			return bindValuer(fromVal, val)
		case neo4j.Date:
			return bindValuer(fromVal, val)
		case neo4j.LocalTime:
			return bindValuer(fromVal, val)
		case neo4j.LocalDateTime:
			return bindValuer(fromVal, val)
		case neo4j.Time:
			return bindValuer(fromVal, val)
		case neo4j.Duration:
			return bindValuer(fromVal, val)
		case time.Time:
			return bindValuer(fromVal, val)
		case []byte:
			return bindValuer(fromVal, val)
		case []any:
			return bindValuer(fromVal, val)
		case map[string]any:
			return bindValuer(fromVal, val)
		case neo4j.Node:
			// ptrToValDO: Support abstract nodes
			err := bindValue(fromVal.Props, val)
			if err != nil {
				return false, err
			}
			return true, nil
		case neo4j.Relationship:
			err := bindValue(fromVal.Props, val)
			if err != nil {
				return false, err
			}
			return true, nil
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
	bytes, err := json.Marshal(fromVal)
	if err != nil {
		return err
	}
	err = json.Unmarshal(bytes, to.Interface())
	if err != nil {
		return err
	}
	return nil
}
