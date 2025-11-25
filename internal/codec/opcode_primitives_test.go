package codec_test

import (
	"testing"

	"github.com/rlch/neogo/internal/codec"
	"github.com/stretchr/testify/assert"
)

// Low-level tests for opcode execution with primitive types

// TestOpcodeBoolEncoding tests bool field encoding
func TestOpcodeBoolEncoding(t *testing.T) {
	registry := codec.NewCodecRegistry()
	registry.RegisterTypes(WithBools{})

	t.Run("EncodeTrueValue", func(t *testing.T) {
		original := WithBools{True: true, False: false}
		encoded, err := registry.Encode(&original)

		assert.NoError(t, err)
		assert.Equal(t, true, encoded["true"])
		assert.Equal(t, false, encoded["false"])
	})

	t.Run("EncodeFalseValue", func(t *testing.T) {
		original := WithBools{True: false, False: true}
		encoded, err := registry.Encode(&original)

		assert.NoError(t, err)
		assert.Equal(t, false, encoded["true"])
		assert.Equal(t, true, encoded["false"])
	})
}

// TestOpcodeIntEncoding tests various int types encoding
func TestOpcodeIntEncoding(t *testing.T) {
	registry := codec.NewCodecRegistry()
	registry.RegisterTypes(PrimitiveTypes{})

	tests := []struct {
		name  string
		setup func() PrimitiveTypes
		check func(map[string]interface{})
	}{
		{
			name: "Int8Encoding",
			setup: func() PrimitiveTypes {
				return PrimitiveTypes{Int8: -42}
			},
			check: func(enc map[string]interface{}) {
				assert.Equal(t, int64(-42), enc["int8"])
			},
		},
		{
			name: "Int16Encoding",
			setup: func() PrimitiveTypes {
				return PrimitiveTypes{Int16: 1000}
			},
			check: func(enc map[string]interface{}) {
				assert.Equal(t, int64(1000), enc["int16"])
			},
		},
		{
			name: "Int32Encoding",
			setup: func() PrimitiveTypes {
				return PrimitiveTypes{Int32: -100000}
			},
			check: func(enc map[string]interface{}) {
				assert.Equal(t, int64(-100000), enc["int32"])
			},
		},
		{
			name: "Int64Encoding",
			setup: func() PrimitiveTypes {
				return PrimitiveTypes{Int64: 9223372036854775807}
			},
			check: func(enc map[string]interface{}) {
				assert.Equal(t, int64(9223372036854775807), enc["int64"])
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			original := tt.setup()
			encoded, err := registry.Encode(&original)

			assert.NoError(t, err)
			tt.check(encoded)
		})
	}
}

// TestOpcodeUintEncoding tests various uint types encoding
func TestOpcodeUintEncoding(t *testing.T) {
	registry := codec.NewCodecRegistry()
	registry.RegisterTypes(PrimitiveTypes{})

	tests := []struct {
		name  string
		setup func() PrimitiveTypes
		check func(map[string]interface{})
	}{
		{
			name: "Uint8Encoding",
			setup: func() PrimitiveTypes {
				return PrimitiveTypes{Uint8: 200}
			},
			check: func(enc map[string]interface{}) {
				assert.Equal(t, int64(200), enc["uint8"])
			},
		},
		{
			name: "Uint16Encoding",
			setup: func() PrimitiveTypes {
				return PrimitiveTypes{Uint16: 50000}
			},
			check: func(enc map[string]interface{}) {
				assert.Equal(t, int64(50000), enc["uint16"])
			},
		},
		{
			name: "Uint32Encoding",
			setup: func() PrimitiveTypes {
				return PrimitiveTypes{Uint32: 4000000000}
			},
			check: func(enc map[string]interface{}) {
				assert.Equal(t, int64(4000000000), enc["uint32"])
			},
		},
		{
			name: "Uint64Encoding",
			setup: func() PrimitiveTypes {
				return PrimitiveTypes{Uint64: 18446744073709551615}
			},
			check: func(enc map[string]interface{}) {
				// Note: uint64 max will overflow int64 when encoded
				assert.Equal(t, int64(-1), enc["uint64"])
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			original := tt.setup()
			encoded, err := registry.Encode(&original)

			assert.NoError(t, err)
			tt.check(encoded)
		})
	}
}

// TestOpcodeFloatEncoding tests float types encoding
func TestOpcodeFloatEncoding(t *testing.T) {
	registry := codec.NewCodecRegistry()
	registry.RegisterTypes(PrimitiveTypes{})

	t.Run("Float32Encoding", func(t *testing.T) {
		original := PrimitiveTypes{Float32: 3.14}
		encoded, err := registry.Encode(&original)

		assert.NoError(t, err)
		assert.InDelta(t, 3.14, encoded["float32"], 0.01)
	})
}

// TestOpcodeStringEncoding tests string encoding (via TestStruct which has a String field)
func TestOpcodeStringEncoding(t *testing.T) {
	registry := codec.NewCodecRegistry()
	registry.RegisterTypes(TestStruct{})

	t.Run("EmptyString", func(t *testing.T) {
		original := TestStruct{String: ""}
		encoded, err := registry.Encode(&original)

		assert.NoError(t, err)
		assert.Equal(t, "", encoded["string"])
	})

	t.Run("SimpleString", func(t *testing.T) {
		original := TestStruct{String: "hello world"}
		encoded, err := registry.Encode(&original)

		assert.NoError(t, err)
		assert.Equal(t, "hello world", encoded["string"])
	})

	t.Run("StringWithSpecialChars", func(t *testing.T) {
		original := TestStruct{String: "hello\nworld\t!"}
		encoded, err := registry.Encode(&original)

		assert.NoError(t, err)
		assert.Equal(t, "hello\nworld\t!", encoded["string"])
	})

	t.Run("LongString", func(t *testing.T) {
		longStr := ""
		for i := 0; i < 10000; i++ {
			longStr += "x"
		}
		original := TestStruct{String: longStr}
		encoded, err := registry.Encode(&original)

		assert.NoError(t, err)
		assert.Equal(t, longStr, encoded["string"])
	})
}

// TestOpcodeByteArrayEncoding tests []byte encoding
func TestOpcodeByteArrayEncoding(t *testing.T) {
	registry := codec.NewCodecRegistry()
	registry.RegisterTypes(PrimitiveTypes{})

	t.Run("EmptyBytes", func(t *testing.T) {
		original := PrimitiveTypes{Bytes: []byte{}}
		encoded, err := registry.Encode(&original)

		assert.NoError(t, err)
		assert.Equal(t, []byte{}, encoded["bytes"])
	})

	t.Run("SimpleBytes", func(t *testing.T) {
		original := PrimitiveTypes{Bytes: []byte("hello")}
		encoded, err := registry.Encode(&original)

		assert.NoError(t, err)
		assert.Equal(t, []byte("hello"), encoded["bytes"])
	})

	t.Run("BinaryBytes", func(t *testing.T) {
		original := PrimitiveTypes{Bytes: []byte{0, 1, 2, 255, 254, 253}}
		encoded, err := registry.Encode(&original)

		assert.NoError(t, err)
		assert.Equal(t, []byte{0, 1, 2, 255, 254, 253}, encoded["bytes"])
	})

	t.Run("NilBytes", func(t *testing.T) {
		original := PrimitiveTypes{Bytes: nil}
		encoded, err := registry.Encode(&original)

		assert.NoError(t, err)
		assert.Nil(t, encoded["bytes"])
	})
}

// TestOpcodeZeroValues tests encoding of zero values
func TestOpcodeZeroValues(t *testing.T) {
	registry := codec.NewCodecRegistry()
	registry.RegisterTypes(PrimitiveTypes{})

	t.Run("AllZeroValues", func(t *testing.T) {
		original := PrimitiveTypes{}
		encoded, err := registry.Encode(&original)

		assert.NoError(t, err)
		// Zero values are still encoded
		assert.Equal(t, int64(0), encoded["int8"])
		assert.Equal(t, int64(0), encoded["int16"])
		assert.Equal(t, int64(0), encoded["int32"])
		assert.Equal(t, int64(0), encoded["int64"])
		// Note: Float32 is in PrimitiveTypes, and defaults to 0.0
		// But it's encoded as float64 in the output
		// Bytes defaults to nil in zero-value struct
		assert.Nil(t, encoded["bytes"])
	})
}
