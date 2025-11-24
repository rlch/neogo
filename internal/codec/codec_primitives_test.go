package codec_test

import (
	"testing"

	"github.com/rlch/neogo/internal/codec"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestPrimitiveTypes tests encoding/decoding of all primitive types
func TestPrimitiveTypes(t *testing.T) {
	registry := codec.NewCodecRegistry()
	registry.RegisterTypes(PrimitiveTypes{})

	original := PrimitiveTypes{
		Int8:    -8,
		Int16:   -16,
		Int32:   -32,
		Int64:   -64,
		Uint:    1,
		Uint8:   8,
		Uint16:  16,
		Uint32:  32,
		Uint64:  64,
		Float32: 3.14,
		Bytes:   []byte("hello"),
	}

	// Encode
	encoded, err := registry.Encode(&original)
	require.NoError(t, err)

	// Verify
	assert.Equal(t, int64(-8), encoded["int8"])
	assert.Equal(t, int64(-16), encoded["int16"])
	assert.Equal(t, int64(-32), encoded["int32"])
	assert.Equal(t, int64(-64), encoded["int64"])
	assert.Equal(t, int64(1), encoded["uint"])
	assert.Equal(t, int64(8), encoded["uint8"])
	assert.Equal(t, int64(16), encoded["uint16"])
	assert.Equal(t, int64(32), encoded["uint32"])
	assert.Equal(t, int64(64), encoded["uint64"])
	assert.InDelta(t, float64(3.14), encoded["float32"], 0.01)
	assert.Equal(t, "hello", string(encoded["bytes"].([]byte)))

	// Decode
	var decoded PrimitiveTypes
	err = registry.Decode(encoded, &decoded)
	require.NoError(t, err)

	assert.Equal(t, original.Int8, decoded.Int8)
	assert.Equal(t, original.Int16, decoded.Int16)
	assert.Equal(t, original.Int32, decoded.Int32)
	assert.Equal(t, original.Int64, decoded.Int64)
	assert.Equal(t, original.Uint, decoded.Uint)
	assert.Equal(t, original.Uint8, decoded.Uint8)
	assert.Equal(t, original.Uint16, decoded.Uint16)
	assert.Equal(t, original.Uint32, decoded.Uint32)
	assert.Equal(t, original.Uint64, decoded.Uint64)
	assert.InDelta(t, original.Float32, decoded.Float32, 0.01)
	assert.Equal(t, original.Bytes, decoded.Bytes)
}

// TestBoolFields tests boolean field encoding/decoding
func TestBoolFields(t *testing.T) {
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
}

// TestNumericBoundaries tests max/min values for numeric types
func TestNumericBoundaries(t *testing.T) {
	registry := codec.NewCodecRegistry()
	registry.RegisterTypes(Boundaries{})

	original := Boundaries{
		MaxInt64:   9223372036854775807,
		MinInt64:   -9223372036854775808,
		MaxFloat64: 1.7976931348623157e+308,
		MinFloat64: -1.7976931348623157e+308,
	}

	encoded, err := registry.Encode(&original)
	require.NoError(t, err)

	var decoded Boundaries
	err = registry.Decode(encoded, &decoded)
	require.NoError(t, err)

	assert.Equal(t, original.MaxInt64, decoded.MaxInt64)
	assert.Equal(t, original.MinInt64, decoded.MinInt64)
	assert.InDelta(t, original.MaxFloat64, decoded.MaxFloat64, 1e+300)
	assert.InDelta(t, original.MinFloat64, decoded.MinFloat64, 1e+300)
}

// TestTypeCoercion tests converting between int types
func TestTypeCoercion(t *testing.T) {
	registry := codec.NewCodecRegistry()
	registry.RegisterTypes(IntTypes{})

	original := IntTypes{
		Int:    -100,
		Int64:  9223372036854775807, // max int64
		Uint:   100,
		Uint32: 4294967295, // max uint32
	}

	encoded, err := registry.Encode(&original)
	require.NoError(t, err)

	var decoded IntTypes
	err = registry.Decode(encoded, &decoded)
	require.NoError(t, err)

	assert.Equal(t, original.Int, decoded.Int)
	assert.Equal(t, original.Int64, decoded.Int64)
	assert.Equal(t, original.Uint, decoded.Uint)
	assert.Equal(t, original.Uint32, decoded.Uint32)
}

// TestEncodeValueZeros tests EncodeValue with zero values
func TestEncodeValueZeros(t *testing.T) {
	registry := codec.NewCodecRegistry()

	// String zero value
	val, err := registry.EncodeValue("")
	require.NoError(t, err)
	assert.Equal(t, "", val)

	// Int zero value
	val, err = registry.EncodeValue(0)
	require.NoError(t, err)
	assert.Equal(t, int64(0), val)
}
