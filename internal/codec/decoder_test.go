package codec_test

import (
	"testing"

	"github.com/rlch/neogo/internal/codec"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestDecoder tests decoder implementations
func TestDecoder(t *testing.T) {
	t.Run("NilValues", func(t *testing.T) {
		registry := codec.NewCodecRegistry()
		registry.RegisterTypes(NilableTypes{}, NestedStruct{})

		original := NilableTypes{
			PtrString: nil,
			PtrInt:    nil,
			SliceVal:  nil,
		}

		encoded, err := registry.Encode(&original)
		require.NoError(t, err)
		assert.Nil(t, encoded["ptr_string"])

		var decoded NilableTypes
		err = registry.Decode(encoded, &decoded)
		require.NoError(t, err)
		assert.Nil(t, decoded.PtrString)
	})

	t.Run("EmptySlices", func(t *testing.T) {
		registry := codec.NewCodecRegistry()
		registry.RegisterTypes(SliceTypes{}, NestedStruct{})

		original := SliceTypes{
			SliceInt:    []int{},
			SliceString: []string{},
		}

		encoded, err := registry.Encode(&original)
		require.NoError(t, err)

		var decoded SliceTypes
		err = registry.Decode(encoded, &decoded)
		require.NoError(t, err)

		assert.Len(t, decoded.SliceInt, 0)
	})

	t.Run("PointerToStruct", func(t *testing.T) {
		registry := codec.NewCodecRegistry()
		registry.RegisterTypes(NestedStruct{}, Wrapper{})

		inner := &NestedStruct{Val: "inner"}
		original := Wrapper{Nested: inner}

		encoded, err := registry.Encode(&original)
		require.NoError(t, err)

		var decoded Wrapper
		err = registry.Decode(encoded, &decoded)
		require.NoError(t, err)

		assert.Equal(t, original.Nested.Val, decoded.Nested.Val)
	})

	t.Run("SliceOfPointers", func(t *testing.T) {
		registry := codec.NewCodecRegistry()
		registry.RegisterTypes(NestedStruct{}, Container{})

		original := Container{
			Items: []*NestedStruct{
				{Val: "first"},
				{Val: "second"},
			},
		}

		encoded, err := registry.Encode(&original)
		require.NoError(t, err)

		var decoded Container
		err = registry.Decode(encoded, &decoded)
		require.NoError(t, err)

		assert.Len(t, decoded.Items, 2)
		assert.Equal(t, original.Items[0].Val, decoded.Items[0].Val)
	})

	t.Run("DecodeErrors", func(t *testing.T) {
		registry := codec.NewCodecRegistry()
		registry.RegisterTypes(TestStruct{})

		invalid := map[string]any{
			"int": "not an int",
		}

		var result TestStruct
		err := registry.Decode(invalid, &result)
		assert.Error(t, err)
	})

	t.Run("AllNilPointers", func(t *testing.T) {
		registry := codec.NewCodecRegistry()
		registry.RegisterTypes(AllNilPtrs{}, NestedStruct{})

		original := AllNilPtrs{}

		encoded, err := registry.Encode(&original)
		require.NoError(t, err)

		var decoded AllNilPtrs
		err = registry.Decode(encoded, &decoded)
		require.NoError(t, err)

		assert.Nil(t, decoded.PtrStr)
		assert.Nil(t, decoded.PtrInt)
		assert.Nil(t, decoded.PtrFlt)
		assert.Nil(t, decoded.PtrBol)
		assert.Nil(t, decoded.PtrNst)
	})

	t.Run("TreeNodeRecursive", func(t *testing.T) {
		registry := codec.NewCodecRegistry()
		registry.RegisterTypes(TreeNode{})

		original := TreeNode{
			Value: 1,
			Children: []*TreeNode{
				{Value: 2, Children: nil},
				{Value: 3, Children: []*TreeNode{
					{Value: 4, Children: nil},
				}},
			},
		}

		encoded, err := registry.Encode(&original)
		require.NoError(t, err)

		var decoded TreeNode
		err = registry.Decode(encoded, &decoded)
		require.NoError(t, err)

		assert.Equal(t, original.Value, decoded.Value)
		assert.Len(t, decoded.Children, 2)
		assert.Equal(t, int64(1), encoded["value"]) // Root node value
	})

	t.Run("ComplexMixedStruct", func(t *testing.T) {
		registry := codec.NewCodecRegistry()
		registry.RegisterTypes(ComplexStruct{}, NestedStruct{}, Status{})

		original := ComplexStruct{
			Simple:    42,
			Nested:    NestedStruct{Val: "nested"},
			PtrNested: &NestedStruct{Val: "ptr_nested"},
			SliceNest: []NestedStruct{{Val: "slice1"}, {Val: "slice2"}},
			Status:    Status{Active: true},
		}

		encoded, err := registry.Encode(&original)
		require.NoError(t, err)

		var decoded ComplexStruct
		err = registry.Decode(encoded, &decoded)
		require.NoError(t, err)

		assert.Equal(t, original.Simple, decoded.Simple)
		assert.Equal(t, original.Nested.Val, decoded.Nested.Val)
		assert.NotNil(t, decoded.PtrNested)
		assert.Equal(t, original.PtrNested.Val, decoded.PtrNested.Val)
		assert.Equal(t, original.Active, decoded.Active)
	})

	t.Run("InterfaceFieldEdgeCases", func(t *testing.T) {
		registry := codec.NewCodecRegistry()
		registry.RegisterTypes(WithInterface{})

		// Test with map
		original := WithInterface{Data: map[string]any{"nested": "map"}}
		encoded, err := registry.Encode(&original)
		require.NoError(t, err)

		var decoded WithInterface
		err = registry.Decode(encoded, &decoded)
		require.NoError(t, err)

		m := decoded.Data.(map[string]any)
		assert.Equal(t, "map", m["nested"])
	})

	t.Run("TypeConversionErrors", func(t *testing.T) {
		registry := codec.NewCodecRegistry()
		registry.RegisterTypes(PrimitiveTypes{})

		// Try to decode a string into an int
		invalid := map[string]any{
			"int8": "not_a_number",
		}

		var result PrimitiveTypes
		err := registry.Decode(invalid, &result)
		assert.Error(t, err)
	})

	t.Run("InvalidBoolValue", func(t *testing.T) {
		registry := codec.NewCodecRegistry()
		registry.RegisterTypes(WithBools{})

		invalid := map[string]any{
			"true": "not_a_bool",
		}

		var result WithBools
		err := registry.Decode(invalid, &result)
		assert.Error(t, err)
	})

	t.Run("InvalidFloatValue", func(t *testing.T) {
		registry := codec.NewCodecRegistry()
		registry.RegisterTypes(FloatSpecial{})

		invalid := map[string]any{
			"normal_float": "not_a_float",
		}

		var result FloatSpecial
		err := registry.Decode(invalid, &result)
		assert.Error(t, err)
	})

	t.Run("ZeroValuesAllTypes", func(t *testing.T) {
		registry := codec.NewCodecRegistry()
		registry.RegisterTypes(PrimitiveTypes{})

		// All zero values
		original := PrimitiveTypes{}

		encoded, err := registry.Encode(&original)
		require.NoError(t, err)

		var decoded PrimitiveTypes
		err = registry.Decode(encoded, &decoded)
		require.NoError(t, err)

		assert.Equal(t, int8(0), decoded.Int8)
		assert.Equal(t, int16(0), decoded.Int16)
		assert.Equal(t, uint64(0), decoded.Uint64)
		assert.Nil(t, decoded.Bytes)
	})

	t.Run("MixedNilAndValues", func(t *testing.T) {
		registry := codec.NewCodecRegistry()
		registry.RegisterTypes(WithPointers{})

		i := 42
		f := 3.14
		original := WithPointers{
			PtrInt:   &i,
			PtrFloat: &f,
			PtrString: nil,
			PtrBool:  nil,
		}

		encoded, err := registry.Encode(&original)
		require.NoError(t, err)

		var decoded WithPointers
		err = registry.Decode(encoded, &decoded)
		require.NoError(t, err)

		require.NotNil(t, decoded.PtrInt)
		assert.Equal(t, 42, *decoded.PtrInt)
		require.NotNil(t, decoded.PtrFloat)
		assert.InDelta(t, 3.14, *decoded.PtrFloat, 0.01)
		assert.Nil(t, decoded.PtrString)
		assert.Nil(t, decoded.PtrBool)
	})
}
