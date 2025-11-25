package codec_test

import (
	"reflect"
	"testing"

	"github.com/rlch/neogo/internal/codec"
	"github.com/stretchr/testify/assert"
)

// TestGetTypePtr tests the getTypePtr function
func TestGetTypePtr(t *testing.T) {
	t.Run("SameTypeReturnsConsistentPtr", func(t *testing.T) {
		typ1 := reflect.TypeOf(TestStruct{})
		typ2 := reflect.TypeOf(TestStruct{})
		
		ptr1 := codec.GetTypePtr(typ1)
		ptr2 := codec.GetTypePtr(typ2)
		
		assert.Equal(t, ptr1, ptr2, "same type should return same pointer")
	})

	t.Run("DifferentTypesReturnDifferentPtrs", func(t *testing.T) {
		typ1 := reflect.TypeOf(TestStruct{})
		typ2 := reflect.TypeOf(NestedStruct{})
		
		ptr1 := codec.GetTypePtr(typ1)
		ptr2 := codec.GetTypePtr(typ2)
		
		assert.NotEqual(t, ptr1, ptr2, "different types should return different pointers")
	})

	t.Run("PrimitiveTypes", func(t *testing.T) {
		int1 := codec.GetTypePtr(reflect.TypeOf(int(0)))
		int2 := codec.GetTypePtr(reflect.TypeOf(int(0)))
		str1 := codec.GetTypePtr(reflect.TypeOf(""))
		
		assert.Equal(t, int1, int2)
		assert.NotEqual(t, int1, str1)
	})

	t.Run("PointerAndValueTypesAreDifferent", func(t *testing.T) {
		val := reflect.TypeOf(TestStruct{})
		ptr := reflect.TypeOf((*TestStruct)(nil))
		
		ptrVal := codec.GetTypePtr(val)
		ptrPtr := codec.GetTypePtr(ptr)
		
		assert.NotEqual(t, ptrVal, ptrPtr)
	})
}

// TestGetTypePtrFromValue tests the getTypePtrFromValue function
func TestGetTypePtrFromValue(t *testing.T) {
	t.Run("ValueAndPointerReturnSameTypePtr", func(t *testing.T) {
		val := TestStruct{Int: 42}
		ptr := &TestStruct{Int: 42}
		
		ptrVal := codec.GetTypePtrFromValue(val)
		ptrPtr := codec.GetTypePtrFromValue(ptr)
		
		assert.Equal(t, ptrVal, ptrPtr, "value and pointer to same type should have same typeptr")
	})

	t.Run("DifferentValuesOfSameType", func(t *testing.T) {
		val1 := TestStruct{Int: 1}
		val2 := TestStruct{Int: 999}
		
		ptr1 := codec.GetTypePtrFromValue(val1)
		ptr2 := codec.GetTypePtrFromValue(val2)
		
		assert.Equal(t, ptr1, ptr2, "different values of same type should have same typeptr")
	})

	t.Run("NilPointer", func(t *testing.T) {
		var ptr *TestStruct
		typePtr := codec.GetTypePtrFromValue(ptr)
		
		val := TestStruct{}
		typePtr2 := codec.GetTypePtrFromValue(val)
		
		assert.Equal(t, typePtr, typePtr2)
	})

	t.Run("InterfaceValues", func(t *testing.T) {
		var iface interface{} = TestStruct{Int: 42}
		typePtr := codec.GetTypePtrFromValue(iface)
		
		val := TestStruct{}
		typePtr2 := codec.GetTypePtrFromValue(val)
		
		assert.Equal(t, typePtr, typePtr2)
	})

	t.Run("PrimitiveTypes", func(t *testing.T) {
		ptrInt := codec.GetTypePtrFromValue(42)
		ptrInt2 := codec.GetTypePtrFromValue(99)
		ptrStr := codec.GetTypePtrFromValue("hello")
		
		assert.Equal(t, ptrInt, ptrInt2)
		assert.NotEqual(t, ptrInt, ptrStr)
	})
}

// TestGetValuePtr tests the getValuePtr function
func TestGetValuePtr(t *testing.T) {
	t.Run("StructPointer", func(t *testing.T) {
		s := TestStruct{Int: 42}
		ptr := codec.GetValuePtr(s)
		
		// Dereference and check value
		deref := (*TestStruct)(ptr)
		assert.Equal(t, s.Int, deref.Int)
	})

	t.Run("NilPointer", func(t *testing.T) {
		var p *TestStruct
		ptr := codec.GetValuePtr(p)
		
		assert.Nil(t, ptr)
	})

	t.Run("String", func(t *testing.T) {
		s := "hello"
		ptr := codec.GetValuePtr(s)
		
		// Should point to string data
		assert.NotNil(t, ptr)
	})

	t.Run("Int", func(t *testing.T) {
		val := 42
		ptr := codec.GetValuePtr(val)
		
		deref := *(*int)(ptr)
		assert.Equal(t, val, deref)
	})

	t.Run("Slice", func(t *testing.T) {
		slice := []int{1, 2, 3}
		ptr := codec.GetValuePtr(slice)
		
		assert.NotNil(t, ptr)
	})
}

// Note: GetInterfaceValue is designed for reading interface{} values from struct fields,
// not for standalone interface{} values. The unsafe memory access is only safe when
// the pointer points to a valid interface{} location in a struct's memory layout.
// Testing it directly with arbitrary interface{} values can cause segfaults.
// The functionality is implicitly tested via OpFieldInterface encoding tests.

// TestGetKind tests the getKind function
func TestGetKind(t *testing.T) {
	t.Run("BoolKind", func(t *testing.T) {
		kind := codec.GetKind(reflect.Bool)
		assert.Equal(t, codec.KindBool, kind)
	})

	t.Run("IntKinds", func(t *testing.T) {
		tests := []struct {
			rk       reflect.Kind
			expected codec.Kind
		}{
			{reflect.Int, codec.KindInt},
			{reflect.Int8, codec.KindInt8},
			{reflect.Int16, codec.KindInt16},
			{reflect.Int32, codec.KindInt32},
			{reflect.Int64, codec.KindInt64},
		}

		for _, tt := range tests {
			kind := codec.GetKind(tt.rk)
			assert.Equal(t, tt.expected, kind)
		}
	})

	t.Run("UintKinds", func(t *testing.T) {
		tests := []struct {
			rk       reflect.Kind
			expected codec.Kind
		}{
			{reflect.Uint, codec.KindUint},
			{reflect.Uint8, codec.KindUint8},
			{reflect.Uint16, codec.KindUint16},
			{reflect.Uint32, codec.KindUint32},
			{reflect.Uint64, codec.KindUint64},
		}

		for _, tt := range tests {
			kind := codec.GetKind(tt.rk)
			assert.Equal(t, tt.expected, kind)
		}
	})

	t.Run("FloatKinds", func(t *testing.T) {
		tests := []struct {
			rk       reflect.Kind
			expected codec.Kind
		}{
			{reflect.Float32, codec.KindFloat32},
			{reflect.Float64, codec.KindFloat64},
		}

		for _, tt := range tests {
			kind := codec.GetKind(tt.rk)
			assert.Equal(t, tt.expected, kind)
		}
	})

	t.Run("StringKind", func(t *testing.T) {
		kind := codec.GetKind(reflect.String)
		assert.Equal(t, codec.KindString, kind)
	})

	t.Run("SliceKind", func(t *testing.T) {
		kind := codec.GetKind(reflect.Slice)
		assert.Equal(t, codec.KindSlice, kind)
	})

	t.Run("MapKind", func(t *testing.T) {
		kind := codec.GetKind(reflect.Map)
		assert.Equal(t, codec.KindMap, kind)
	})

	t.Run("StructKind", func(t *testing.T) {
		kind := codec.GetKind(reflect.Struct)
		assert.Equal(t, codec.KindStruct, kind)
	})

	t.Run("InterfaceKind", func(t *testing.T) {
		kind := codec.GetKind(reflect.Interface)
		assert.Equal(t, codec.KindInterface, kind)
	})

	t.Run("PtrKind", func(t *testing.T) {
		kind := codec.GetKind(reflect.Ptr)
		assert.Equal(t, codec.KindPtr, kind)
	})

	t.Run("UnknownKindDefaultsToInterface", func(t *testing.T) {
		kind := codec.GetKind(reflect.Chan)
		assert.Equal(t, codec.KindInterface, kind)
	})
}

// TestConvertToInt tests type conversion to int
func TestConvertToInt(t *testing.T) {
	tests := []struct {
		name    string
		input   interface{}
		want    int
		ok      bool
	}{
		{"int", 42, 42, true},
		{"int64", int64(42), 42, true},
		{"float64", 42.0, 42, true},
		{"float64 truncates", 42.9, 42, true},
		{"string fails", "42", 0, false},
		{"bool fails", true, 0, false},
		{"nil fails", nil, 0, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := codec.ConvertToInt(tt.input)
			assert.Equal(t, tt.ok, ok)
			if ok {
				assert.Equal(t, tt.want, got)
			}
		})
	}
}

// TestConvertToInt8 tests type conversion to int8
func TestConvertToInt8(t *testing.T) {
	tests := []struct {
		name    string
		input   interface{}
		want    int8
		ok      bool
	}{
		{"int8", int8(42), 42, true},
		{"int", 42, 42, true},
		{"int64", int64(42), 42, true},
		{"float64", 42.0, 42, true},
		{"overflow", 200, int8(200 - 256), true}, // wraps
		{"string fails", "42", 0, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := codec.ConvertToInt8(tt.input)
			assert.Equal(t, tt.ok, ok)
			if ok {
				assert.Equal(t, tt.want, got)
			}
		})
	}
}

// TestConvertToInt16 tests type conversion to int16
func TestConvertToInt16(t *testing.T) {
	tests := []struct {
		name  string
		input interface{}
		want  int16
		ok    bool
	}{
		{"int16", int16(1000), 1000, true},
		{"int", 1000, 1000, true},
		{"int64", int64(1000), 1000, true},
		{"float64", 1000.0, 1000, true},
		{"string fails", "1000", 0, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := codec.ConvertToInt16(tt.input)
			assert.Equal(t, tt.ok, ok)
			if ok {
				assert.Equal(t, tt.want, got)
			}
		})
	}
}

// TestConvertToInt32 tests type conversion to int32
func TestConvertToInt32(t *testing.T) {
	tests := []struct {
		name  string
		input interface{}
		want  int32
		ok    bool
	}{
		{"int32", int32(100000), 100000, true},
		{"int", 100000, 100000, true},
		{"int64", int64(100000), 100000, true},
		{"float64", 100000.0, 100000, true},
		{"string fails", "100000", 0, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := codec.ConvertToInt32(tt.input)
			assert.Equal(t, tt.ok, ok)
			if ok {
				assert.Equal(t, tt.want, got)
			}
		})
	}
}

// TestConvertToInt64 tests type conversion to int64
func TestConvertToInt64(t *testing.T) {
	tests := []struct {
		name  string
		input interface{}
		want  int64
		ok    bool
	}{
		{"int64", int64(9223372036854775807), 9223372036854775807, true},
		{"int", 42, 42, true},
		{"float64", 42.0, 42, true},
		{"string fails", "42", 0, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := codec.ConvertToInt64(tt.input)
			assert.Equal(t, tt.ok, ok)
			if ok {
				assert.Equal(t, tt.want, got)
			}
		})
	}
}

// TestConvertToUint tests type conversion to uint
func TestConvertToUint(t *testing.T) {
	tests := []struct {
		name  string
		input interface{}
		want  uint
		ok    bool
	}{
		{"uint", uint(42), 42, true},
		{"uint64", uint64(42), 42, true},
		{"int", 42, 42, true},
		{"int64", int64(42), 42, true},
		{"float64", 42.0, 42, true},
		{"string fails", "42", 0, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := codec.ConvertToUint(tt.input)
			assert.Equal(t, tt.ok, ok)
			if ok {
				assert.Equal(t, tt.want, got)
			}
		})
	}
}

// TestConvertToUint8 tests type conversion to uint8
func TestConvertToUint8(t *testing.T) {
	tests := []struct {
		name  string
		input interface{}
		want  uint8
		ok    bool
	}{
		{"uint8", uint8(200), 200, true},
		{"uint", uint(200), 200, true},
		{"uint64", uint64(200), 200, true},
		{"int", 200, 200, true},
		{"int64", int64(200), 200, true},
		{"float64", 200.0, 200, true},
		{"string fails", "200", 0, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := codec.ConvertToUint8(tt.input)
			assert.Equal(t, tt.ok, ok)
			if ok {
				assert.Equal(t, tt.want, got)
			}
		})
	}
}

// TestConvertToUint16 tests type conversion to uint16
func TestConvertToUint16(t *testing.T) {
	tests := []struct {
		name  string
		input interface{}
		want  uint16
		ok    bool
	}{
		{"uint16", uint16(50000), 50000, true},
		{"uint", uint(50000), 50000, true},
		{"uint64", uint64(50000), 50000, true},
		{"int", 50000, 50000, true},
		{"int64", int64(50000), 50000, true},
		{"float64", 50000.0, 50000, true},
		{"string fails", "50000", 0, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := codec.ConvertToUint16(tt.input)
			assert.Equal(t, tt.ok, ok)
			if ok {
				assert.Equal(t, tt.want, got)
			}
		})
	}
}

// TestConvertToUint32 tests type conversion to uint32
func TestConvertToUint32(t *testing.T) {
	tests := []struct {
		name  string
		input interface{}
		want  uint32
		ok    bool
	}{
		{"uint32", uint32(4000000000), 4000000000, true},
		{"uint", uint(4000000000), 4000000000, true},
		{"uint64", uint64(4000000000), 4000000000, true},
		{"int", 1000, 1000, true},
		{"int64", int64(1000), 1000, true},
		{"float64", 1000.0, 1000, true},
		{"string fails", "1000", 0, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := codec.ConvertToUint32(tt.input)
			assert.Equal(t, tt.ok, ok)
			if ok {
				assert.Equal(t, tt.want, got)
			}
		})
	}
}

// TestConvertToUint64 tests type conversion to uint64
func TestConvertToUint64(t *testing.T) {
	tests := []struct {
		name  string
		input interface{}
		want  uint64
		ok    bool
	}{
		{"uint64", uint64(18446744073709551615), 18446744073709551615, true},
		{"uint", uint(42), 42, true},
		{"int", 42, 42, true},
		{"int64", int64(42), 42, true},
		{"float64", 42.0, 42, true},
		{"string fails", "42", 0, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := codec.ConvertToUint64(tt.input)
			assert.Equal(t, tt.ok, ok)
			if ok {
				assert.Equal(t, tt.want, got)
			}
		})
	}
}

// TestConvertToFloat32 tests type conversion to float32
func TestConvertToFloat32(t *testing.T) {
	tests := []struct {
		name  string
		input interface{}
		want  float32
		ok    bool
	}{
		{"float32", float32(3.14), 3.14, true},
		{"float64", 3.14, float32(3.14), true},
		{"int", 42, 42.0, true},
		{"int64", int64(42), 42.0, true},
		{"string fails", "3.14", 0, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := codec.ConvertToFloat32(tt.input)
			assert.Equal(t, tt.ok, ok)
			if ok {
				assert.InDelta(t, tt.want, got, 0.01)
			}
		})
	}
}

// TestConvertToFloat64 tests type conversion to float64
func TestConvertToFloat64(t *testing.T) {
	tests := []struct {
		name  string
		input interface{}
		want  float64
		ok    bool
	}{
		{"float64", 3.14159, 3.14159, true},
		{"float32", float32(3.14), float64(float32(3.14)), true},
		{"int", 42, 42.0, true},
		{"int64", int64(42), 42.0, true},
		{"string fails", "3.14", 0, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := codec.ConvertToFloat64(tt.input)
			assert.Equal(t, tt.ok, ok)
			if ok {
				assert.InDelta(t, tt.want, got, 0.0001)
			}
		})
	}
}

// TestExtractFieldMeta tests extractFieldMeta function
func TestExtractFieldMeta(t *testing.T) {
	t.Run("PrimitiveFieldMeta", func(t *testing.T) {
		typ := reflect.TypeOf(TestStruct{})
		intField, _ := typ.FieldByName("Int")

		meta := codec.ExtractFieldMeta(intField)

		assert.Equal(t, intField.Offset, meta.Offset)
		assert.Equal(t, intField.Type.Size(), meta.Size)
		assert.Equal(t, codec.KindInt, meta.Kind)
		assert.Equal(t, intField.Type, meta.Type)
	})

	t.Run("StringFieldMeta", func(t *testing.T) {
		typ := reflect.TypeOf(TestStruct{})
		strField, _ := typ.FieldByName("String")

		meta := codec.ExtractFieldMeta(strField)

		assert.Equal(t, strField.Offset, meta.Offset)
		assert.Equal(t, strField.Type.Size(), meta.Size)
		assert.Equal(t, codec.KindString, meta.Kind)
	})

	t.Run("SliceFieldMeta", func(t *testing.T) {
		typ := reflect.TypeOf(TestStruct{})
		sliceField, _ := typ.FieldByName("SliceInt")

		meta := codec.ExtractFieldMeta(sliceField)

		assert.Equal(t, sliceField.Offset, meta.Offset)
		assert.Equal(t, sliceField.Type.Size(), meta.Size)
		// Note: []int is KindSlice, but []byte would be KindBytes
		// TestStruct.SliceInt is []int, so it's KindSlice
		assert.Equal(t, codec.KindSlice, meta.Kind)
	})

	t.Run("PointerFieldMeta", func(t *testing.T) {
		typ := reflect.TypeOf(TestStruct{})
		ptrField, _ := typ.FieldByName("PtrInt")

		meta := codec.ExtractFieldMeta(ptrField)

		assert.Equal(t, ptrField.Offset, meta.Offset)
		assert.Equal(t, codec.KindPtr, meta.Kind)
	})

	t.Run("StructFieldMeta", func(t *testing.T) {
		typ := reflect.TypeOf(TestStruct{})
		structField, _ := typ.FieldByName("Nested")

		meta := codec.ExtractFieldMeta(structField)

		assert.Equal(t, structField.Offset, meta.Offset)
		assert.Equal(t, codec.KindStruct, meta.Kind)
	})
}
