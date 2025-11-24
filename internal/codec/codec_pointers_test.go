package codec_test

import (
	"testing"

	"github.com/rlch/neogo/internal/codec"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestPointerFields tests pointer fields with dereferencing
func TestPointerFields(t *testing.T) {
	registry := codec.NewCodecRegistry()
	registry.RegisterTypes(WithPointers{})

	str := "hello"
	i := 42
	f := 3.14
	b := true

	original := WithPointers{
		PtrString: &str,
		PtrInt:    &i,
		PtrFloat:  &f,
		PtrBool:   &b,
	}

	// Encode
	encoded, err := registry.Encode(&original)
	require.NoError(t, err)

	assert.Equal(t, "hello", encoded["ptr_string"])
	assert.Equal(t, int64(42), encoded["ptr_int"])
	assert.InDelta(t, 3.14, encoded["ptr_float"], 0.01)
	assert.Equal(t, true, encoded["ptr_bool"])

	// Decode
	var decoded WithPointers
	err = registry.Decode(encoded, &decoded)
	require.NoError(t, err)

	assert.Equal(t, *original.PtrString, *decoded.PtrString)
	assert.Equal(t, *original.PtrInt, *decoded.PtrInt)
	assert.InDelta(t, *original.PtrFloat, *decoded.PtrFloat, 0.01)
	assert.Equal(t, *original.PtrBool, *decoded.PtrBool)
}

// TestNilValues tests nil pointers and nil slices
func TestNilValues(t *testing.T) {
	registry := codec.NewCodecRegistry()
	registry.RegisterTypes(NilableTypes{}, NestedStruct{})

	original := NilableTypes{
		PtrString: nil,
		PtrInt:    nil,
		SliceVal:  nil,
		NestedPtr: nil,
	}

	// Encode
	encoded, err := registry.Encode(&original)
	require.NoError(t, err)

	// Verify nils
	assert.Nil(t, encoded["ptr_string"])
	assert.Nil(t, encoded["ptr_int"])
	assert.Nil(t, encoded["slice_val"])
	assert.Nil(t, encoded["nested_ptr"])

	// Decode
	var decoded NilableTypes
	err = registry.Decode(encoded, &decoded)
	require.NoError(t, err)

	assert.Nil(t, decoded.PtrString)
	assert.Nil(t, decoded.PtrInt)
	assert.Nil(t, decoded.SliceVal)
	assert.Nil(t, decoded.NestedPtr)
}

// TestPointerToStruct tests pointer to struct encoding
func TestPointerToStruct(t *testing.T) {
	registry := codec.NewCodecRegistry()
	registry.RegisterTypes(NestedStruct{})
	registry.RegisterTypes(Wrapper{})

	inner := &NestedStruct{Val: "inner"}
	original := Wrapper{Nested: inner}

	encoded, err := registry.Encode(&original)
	require.NoError(t, err)

	nested := encoded["nested"].(map[string]any)
	assert.Equal(t, "inner", nested["val"])

	var decoded Wrapper
	err = registry.Decode(encoded, &decoded)
	require.NoError(t, err)

	assert.Equal(t, original.Nested.Val, decoded.Nested.Val)
}

// TestSliceOfPointers tests slices of pointers
func TestSliceOfPointers(t *testing.T) {
	registry := codec.NewCodecRegistry()
	registry.RegisterTypes(NestedStruct{})
	registry.RegisterTypes(Container{})

	original := Container{
		Items: []*NestedStruct{
			{Val: "first"},
			{Val: "second"},
		},
	}

	encoded, err := registry.Encode(&original)
	require.NoError(t, err)

	items := encoded["items"].([]any)
	assert.Len(t, items, 2)
	assert.Equal(t, "first", items[0].(map[string]any)["val"])

	var decoded Container
	err = registry.Decode(encoded, &decoded)
	require.NoError(t, err)

	assert.Len(t, decoded.Items, 2)
	assert.Equal(t, original.Items[0].Val, decoded.Items[0].Val)
}
