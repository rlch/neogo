package internal

import (
	"errors"
	"reflect"
	"testing"
)

// Helper function to assert visited fields
func assertFieldsVisited(
	t *testing.T,
	strct reflect.Value,
	expectedFields []string,
	shouldRecurse bool,
) {
	visitedFields := make(map[string]bool)
	err := WalkStruct(strct, func(
		i int,
		typ reflect.StructField,
		val reflect.Value,
	) (bool, error) {
		visitedFields[typ.Name] = true
		return shouldRecurse && val.Kind() == reflect.Struct, nil
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	for _, fieldName := range expectedFields {
		if !visitedFields[fieldName] {
			t.Fatalf("expected field %s to be visited", fieldName)
		}
	}
}

func TestWalkStruct_NonStruct(t *testing.T) {
	err := WalkStruct(reflect.ValueOf(42), func(
		i int,
		typ reflect.StructField,
		val reflect.Value,
	) (bool, error) {
		return false, nil
	})
	if err == nil || err.Error() != "expected struct, got int" {
		t.Fatalf("expected error for non-struct, got: %v", err)
	}
}

func TestWalkStruct_EmptyStruct(t *testing.T) {
	type Empty struct{}
	err := WalkStruct(reflect.ValueOf(Empty{}), func(
		i int,
		typ reflect.StructField,
		val reflect.Value,
	) (bool, error) {
		return false, errors.New("should not be called")
	})
	if err != nil {
		t.Fatalf("expected no error for empty struct, got: %v", err)
	}
}

func TestWalkStruct_PointerToStruct(t *testing.T) {
	type Simple struct {
		A int
		B string
	}
	s := &Simple{A: 10, B: "hello"}

	assertFieldsVisited(
		t,
		reflect.ValueOf(s),
		[]string{"A", "B"},
		false,
	)
}

func TestWalkStruct_AnonymousStruct(t *testing.T) {
	type (
		Inner struct {
			X int
		}
		Outer struct {
			Inner
			Y string
		}
	)
	o := Outer{Inner: Inner{X: 5}, Y: "world"}

	assertFieldsVisited(
		t,
		reflect.ValueOf(o),
		[]string{"X", "Y"},
		true,
	)
}

func TestWalkStruct_VaryingFieldTypes(t *testing.T) {
	type Mixed struct {
		A int
		B string
		C float64
	}
	m := Mixed{A: 42, B: "test", C: 3.14}

	assertFieldsVisited(
		t,
		reflect.ValueOf(m),
		[]string{"A", "B", "C"},
		false,
	)
}

func TestWalkStruct_NestedAnonymousStruct(t *testing.T) {
	type Level1 struct{ L1Field int }
	type Level2 struct{ Level1 }
	o := Level2{Level1{L1Field: 1}}

	assertFieldsVisited(
		t,
		reflect.ValueOf(o),
		[]string{"L1Field"},
		true,
	)
}

func TestUnwindValue(t *testing.T) {
	// Testing with multi-level pointer
	i := 42
	ptr := &i
	ptrToPtr := &ptr

	got := UnwindValue(reflect.ValueOf(ptrToPtr))
	if got.Kind() != reflect.Int || got.Int() != int64(i) {
		t.Errorf("UnwindValue failed, expected %d got %d", i, got.Int())
	}

	// Testing with non-pointer type
	got = UnwindValue(reflect.ValueOf(i))
	if got.Kind() != reflect.Int || got.Int() != int64(i) {
		t.Errorf("UnwindValue failed, expected %d for non-pointer type", i)
	}
}

func TestUnwindType(t *testing.T) {
	// Testing with multi-level pointer
	var i int
	pi := &i
	ptrToPtr := reflect.TypeOf(&pi)

	got := UnwindType(ptrToPtr)
	if got.Kind() != reflect.Int {
		t.Errorf("UnwindType failed, expected int got %s", got.Kind())
	}

	// Testing with non-pointer type
	got = UnwindType(reflect.TypeOf(i))
	if got.Kind() != reflect.Int {
		t.Errorf("UnwindType failed, expected int for non-pointer type")
	}
}
