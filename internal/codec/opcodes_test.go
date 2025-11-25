package codec_test

import (
	"testing"
	"time"

	"github.com/neo4j/neo4j-go-driver/v5/neo4j"
	"github.com/rlch/neogo/internal/codec"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestOpcodes tests opcode execution for encoding
func TestOpcodes(t *testing.T) {
	t.Run("RoundTrip", func(t *testing.T) {
		registry := codec.NewCodecRegistry()
		registry.RegisterTypes(TestStruct{}, NestedStruct{})

		ptrInt := 42
		now := time.Now().Truncate(time.Millisecond)
		point := neo4j.Point2D{X: 1.5, Y: 2.5, SpatialRefId: 4326}

		original := TestStruct{
			Int:        123,
			String:     "hello",
			Bool:       true,
			Float:      3.14,
			PtrInt:     &ptrInt,
			SliceInt:   []int{1, 2, 3},
			SliceStr:   []string{"a", "b"},
			Nested:     NestedStruct{Val: "nested_val"},
			PtrNested:  &NestedStruct{Val: "ptr_nested_val"},
			EmbeddedStruct: EmbeddedStruct{EmbedVal: 999},
			Time:  now,
			Point: point,
		}

		encoded, err := registry.Encode(&original)
		require.NoError(t, err)

		assert.Equal(t, int64(123), encoded["int"])
		assert.Equal(t, "hello", encoded["string"])
		assert.Equal(t, int64(999), encoded["embed_val"])

		var decoded TestStruct
		err = registry.Decode(encoded, &decoded)
		require.NoError(t, err)

		assert.Equal(t, original.Int, decoded.Int)
		assert.Equal(t, original.EmbeddedStruct, decoded.EmbeddedStruct)
	})

	t.Run("BoolFields", func(t *testing.T) {
		registry := codec.NewCodecRegistry()
		registry.RegisterTypes(WithBools{})

		original := WithBools{True: true, False: false}

		encoded, err := registry.Encode(&original)
		require.NoError(t, err)

		assert.Equal(t, true, encoded["true"])
		assert.Equal(t, false, encoded["false"])

		var decoded WithBools
		err = registry.Decode(encoded, &decoded)
		require.NoError(t, err)

		assert.Equal(t, original.True, decoded.True)
		assert.Equal(t, original.False, decoded.False)
	})

	t.Run("NumericBoundaries", func(t *testing.T) {
		registry := codec.NewCodecRegistry()
		registry.RegisterTypes(Boundaries{})

		original := Boundaries{
			MaxInt64: 9223372036854775807,
			MinInt64: -9223372036854775808,
		}

		encoded, err := registry.Encode(&original)
		require.NoError(t, err)

		var decoded Boundaries
		err = registry.Decode(encoded, &decoded)
		require.NoError(t, err)

		assert.Equal(t, original.MaxInt64, decoded.MaxInt64)
		assert.Equal(t, original.MinInt64, decoded.MinInt64)
	})

	t.Run("Neo4jTypes", func(t *testing.T) {
		registry := codec.NewCodecRegistry()
		registry.RegisterTypes(AllNeo4jTypes{})

		original := AllNeo4jTypes{
			Point: neo4j.Point2D{X: 1.5, Y: 2.5, SpatialRefId: 4326},
		}

		encoded, err := registry.Encode(&original)
		require.NoError(t, err)

		point := encoded["point"].(map[string]any)
		assert.InDelta(t, 1.5, point["x"], 0.01)

		var decoded AllNeo4jTypes
		err = registry.Decode(encoded, &decoded)
		require.NoError(t, err)

		assert.InDelta(t, original.Point.X, decoded.Point.X, 0.01)
	})

	t.Run("InterfaceFields", func(t *testing.T) {
		registry := codec.NewCodecRegistry()
		registry.RegisterTypes(WithInterface{})

		original := WithInterface{Data: "test string"}

		encoded, err := registry.Encode(&original)
		require.NoError(t, err)

		var decoded WithInterface
		err = registry.Decode(encoded, &decoded)
		require.NoError(t, err)

		assert.Equal(t, original.Data, decoded.Data)
	})

	t.Run("SliceOfStructs", func(t *testing.T) {
		registry := codec.NewCodecRegistry()
		registry.RegisterTypes(NestedStruct{})

		input := []NestedStruct{{Val: "1"}, {Val: "2"}}

		encoded, err := registry.EncodeValue(input)
		require.NoError(t, err)

		sliceEnc := encoded.([]any)
		assert.Len(t, sliceEnc, 2)

		var decoded []NestedStruct
		err = registry.Decode(sliceEnc, &decoded)
		require.NoError(t, err)

		assert.Equal(t, input, decoded)
	})

	t.Run("SlicePrimitives", func(t *testing.T) {
		registry := codec.NewCodecRegistry()
		registry.RegisterTypes(SliceContainer{})

		original := SliceContainer{
			Ints:    []int{-1, 0, 1, 42, 999},
			Strings: []string{"a", "b", "c"},
		}

		encoded, err := registry.Encode(&original)
		require.NoError(t, err)

		var decoded SliceContainer
		err = registry.Decode(encoded, &decoded)
		require.NoError(t, err)

		assert.Equal(t, original, decoded)
	})

	t.Run("ComplexNestedWithAllTypes", func(t *testing.T) {
		registry := codec.NewCodecRegistry()
		registry.RegisterTypes(ComplexStruct{}, NestedStruct{}, Status{})

		nested := NestedStruct{Val: "nested"}
		
		original := ComplexStruct{
			Simple:    99,
			Nested:    NestedStruct{Val: "direct"},
			PtrNested: &nested,
			SliceNest: []NestedStruct{{Val: "s1"}, {Val: "s2"}},
			Status:    Status{Active: false},
		}

		encoded, err := registry.Encode(&original)
		require.NoError(t, err)

		assert.Equal(t, int64(99), encoded["simple"])
		assert.Equal(t, false, encoded["active"])

		var decoded ComplexStruct
		err = registry.Decode(encoded, &decoded)
		require.NoError(t, err)

		assert.Equal(t, original.Simple, decoded.Simple)
		assert.Equal(t, original.Active, decoded.Active)
		assert.Len(t, decoded.SliceNest, 2)
	})

	t.Run("AllNilPointersEncoding", func(t *testing.T) {
		registry := codec.NewCodecRegistry()
		registry.RegisterTypes(AllNilPtrs{}, NestedStruct{})

		original := AllNilPtrs{
			PtrStr: nil,
			PtrInt: nil,
			PtrFlt: nil,
			PtrBol: nil,
			PtrNst: nil,
		}

		encoded, err := registry.Encode(&original)
		require.NoError(t, err)

		assert.Nil(t, encoded["ptr_str"])
		assert.Nil(t, encoded["ptr_int"])
		assert.Nil(t, encoded["ptr_flt"])

		var decoded AllNilPtrs
		err = registry.Decode(encoded, &decoded)
		require.NoError(t, err)

		assert.Nil(t, decoded.PtrStr)
		assert.Nil(t, decoded.PtrInt)
		assert.Nil(t, decoded.PtrFlt)
	})

	t.Run("EmptySlicesEncoding", func(t *testing.T) {
		registry := codec.NewCodecRegistry()
		registry.RegisterTypes(SliceContainer{})

		// Empty slices: test actual behavior
		original := SliceContainer{
			Ints:    []int{},
			Strings: []string{},
		}

		encoded, err := registry.Encode(&original)
		require.NoError(t, err)

		// Empty slices may encode as nil or empty []any
		// What matters is that decode works correctly
		var decoded SliceContainer
		err = registry.Decode(encoded, &decoded)
		require.NoError(t, err)

		// The important part: decoding works correctly
		assert.Len(t, decoded.Ints, 0)
		assert.Len(t, decoded.Strings, 0)
	})

	t.Run("MaxMinIntValues", func(t *testing.T) {
		registry := codec.NewCodecRegistry()
		registry.RegisterTypes(Boundaries{})

		original := Boundaries{
			MaxInt64:   9223372036854775807,  // math.MaxInt64
			MinInt64:   -9223372036854775808, // math.MinInt64
			MaxFloat64: 1.7976931348623157e+308,
		}

		encoded, err := registry.Encode(&original)
		require.NoError(t, err)

		var decoded Boundaries
		err = registry.Decode(encoded, &decoded)
		require.NoError(t, err)

		assert.Equal(t, original.MaxInt64, decoded.MaxInt64)
		assert.Equal(t, original.MinInt64, decoded.MinInt64)
		assert.InDelta(t, original.MaxFloat64, decoded.MaxFloat64, 1e+298)
	})
}
