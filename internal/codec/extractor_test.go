package codec_test

import (
	"reflect"
	"testing"

	"github.com/rlch/neogo/internal/codec"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestExtractNeo4jNodeMeta tests Neo4j node metadata extraction
func TestExtractNeo4jNodeMeta(t *testing.T) {
	t.Run("InvalidType", func(t *testing.T) {
		registry := codec.NewCodecRegistry()

		_, err := registry.ExtractNeo4jNodeMeta(42)
		assert.Error(t, err)
	})
}

// TestReflectionHelpers tests reflection helper functions
func TestReflectionHelpers(t *testing.T) {
	t.Run("UnwindType", func(t *testing.T) {
		// Test with pointer
		ptrType := reflect.TypeOf((*TestStruct)(nil))
		unwound := codec.UnwindType(ptrType)
		assert.Equal(t, reflect.Struct, unwound.Kind())
		assert.Equal(t, "TestStruct", unwound.Name())
	})

	t.Run("UnwindValue", func(t *testing.T) {
		// Test with pointer value
		s := &TestStruct{Int: 42}
		val := reflect.ValueOf(s)
		unwound := codec.UnwindValue(val)
		assert.Equal(t, reflect.Struct, unwound.Kind())
	})

	t.Run("GetTypeName", func(t *testing.T) {
		registry := codec.NewCodecRegistry()

		s := TestStruct{Int: 42}
		name := registry.GetTypeName(s)
		assert.Equal(t, "TestStruct", name)
	})

	t.Run("GetTypeNameWithPointer", func(t *testing.T) {
		registry := codec.NewCodecRegistry()

		s := &TestStruct{Int: 42}
		name := registry.GetTypeName(s)
		assert.Equal(t, "TestStruct", name)
	})

	t.Run("IsRegistered", func(t *testing.T) {
		registry := codec.NewCodecRegistry()
		registry.RegisterTypes(TestStruct{})

		s := TestStruct{Int: 42}
		assert.True(t, registry.IsRegistered(s))

		ns := NestedStruct{Val: "test"}
		assert.False(t, registry.IsRegistered(ns))
	})

	t.Run("CreateInstance", func(t *testing.T) {
		registry := codec.NewCodecRegistry()

		typ := reflect.TypeOf(TestStruct{})
		instance, err := registry.CreateInstance(typ)
		require.NoError(t, err)
		assert.NotNil(t, instance)

		// Should be a pointer to struct
		assert.Equal(t, reflect.Ptr, reflect.TypeOf(instance).Kind())
	})

	t.Run("CreateInstanceWithPointer", func(t *testing.T) {
		registry := codec.NewCodecRegistry()

		typ := reflect.TypeOf((*TestStruct)(nil))
		instance, err := registry.CreateInstance(typ)
		require.NoError(t, err)
		assert.NotNil(t, instance)
	})

	t.Run("CreateInstanceInvalidType", func(t *testing.T) {
		registry := codec.NewCodecRegistry()

		typ := reflect.TypeOf(42)
		_, err := registry.CreateInstance(typ)
		assert.Error(t, err)
	})

	t.Run("WalkStruct", func(t *testing.T) {
		s := TestStruct{Int: 42, String: "hello"}
		val := reflect.ValueOf(s)

		fieldCount := 0
		codec.WalkStruct(val, func(i int, field reflect.StructField, fval reflect.Value) (bool, error) {
			fieldCount++
			return false, nil
		})

		// Should have walked through some fields
		assert.Greater(t, fieldCount, 0)
	})

	t.Run("WalkStructWithPointer", func(t *testing.T) {
		s := &TestStruct{Int: 42, String: "hello"}
		val := reflect.ValueOf(s)

		fieldCount := 0
		codec.WalkStruct(val, func(i int, field reflect.StructField, fval reflect.Value) (bool, error) {
			fieldCount++
			return false, nil
		})

		assert.Greater(t, fieldCount, 0)
	})

	t.Run("WalkStructInvalidType", func(t *testing.T) {
		val := reflect.ValueOf(42)

		err := codec.WalkStruct(val, func(i int, field reflect.StructField, fval reflect.Value) (bool, error) {
			return false, nil
		})

		assert.Error(t, err)
	})
}

// TestCompileMapEncoder tests map compilation for encoding
func TestCompileMapEncoder(t *testing.T) {
	t.Run("StringStringMapType", func(t *testing.T) {
		compiler := codec.NewCompiler()

		mapType := reflect.TypeOf(map[string]string{})
		op, err := compiler.Compile(mapType)

		require.NoError(t, err)
		assert.NotNil(t, op)
	})

	t.Run("StringIntMapType", func(t *testing.T) {
		compiler := codec.NewCompiler()

		mapType := reflect.TypeOf(map[string]int{})
		op, err := compiler.Compile(mapType)

		require.NoError(t, err)
		assert.NotNil(t, op)
	})

	t.Run("NonStringKeyMapFails", func(t *testing.T) {
		compiler := codec.NewCompiler()

		mapType := reflect.TypeOf(map[int]string{})
		_, err := compiler.Compile(mapType)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "string")
	})
}

// TestCompileMapDecoder tests map compilation for decoding
func TestCompileMapDecoder(t *testing.T) {
	t.Run("StringStringMapType", func(t *testing.T) {
		compiler := codec.NewCompiler()

		mapType := reflect.TypeOf(map[string]string{})
		decoder, err := compiler.CompileDecoder(mapType)

		// Maps should be rejected for decoding
		assert.Error(t, err)
		assert.Nil(t, decoder)
	})

	t.Run("StringIntMapType", func(t *testing.T) {
		compiler := codec.NewCompiler()

		mapType := reflect.TypeOf(map[string]int{})
		decoder, err := compiler.CompileDecoder(mapType)

		// Maps should be rejected for decoding
		assert.Error(t, err)
		assert.Nil(t, decoder)
	})
}

// TestTimeDecoder tests time.Time decoding
func TestTimeDecoder(t *testing.T) {
	t.Run("DecodeTime", func(t *testing.T) {
		registry := codec.NewCodecRegistry()
		registry.RegisterTypes(TestStruct{})

		s := TestStruct{Int: 1, String: "test"}
		encoded, err := registry.Encode(&s)
		require.NoError(t, err)

		var decoded TestStruct
		err = registry.Decode(encoded, &decoded)
		require.NoError(t, err)

		// Time should be zero-value if not set
		assert.Equal(t, s.Time, decoded.Time)
	})
}
