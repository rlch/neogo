package codec_test

import (
	"testing"
	"time"

	"github.com/neo4j/neo4j-go-driver/v5/neo4j"
	"github.com/rlch/neogo/internal/codec"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestRoundTrip tests comprehensive struct with all field types
func TestRoundTrip(t *testing.T) {
	registry := codec.NewCodecRegistry()
	registry.RegisterTypes(TestStruct{}, NestedStruct{})

	ptrInt := 42
	now := time.Now().Truncate(time.Millisecond)
	point := neo4j.Point2D{X: 1.5, Y: 2.5, SpatialRefId: 4326}

	original := TestStruct{
		Int:    123,
		String: "hello",
		Bool:   true,
		Float:  3.14,
		PtrInt: &ptrInt,
		SliceInt: []int{1, 2, 3},
		SliceStr: []string{"a", "b"},
		Nested: NestedStruct{
			Val: "nested_val",
		},
		PtrNested: &NestedStruct{
			Val: "ptr_nested_val",
		},
		EmbeddedStruct: EmbeddedStruct{
			EmbedVal: 999,
		},
		Time:  now,
		Point: point,
	}

	// Encode
	encoded, err := registry.Encode(&original)
	require.NoError(t, err)

	// Verify Encoding
	assert.Equal(t, int64(123), encoded["int"])
	assert.Equal(t, "hello", encoded["string"])
	assert.Equal(t, true, encoded["bool"])
	assert.Equal(t, 3.14, encoded["float"])
	assert.Equal(t, int64(42), encoded["ptr_int"])

	// Slice elements (encoded as []any)
	sliceInt := encoded["slice_int"].([]any)
	assert.Len(t, sliceInt, 3)
	assert.Equal(t, int64(1), sliceInt[0])

	// Nested
	nested := encoded["nested"].(map[string]any)
	assert.Equal(t, "nested_val", nested["val"])

	// Embedded (flattened)
	assert.Equal(t, int64(999), encoded["embed_val"])

	// Decode
	var decoded TestStruct
	err = registry.Decode(encoded, &decoded)
	require.NoError(t, err)

	// Verify Decoding
	assert.Equal(t, original.Int, decoded.Int)
	assert.Equal(t, original.String, decoded.String)
	assert.Equal(t, original.Bool, decoded.Bool)
	assert.Equal(t, original.Float, decoded.Float)
	assert.Equal(t, *original.PtrInt, *decoded.PtrInt)
	assert.Equal(t, original.SliceInt, decoded.SliceInt)
	assert.Equal(t, original.SliceStr, decoded.SliceStr)
	assert.Equal(t, original.Nested, decoded.Nested)
	assert.Equal(t, original.PtrNested.Val, decoded.PtrNested.Val)
	assert.Equal(t, original.EmbeddedStruct, decoded.EmbeddedStruct)
	assert.Equal(t, original.Time, decoded.Time)
	assert.Equal(t, original.Point, decoded.Point)
}

// TestNestedStructs tests multi-level struct nesting
func TestNestedStructs(t *testing.T) {
	registry := codec.NewCodecRegistry()
	registry.RegisterTypes(Company{}, Address{})

	original := Company{
		Name: "Tech Corp",
		Address: Address{
			Street: "123 Main St",
			City:   "New York",
			Zip:    "10001",
		},
	}

	// Encode
	encoded, err := registry.Encode(&original)
	require.NoError(t, err)

	address := encoded["address"].(map[string]any)
	assert.Equal(t, "Tech Corp", encoded["name"])
	assert.Equal(t, "123 Main St", address["street"])
	assert.Equal(t, "New York", address["city"])
	assert.Equal(t, "10001", address["zip"])

	// Decode
	var decoded Company
	err = registry.Decode(encoded, &decoded)
	require.NoError(t, err)

	assert.Equal(t, original, decoded)
}

// TestDeeplyNestedStructs tests 3-level nesting
func TestDeeplyNestedStructs(t *testing.T) {
	registry := codec.NewCodecRegistry()
	registry.RegisterTypes(Level1{}, Level2{}, Level3{})

	original := Level1{
		ID: 1,
		Level2: Level2{
			Name: "Level2",
			Level3: Level3{
				Value: "Level3",
			},
		},
	}

	// Encode
	encoded, err := registry.Encode(&original)
	require.NoError(t, err)

	level2 := encoded["level2"].(map[string]any)
	level3 := level2["level3"].(map[string]any)

	assert.Equal(t, int64(1), encoded["id"])
	assert.Equal(t, "Level2", level2["name"])
	assert.Equal(t, "Level3", level3["value"])

	// Decode
	var decoded Level1
	err = registry.Decode(encoded, &decoded)
	require.NoError(t, err)

	assert.Equal(t, original, decoded)
}

// TestMultipleEmbedded tests multiple embedded structs
func TestMultipleEmbedded(t *testing.T) {
	registry := codec.NewCodecRegistry()
	registry.RegisterTypes(MultiEmbed{}, Metadata{}, Status{})

	now := time.Now().Truncate(time.Millisecond)
	later := now.Add(time.Hour).Truncate(time.Millisecond)

	original := MultiEmbed{
		Name: "Test",
		Metadata: Metadata{
			CreatedAt: now,
			UpdatedAt: later,
		},
		Status: Status{
			Active: true,
		},
	}

	// Encode
	encoded, err := registry.Encode(&original)
	require.NoError(t, err)

	assert.Equal(t, "Test", encoded["name"])
	assert.Equal(t, true, encoded["active"])
	assert.NotNil(t, encoded["created_at"])
	assert.NotNil(t, encoded["updated_at"])

	// Decode
	var decoded MultiEmbed
	err = registry.Decode(encoded, &decoded)
	require.NoError(t, err)

	assert.Equal(t, original.Name, decoded.Name)
	assert.Equal(t, original.Active, decoded.Active)
	assert.Equal(t, original.CreatedAt, decoded.CreatedAt)
	assert.Equal(t, original.UpdatedAt, decoded.UpdatedAt)
}
