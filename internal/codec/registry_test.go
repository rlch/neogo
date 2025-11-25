package codec_test

import (
	"testing"

	"github.com/rlch/neogo/internal/codec"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestRegistry tests the CodecRegistry API
func TestRegistry(t *testing.T) {
	t.Run("EncodeValue", func(t *testing.T) {
		registry := codec.NewCodecRegistry()
		registry.RegisterTypes(NestedStruct{})

		val, err := registry.EncodeValue("hello")
		require.NoError(t, err)
		assert.Equal(t, "hello", val)

		val, err = registry.EncodeValue(42)
		require.NoError(t, err)
		assert.Equal(t, int64(42), val)

		val, err = registry.EncodeValue(3.14)
		require.NoError(t, err)
		assert.InDelta(t, 3.14, val, 0.01)

		val, err = registry.EncodeValue(true)
		require.NoError(t, err)
		assert.Equal(t, true, val)

		val, err = registry.EncodeValue(&NestedStruct{Val: "test"})
		require.NoError(t, err)
		assert.Equal(t, "test", val.(map[string]any)["val"])
	})

	t.Run("EncodeValueSlices", func(t *testing.T) {
		registry := codec.NewCodecRegistry()

		val, err := registry.EncodeValue([]int{1, 2, 3})
		require.NoError(t, err)
		assert.Equal(t, []any{int64(1), int64(2), int64(3)}, val)

		val, err = registry.EncodeValue([]string{"a", "b"})
		require.NoError(t, err)
		assert.Equal(t, []any{"a", "b"}, val)
	})

	t.Run("EncodeValueZeros", func(t *testing.T) {
		registry := codec.NewCodecRegistry()

		val, err := registry.EncodeValue("")
		require.NoError(t, err)
		assert.Equal(t, "", val)

		val, err = registry.EncodeValue(0)
		require.NoError(t, err)
		assert.Equal(t, int64(0), val)
	})

	t.Run("RegisterAndEncode", func(t *testing.T) {
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
		assert.Equal(t, "Tech Corp", encoded["name"])

		var decoded Company
		err = registry.Decode(encoded, &decoded)
		require.NoError(t, err)

		assert.Equal(t, original, decoded)
	})

	t.Run("MultipleRegistrations", func(t *testing.T) {
		registry := codec.NewCodecRegistry()

		registry.RegisterTypes(
			Level1{},
			Level2{},
			Level3{},
			WithPointers{},
			WithInterface{},
		)

		original1 := Level1{
			ID: 1,
			Level2: Level2{
				Name: "L2",
				Level3: Level3{Value: "L3"},
			},
		}

		encoded1, err := registry.Encode(&original1)
		require.NoError(t, err)

		var decoded1 Level1
		err = registry.Decode(encoded1, &decoded1)
		require.NoError(t, err)
		assert.Equal(t, original1, decoded1)

		original2 := WithInterface{
			Data: "test",
		}

		encoded2, err := registry.Encode(&original2)
		require.NoError(t, err)

		var decoded2 WithInterface
		err = registry.Decode(encoded2, &decoded2)
		require.NoError(t, err)
		assert.Equal(t, original2.Data, decoded2.Data)
	})

	t.Run("TypeMetadataRetrieval", func(t *testing.T) {
		registry := codec.NewCodecRegistry()
		registry.RegisterTypes(Company{}, Address{})

		metadata := registry.GetTypeMetadata(&Company{})
		require.NotNil(t, metadata, "metadata should be available after RegisterTypes")
		assert.Equal(t, "Company", metadata.Name)
		assert.Equal(t, "name", metadata.FieldsMap["Name"])
		assert.Equal(t, "address", metadata.FieldsMap["Address"])
	})

	t.Run("GetTypeByName", func(t *testing.T) {
		registry := codec.NewCodecRegistry()
		registry.RegisterTypes(Address{})

		retrieved := registry.GetTypeByName("Address")
		assert.NotNil(t, retrieved)
	})

	t.Run("IsRegisteredAfterRegisterTypes", func(t *testing.T) {
		registry := codec.NewCodecRegistry()
		registry.RegisterTypes(NestedStruct{})

		assert.True(t, registry.IsRegistered(&NestedStruct{}), "registered types should return true")
		assert.False(t, registry.IsRegistered(&Company{}), "unregistered types should return false")
	})

	t.Run("IsRegisteredAfterLazyCompilation", func(t *testing.T) {
		registry := codec.NewCodecRegistry()

		// Lazy compilation via EncodeValue does NOT populate IsRegistered
		val := NestedStruct{Val: "test"}
		_, err := registry.EncodeValue(&val)
		require.NoError(t, err)

		// Type is usable but not "registered" per the metadata
		assert.False(t, registry.IsRegistered(&NestedStruct{}), 
			"lazy compilation does not populate metadata, so IsRegistered remains false")
	})

	t.Run("UnregisteredStructCanLazyCompile", func(t *testing.T) {
		// Unlike some codecs, neogo allows lazy compilation of any struct.
		// ErrTypeNotRegistered is only returned for truly unsupported types.
		registry := codec.NewCodecRegistry()

		type UnregisteredType struct {
			Field string
		}

		unregistered := UnregisteredType{Field: "test"}
		_, err := registry.Encode(&unregistered)
		// Should NOT error because lazy compilation works for structs
		assert.NoError(t, err, "structs can be lazily compiled without RegisterTypes")
	})

	t.Run("DecodeIntoUnregisteredStruct", func(t *testing.T) {
		// Lazy decoding also works for unregistered structs
		registry := codec.NewCodecRegistry()

		type UnregisteredType struct {
			Field string
		}

		data := map[string]any{"field": "test"}
		var result UnregisteredType
		err := registry.Decode(data, &result)
		// Should NOT error; lazy compilation works
		assert.NoError(t, err, "structs can be lazily decoded without RegisterTypes")
		assert.Equal(t, "test", result.Field)
	})

	t.Run("LazyRegistrationViaEncode", func(t *testing.T) {
		registry := codec.NewCodecRegistry()

		original := NestedStruct{Val: "test"}
		// Don't register, just encode
		encoded, err := registry.EncodeValue(&original)
		require.NoError(t, err)

		assert.Equal(t, "test", encoded.(map[string]any)["val"])
	})

	t.Run("LazydRegistrationViaDecode", func(t *testing.T) {
		registry := codec.NewCodecRegistry()

		data := map[string]any{"val": "test_value"}
		var result NestedStruct
		// Don't register, just decode
		err := registry.Decode(data, &result)
		require.NoError(t, err)

		assert.Equal(t, "test_value", result.Val)
	})

	t.Run("CachedEncoderReuse", func(t *testing.T) {
		registry := codec.NewCodecRegistry()
		registry.RegisterTypes(Company{}, Address{})

		// First encode
		original := Company{Name: "Corp1", Address: Address{Street: "St1"}}
		encoded1, err := registry.Encode(&original)
		require.NoError(t, err)

		// Second encode - should reuse cached encoder
		original2 := Company{Name: "Corp2", Address: Address{Street: "St2"}}
		encoded2, err := registry.Encode(&original2)
		require.NoError(t, err)

		assert.Equal(t, "Corp1", encoded1["name"])
		assert.Equal(t, "Corp2", encoded2["name"])
	})

	t.Run("MultipleEmbeddedFlattening", func(t *testing.T) {
		registry := codec.NewCodecRegistry()
		registry.RegisterTypes(MultiEmbed{}, Metadata{}, Status{})

		original := MultiEmbed{
			Name: "test",
			Status: Status{Active: true},
		}

		encoded, err := registry.Encode(&original)
		require.NoError(t, err)

		// All fields should be at top level
		assert.Equal(t, "test", encoded["name"])
		assert.Equal(t, true, encoded["active"])
		// Metadata fields should also be present
		assert.NotNil(t, encoded["created_at"])
		assert.NotNil(t, encoded["updated_at"])

		var decoded MultiEmbed
		err = registry.Decode(encoded, &decoded)
		require.NoError(t, err)

		assert.Equal(t, "test", decoded.Name)
		assert.Equal(t, true, decoded.Active)
	})
}
