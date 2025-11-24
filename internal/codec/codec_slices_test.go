package codec_test

import (
	"testing"

	"github.com/rlch/neogo/internal/codec"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestSliceOfStructs tests encoding/decoding slices of structs
func TestSliceOfStructs(t *testing.T) {
	registry := codec.NewCodecRegistry()
	registry.RegisterTypes(NestedStruct{})

	input := []NestedStruct{
		{Val: "1"},
		{Val: "2"},
	}

	// Encode value (slice)
	encoded, err := registry.EncodeValue(input)
	require.NoError(t, err)

	sliceEnc := encoded.([]any)
	assert.Len(t, sliceEnc, 2)
	assert.Equal(t, "1", sliceEnc[0].(map[string]any)["val"])

	var decoded []NestedStruct
	err = registry.Decode(sliceEnc, &decoded)
	require.NoError(t, err)
	assert.Equal(t, input, decoded)
}

// TestSlicePrimitives tests slices of primitive types
func TestSlicePrimitives(t *testing.T) {
	registry := codec.NewCodecRegistry()
	registry.RegisterTypes(SliceContainer{})

	original := SliceContainer{
		Ints:    []int{-1, 0, 1, 42, 999},
		Strings: []string{"a", "b", "c", "hello world"},
		Floats:  []float64{0.0, 3.14, -2.71, 1.41},
		Bools:   []bool{true, false, true},
	}

	// Encode
	encoded, err := registry.Encode(&original)
	require.NoError(t, err)

	ints := encoded["ints"].([]any)
	strings := encoded["strings"].([]any)
	floats := encoded["floats"].([]any)
	bools := encoded["bools"].([]any)

	assert.Equal(t, int64(-1), ints[0])
	assert.Equal(t, int64(999), ints[4])
	assert.Equal(t, "hello world", strings[3])
	assert.InDelta(t, 3.14, floats[1], 0.01)
	assert.Equal(t, false, bools[1])

	// Decode
	var decoded SliceContainer
	err = registry.Decode(encoded, &decoded)
	require.NoError(t, err)

	assert.Equal(t, original, decoded)
}

// TestEmptySlices tests empty slice encoding/decoding
func TestEmptySlices(t *testing.T) {
	registry := codec.NewCodecRegistry()
	registry.RegisterTypes(SliceTypes{}, NestedStruct{})

	original := SliceTypes{
		SliceInt:    []int{},
		SliceString: []string{},
		SliceNested: []NestedStruct{},
	}

	// Encode
	encoded, err := registry.Encode(&original)
	require.NoError(t, err)

	// Verify empty slices are encoded as empty slices (not nil)
	assert.Equal(t, []any{}, encoded["slice_int"])
	assert.Equal(t, []any{}, encoded["slice_string"])
	assert.Equal(t, []any{}, encoded["slice_nested"])

	// Decode
	var decoded SliceTypes
	err = registry.Decode(encoded, &decoded)
	require.NoError(t, err)

	assert.Len(t, decoded.SliceInt, 0)
	assert.Len(t, decoded.SliceString, 0)
	assert.Len(t, decoded.SliceNested, 0)
}

// TestEncodeValueSlices tests EncodeValue with slices
func TestEncodeValueSlices(t *testing.T) {
	registry := codec.NewCodecRegistry()

	// Slice of ints
	val, err := registry.EncodeValue([]int{1, 2, 3})
	require.NoError(t, err)
	assert.Equal(t, []any{int64(1), int64(2), int64(3)}, val)

	// Slice of strings
	val, err = registry.EncodeValue([]string{"a", "b"})
	require.NoError(t, err)
	assert.Equal(t, []any{"a", "b"}, val)
}
