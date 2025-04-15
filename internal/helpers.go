package internal

import (
	"fmt"
	"reflect"
)

func WalkStruct(
	v reflect.Value,
	visit func(
		i int,
		typ reflect.StructField,
		val reflect.Value,
	) (recurseInto bool, err error),
) (recurseInto error) {
	vt := v.Type()
	for vt.Kind() == reflect.Ptr {
		vt = vt.Elem()
		v = v.Elem()
	}
	if vt.Kind() != reflect.Struct || v.Kind() != reflect.Struct {
		return fmt.Errorf("expected struct, got %s", vt.Kind())
	}
	for i := 0; i < vt.NumField(); i++ {
		vtf := vt.Field(i)
		vf := v.Field(i)
		shouldRecurse, err := visit(i, vtf, vf)
		if err != nil {
			return err
		}
		if shouldRecurse && vf.Kind() == reflect.Struct {
			if err := WalkStruct(vf, visit); err != nil {
				return err
			}
			continue
		}
	}
	return nil
}

func UnwindValue(ptrTo reflect.Value) reflect.Value {
	for ptrTo.Kind() == reflect.Ptr {
		ptrTo = ptrTo.Elem()
	}
	return ptrTo
}

func UnwindType(ptrTo reflect.Type) reflect.Type {
	for ptrTo.Kind() == reflect.Ptr {
		ptrTo = ptrTo.Elem()
	}
	return ptrTo
}
