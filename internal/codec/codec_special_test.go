package codec_test

import (
	"testing"

	"github.com/neo4j/neo4j-go-driver/v5/neo4j"
	"github.com/rlch/neogo/internal/codec"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestNeo4jTypes tests Neo4j types encoding/decoding
func TestNeo4jTypes(t *testing.T) {
	registry := codec.NewCodecRegistry()
	registry.RegisterTypes(AllNeo4jTypes{})

	original := AllNeo4jTypes{
		Point: neo4j.Point2D{X: 1.5, Y: 2.5, SpatialRefId: 4326},
	}

	encoded, err := registry.Encode(&original)
	require.NoError(t, err)

	// Point should be encoded as neo4j.Point2D directly (Neo4j native type)
	// This allows the Neo4j driver to serialize it correctly
	point, ok := encoded["point"].(neo4j.Point2D)
	require.True(t, ok, "expected neo4j.Point2D, got %T", encoded["point"])
	assert.InDelta(t, 1.5, point.X, 0.01)
	assert.InDelta(t, 2.5, point.Y, 0.01)
	assert.Equal(t, uint32(4326), point.SpatialRefId)

	var decoded AllNeo4jTypes
	err = registry.Decode(encoded, &decoded)
	require.NoError(t, err)

	assert.InDelta(t, original.Point.X, decoded.Point.X, 0.01)
	assert.InDelta(t, original.Point.Y, decoded.Point.Y, 0.01)
	assert.Equal(t, original.Point.SpatialRefId, decoded.Point.SpatialRefId)
}

// TestInterfaceField tests interface{} fields
func TestInterfaceField(t *testing.T) {
	registry := codec.NewCodecRegistry()
	registry.RegisterTypes(WithInterface{})

	// Encode with various types as interface{}
	original := WithInterface{
		Data: "test string",
	}

	encoded, err := registry.Encode(&original)
	require.NoError(t, err)
	assert.Equal(t, "test string", encoded["data"])

	var decoded WithInterface
	err = registry.Decode(encoded, &decoded)
	require.NoError(t, err)

	assert.Equal(t, original.Data, decoded.Data)
}

// TestEncodeValue tests EncodeValue with various types
func TestEncodeValue(t *testing.T) {
	registry := codec.NewCodecRegistry()
	registry.RegisterTypes(NestedStruct{})

	// String
	val, err := registry.EncodeValue("hello")
	require.NoError(t, err)
	assert.Equal(t, "hello", val)

	// Int
	val, err = registry.EncodeValue(42)
	require.NoError(t, err)
	assert.Equal(t, int64(42), val)

	// Float
	val, err = registry.EncodeValue(3.14)
	require.NoError(t, err)
	assert.InDelta(t, 3.14, val, 0.01)

	// Bool
	val, err = registry.EncodeValue(true)
	require.NoError(t, err)
	assert.Equal(t, true, val)

	// Struct
	val, err = registry.EncodeValue(&NestedStruct{Val: "test"})
	require.NoError(t, err)
	assert.Equal(t, "test", val.(map[string]any)["val"])
}

// TestFieldSkip tests field skip with db:"-"
func TestFieldSkip(t *testing.T) {
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
	assert.Equal(t, "", decoded.Internal) // Zero value since it was skipped
}

// TestDecodeErrors tests error handling in decoding
func TestDecodeErrors(t *testing.T) {
	registry := codec.NewCodecRegistry()
	registry.RegisterTypes(TestStruct{})

	// Invalid type for field
	invalid := map[string]any{
		"int": "not an int",
	}

	var result TestStruct
	err := registry.Decode(invalid, &result)
	assert.Error(t, err)
}

// TestLazyRegistration tests lazily registered types
func TestLazyRegistration(t *testing.T) {
	registry := codec.NewCodecRegistry()

	// Don't pre-register, just use
	val := NestedStruct{Val: "lazy"}

	encoded, err := registry.EncodeValue(&val)
	require.NoError(t, err)

	assert.Equal(t, "lazy", encoded.(map[string]any)["val"])

	var decoded NestedStruct
	err = registry.Decode(encoded, &decoded)
	require.NoError(t, err)

	assert.Equal(t, val, decoded)
}
