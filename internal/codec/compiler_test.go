package codec_test

import (
	"testing"
	"time"

	"github.com/rlch/neogo/internal/codec"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestCompiler tests the Compiler for encoding and decoding
func TestCompiler(t *testing.T) {
	t.Run("PrimitiveTypes", func(t *testing.T) {
		registry := codec.NewCodecRegistry()
		registry.RegisterTypes(PrimitiveTypes{})

		original := PrimitiveTypes{
			Int8:    -8,
			Int16:   -16,
			Int64:   -64,
			Uint64:  64,
			Float32: 3.14,
			Bytes:   []byte("hello"),
		}

		encoded, err := registry.Encode(&original)
		require.NoError(t, err)
		assert.Equal(t, int64(-8), encoded["int8"])

		var decoded PrimitiveTypes
		err = registry.Decode(encoded, &decoded)
		require.NoError(t, err)
		assert.Equal(t, original.Int8, decoded.Int8)
	})

	t.Run("TypeCoercion", func(t *testing.T) {
		registry := codec.NewCodecRegistry()
		registry.RegisterTypes(IntTypes{})

		original := IntTypes{
			Int:   -100,
			Int64: 9223372036854775807,
		}

		encoded, err := registry.Encode(&original)
		require.NoError(t, err)

		var decoded IntTypes
		err = registry.Decode(encoded, &decoded)
		require.NoError(t, err)

		assert.Equal(t, original.Int, decoded.Int)
		assert.Equal(t, original.Int64, decoded.Int64)
	})

	t.Run("NestedStructs", func(t *testing.T) {
		registry := codec.NewCodecRegistry()
		registry.RegisterTypes(Company{}, Address{})

		original := Company{
			Name: "Tech Corp",
			Address: Address{
				Street: "123 Main St",
				City:   "New York",
			},
		}

		encoded, err := registry.Encode(&original)
		require.NoError(t, err)

		var decoded Company
		err = registry.Decode(encoded, &decoded)
		require.NoError(t, err)

		assert.Equal(t, original, decoded)
	})

	t.Run("DeeplyNestedStructs", func(t *testing.T) {
		registry := codec.NewCodecRegistry()
		registry.RegisterTypes(Level1{}, Level2{}, Level3{})

		original := Level1{
			ID: 1,
			Level2: Level2{
				Name:   "L2",
				Level3: Level3{Value: "L3"},
			},
		}

		encoded, err := registry.Encode(&original)
		require.NoError(t, err)

		var decoded Level1
		err = registry.Decode(encoded, &decoded)
		require.NoError(t, err)

		assert.Equal(t, original, decoded)
	})

	t.Run("EmbeddedStructs", func(t *testing.T) {
		registry := codec.NewCodecRegistry()
		registry.RegisterTypes(MultiEmbed{}, Status{})

		original := MultiEmbed{
			Name:   "Test",
			Status: Status{Active: true},
		}

		encoded, err := registry.Encode(&original)
		require.NoError(t, err)
		assert.Equal(t, true, encoded["active"])

		var decoded MultiEmbed
		err = registry.Decode(encoded, &decoded)
		require.NoError(t, err)

		assert.Equal(t, original.Name, decoded.Name)
		assert.Equal(t, original.Active, decoded.Active)
	})

	t.Run("Pointers", func(t *testing.T) {
		registry := codec.NewCodecRegistry()
		registry.RegisterTypes(WithPointers{})

		i := 42
		original := WithPointers{PtrInt: &i}

		encoded, err := registry.Encode(&original)
		require.NoError(t, err)

		var decoded WithPointers
		err = registry.Decode(encoded, &decoded)
		require.NoError(t, err)

		assert.Equal(t, *original.PtrInt, *decoded.PtrInt)
	})

	t.Run("Slices", func(t *testing.T) {
		registry := codec.NewCodecRegistry()
		registry.RegisterTypes(SliceContainer{})

		original := SliceContainer{
			Ints:    []int{1, 2, 3},
			Strings: []string{"a", "b"},
		}

		encoded, err := registry.Encode(&original)
		require.NoError(t, err)

		var decoded SliceContainer
		err = registry.Decode(encoded, &decoded)
		require.NoError(t, err)

		assert.Equal(t, original, decoded)
	})

	t.Run("LazyRegistration", func(t *testing.T) {
		registry := codec.NewCodecRegistry()

		val := NestedStruct{Val: "lazy"}
		encoded, err := registry.EncodeValue(&val)
		require.NoError(t, err)

		var decoded NestedStruct
		err = registry.Decode(encoded, &decoded)
		require.NoError(t, err)

		assert.Equal(t, val, decoded)
	})

	t.Run("EmptyStruct", func(t *testing.T) {
		registry := codec.NewCodecRegistry()
		registry.RegisterTypes(EmptyStruct{})

		original := EmptyStruct{}
		encoded, err := registry.Encode(&original)
		require.NoError(t, err)

		var decoded EmptyStruct
		err = registry.Decode(encoded, &decoded)
		require.NoError(t, err)

		assert.Equal(t, original, decoded)
	})

	t.Run("OnlyEmbedded", func(t *testing.T) {
		registry := codec.NewCodecRegistry()
		registry.RegisterTypes(OnlyEmbedded{}, Status{})

		original := OnlyEmbedded{Status: Status{Active: true}}
		encoded, err := registry.Encode(&original)
		require.NoError(t, err)

		assert.Equal(t, true, encoded["active"])

		var decoded OnlyEmbedded
		err = registry.Decode(encoded, &decoded)
		require.NoError(t, err)

		assert.Equal(t, true, decoded.Active)
	})

	t.Run("SamePrimitiveFields", func(t *testing.T) {
		registry := codec.NewCodecRegistry()
		registry.RegisterTypes(SamePrimitive{})

		original := SamePrimitive{A: 1, B: 2, C: 3}
		encoded, err := registry.Encode(&original)
		require.NoError(t, err)

		assert.Equal(t, int64(1), encoded["a"])
		assert.Equal(t, int64(2), encoded["b"])
		assert.Equal(t, int64(3), encoded["c"])

		var decoded SamePrimitive
		err = registry.Decode(encoded, &decoded)
		require.NoError(t, err)

		assert.Equal(t, original.A, decoded.A)
		assert.Equal(t, original.B, decoded.B)
		assert.Equal(t, original.C, decoded.C)
	})

	t.Run("NumericOverflow", func(t *testing.T) {
		registry := codec.NewCodecRegistry()
		registry.RegisterTypes(NumericOverflow{})

		original := NumericOverflow{
			SmallInt:  127, // max int8
			SmallUint: 255, // max uint8
		}

		encoded, err := registry.Encode(&original)
		require.NoError(t, err)

		var decoded NumericOverflow
		err = registry.Decode(encoded, &decoded)
		require.NoError(t, err)

		assert.Equal(t, original.SmallInt, decoded.SmallInt)
		assert.Equal(t, original.SmallUint, decoded.SmallUint)
	})

	t.Run("MultipleEmbedded", func(t *testing.T) {
		registry := codec.NewCodecRegistry()
		registry.RegisterTypes(MultiEmbedWithMeta{}, Metadata{}, Status{})

		original := MultiEmbedWithMeta{
			Name:     "Test",
			Metadata: Metadata{CreatedAt: time.Time{}, UpdatedAt: time.Time{}},
			Status:   Status{Active: true},
		}

		encoded, err := registry.Encode(&original)
		require.NoError(t, err)

		assert.Equal(t, "Test", encoded["name"])
		assert.Equal(t, true, encoded["active"])

		var decoded MultiEmbedWithMeta
		err = registry.Decode(encoded, &decoded)
		require.NoError(t, err)

		assert.Equal(t, original.Name, decoded.Name)
		assert.Equal(t, original.Active, decoded.Active)
	})

	t.Run("AllPrimitiveVariantsRoundTrip", func(t *testing.T) {
		registry := codec.NewCodecRegistry()
		registry.RegisterTypes(PrimitiveTypes{})

		original := PrimitiveTypes{
			Int8:    -128,  // min int8
			Int16:   32767, // max int16
			Int32:   -2147483648,
			Int64:   9223372036854775807, // max int64
			Uint:    ^uint(0),            // max uint
			Uint8:   255,
			Uint16:  65535,
			Uint32:  4294967295,
			Uint64:  18446744073709551615, // max uint64
			Float32: 3.14159,
			Bytes:   []byte("hello world"),
		}

		encoded, err := registry.Encode(&original)
		require.NoError(t, err)

		var decoded PrimitiveTypes
		err = registry.Decode(encoded, &decoded)
		require.NoError(t, err)

		assert.Equal(t, original.Int8, decoded.Int8)
		assert.Equal(t, original.Int16, decoded.Int16)
		assert.Equal(t, original.Uint8, decoded.Uint8)
		assert.Equal(t, original.Bytes, decoded.Bytes)
	})
}
