package codec_test

import (
	"testing"

	"github.com/rlch/neogo/internal/codec"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestTags tests tag parsing and field handling
func TestTags(t *testing.T) {
	t.Run("FieldSkip", func(t *testing.T) {
		registry := codec.NewCodecRegistry()
		registry.RegisterTypes(WithSkip{})

		original := WithSkip{
			Name:     "test",
			Internal: "should not encode",
			Other:    "other",
		}

		encoded, err := registry.Encode(&original)
		require.NoError(t, err)

		assert.Equal(t, "test", encoded["name"])
		assert.Equal(t, "other", encoded["other"])
		assert.Nil(t, encoded["internal"])

		var decoded WithSkip
		err = registry.Decode(encoded, &decoded)
		require.NoError(t, err)

		assert.Equal(t, original.Name, decoded.Name)
		assert.Equal(t, original.Other, decoded.Other)
		assert.Equal(t, "", decoded.Internal)
	})

	t.Run("EmbeddedStructFlattening", func(t *testing.T) {
		registry := codec.NewCodecRegistry()
		registry.RegisterTypes(MultiEmbed{}, Status{})

		original := MultiEmbed{
			Name:   "Test",
			Status: Status{Active: true},
		}

		encoded, err := registry.Encode(&original)
		require.NoError(t, err)

		assert.Equal(t, "Test", encoded["name"])
		assert.Equal(t, true, encoded["active"])

		var decoded MultiEmbed
		err = registry.Decode(encoded, &decoded)
		require.NoError(t, err)

		assert.Equal(t, original.Name, decoded.Name)
		assert.Equal(t, original.Active, decoded.Active)
	})

	t.Run("CustomFieldNames", func(t *testing.T) {
		registry := codec.NewCodecRegistry()
		registry.RegisterTypes(Company{}, Address{})

		original := Company{
			Name: "My Company",
			Address: Address{
				Street: "123 Main",
				City:   "NYC",
				Zip:    "10001",
			},
		}

		encoded, err := registry.Encode(&original)
		require.NoError(t, err)

		assert.Equal(t, "My Company", encoded["name"])
		address := encoded["address"].(map[string]any)
		assert.Equal(t, "123 Main", address["street"])
		assert.Equal(t, "NYC", address["city"])

		var decoded Company
		err = registry.Decode(encoded, &decoded)
		require.NoError(t, err)

		assert.Equal(t, original, decoded)
	})

	t.Run("MissingFieldsInData", func(t *testing.T) {
		registry := codec.NewCodecRegistry()
		registry.RegisterTypes(WithSkip{})

		// Only provide some fields
		data := map[string]any{
			"name": "test",
		}

		var result WithSkip
		err := registry.Decode(data, &result)
		require.NoError(t, err)

		assert.Equal(t, "test", result.Name)
		assert.Equal(t, "", result.Other) // Zero value
	})

	t.Run("SkipFieldNotEncodedInMap", func(t *testing.T) {
		registry := codec.NewCodecRegistry()
		registry.RegisterTypes(WithSkip{})

		original := WithSkip{
			Name:     "visible",
			Internal: "hidden",
			Other:    "also_visible",
		}

		encoded, err := registry.Encode(&original)
		require.NoError(t, err)

		// Verify the skipped field is NOT in the encoded map
		_, hasInternal := encoded["internal"]
		_, hasInternal2 := encoded["Internal"]
		assert.False(t, hasInternal, "internal field should not be in encoded map")
		assert.False(t, hasInternal2, "Internal field should not be in encoded map")

		// Verify other fields ARE present
		assert.Equal(t, "visible", encoded["name"])
		assert.Equal(t, "also_visible", encoded["other"])
	})

	t.Run("OnlyEmbeddedFieldFlattening", func(t *testing.T) {
		registry := codec.NewCodecRegistry()
		registry.RegisterTypes(OnlyEmbedded{}, Status{})

		original := OnlyEmbedded{
			Status: Status{Active: true},
		}

		encoded, err := registry.Encode(&original)
		require.NoError(t, err)

		// Field should be flattened, not nested
		assert.Equal(t, true, encoded["active"])
		assert.Nil(t, encoded["status"])

		var decoded OnlyEmbedded
		err = registry.Decode(encoded, &decoded)
		require.NoError(t, err)

		assert.Equal(t, original.Active, decoded.Active)
	})

	t.Run("ExtraFieldsInData", func(t *testing.T) {
		registry := codec.NewCodecRegistry()
		registry.RegisterTypes(Address{})

		// Provide more fields than the struct has
		data := map[string]any{
			"street": "123 Main",
			"city":   "NYC",
			"zip":    "10001",
			"extra":  "should be ignored",
		}

		var result Address
		err := registry.Decode(data, &result)
		require.NoError(t, err)

		assert.Equal(t, "123 Main", result.Street)
		assert.Equal(t, "NYC", result.City)
		assert.Equal(t, "10001", result.Zip)
	})

	t.Run("ExplicitDbTagNamesPreferred", func(t *testing.T) {
		// BUG: Same TypePtr mismatch affects GetTypeMetadata here too.
		// Verify that explicit db tags work during actual encoding/decoding.
		registry := codec.NewCodecRegistry()
		registry.RegisterTypes(WithSkip{})

		original := WithSkip{
			Name:     "test",
			Internal: "hidden",
			Other:    "visible",
		}

		encoded, err := registry.Encode(&original)
		require.NoError(t, err)

		// Verify field names are as expected
		assert.Equal(t, "test", encoded["name"])
		assert.Equal(t, "visible", encoded["other"])
		// Internal field should not be in encoded output
		_, hasInternal := encoded["internal"]
		assert.False(t, hasInternal, "skipped field should not be encoded")

		var decoded WithSkip
		err = registry.Decode(encoded, &decoded)
		require.NoError(t, err)

		assert.Equal(t, "test", decoded.Name)
		assert.Equal(t, "visible", decoded.Other)
	})
}
