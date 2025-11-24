package codec_test

import (
	"testing"
	"time"

	"github.com/neo4j/neo4j-go-driver/v5/neo4j"
	"github.com/rlch/neogo/internal/codec"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type TestStruct struct {
	Int    int     `db:"int"`
	String string  `db:"string"`
	Bool   bool    `db:"bool"`
	Float  float64 `db:"float"`

	// Pointers
	PtrInt *int `db:"ptr_int"`

	// Slices
	SliceInt []int    `db:"slice_int"`
	SliceStr []string `db:"slice_str"`

	// Nested
	Nested    NestedStruct  `db:"nested"`
	PtrNested *NestedStruct `db:"ptr_nested"`

	// Embedded
	EmbeddedStruct `db:",embed"`

	// Neo4j types
	Time  time.Time     `db:"time"`
	Point neo4j.Point2D `db:"point"`
}

type NestedStruct struct {
	Val string `db:"val"`
}

type EmbeddedStruct struct {
	EmbedVal int `db:"embed_val"`
}

func TestRoundTrip(t *testing.T) {
	registry := codec.NewCodecRegistry()
	registry.RegisterTypes(TestStruct{}, NestedStruct{})

	ptrInt := 42
	now := time.Now().Truncate(time.Millisecond) // Truncate for consistent comparisons (json/map might lose precision?) No, direct copy.
	// But Neo4j time handling might be tricky.

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
	
	// Embedded (flattened or not? currently logic might put it in map, or merge?)
	// OpFieldEmbed logic in opcodes.go:
	// err := encodeStructToMap(current.SubOpcodes, ptr, result)
	// It merges into result.
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

func TestSliceOfStructs(t *testing.T) {
	registry := codec.NewCodecRegistry()
	registry.RegisterTypes(NestedStruct{})

	input := []NestedStruct{
		{Val: "1"},
		{Val: "2"},
	}

	// Encode value (slice)
	encoded, err := registry.EncodeValue(input)
	require.NoError(t, err)

	sliceEnc := encoded.([]any)
	assert.Len(t, sliceEnc, 2)
	assert.Equal(t, "1", sliceEnc[0].(map[string]any)["val"])

	var decoded []NestedStruct
	err = registry.Decode(sliceEnc, &decoded)
	require.NoError(t, err)
	assert.Equal(t, input, decoded)
}

// PrimitiveTypes tests encoding/decoding of all primitive types
type PrimitiveTypes struct {
	Int8    int8    `db:"int8"`
	Int16   int16   `db:"int16"`
	Int32   int32   `db:"int32"`
	Int64   int64   `db:"int64"`
	Uint    uint    `db:"uint"`
	Uint8   uint8   `db:"uint8"`
	Uint16  uint16  `db:"uint16"`
	Uint32  uint32  `db:"uint32"`
	Uint64  uint64  `db:"uint64"`
	Float32 float32 `db:"float32"`
	Bytes   []byte  `db:"bytes"`
}

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

// Test nil values
type NilableTypes struct {
	PtrString *string    `db:"ptr_string"`
	PtrInt    *int       `db:"ptr_int"`
	SliceVal  []string   `db:"slice_val"`
	NestedPtr *NestedStruct `db:"nested_ptr"`
}

func TestNilValues(t *testing.T) {
	registry := codec.NewCodecRegistry()
	registry.RegisterTypes(NilableTypes{}, NestedStruct{})

	original := NilableTypes{
		PtrString: nil,
		PtrInt:    nil,
		SliceVal:  nil,
		NestedPtr: nil,
	}

	// Encode
	encoded, err := registry.Encode(&original)
	require.NoError(t, err)

	// Verify nils
	assert.Nil(t, encoded["ptr_string"])
	assert.Nil(t, encoded["ptr_int"])
	assert.Nil(t, encoded["slice_val"])
	assert.Nil(t, encoded["nested_ptr"])

	// Decode
	var decoded NilableTypes
	err = registry.Decode(encoded, &decoded)
	require.NoError(t, err)

	assert.Nil(t, decoded.PtrString)
	assert.Nil(t, decoded.PtrInt)
	assert.Nil(t, decoded.SliceVal)
	assert.Nil(t, decoded.NestedPtr)
}

// Test empty slices
type SliceTypes struct {
	SliceInt    []int    `db:"slice_int"`
	SliceString []string `db:"slice_string"`
	SliceNested []NestedStruct `db:"slice_nested"`
}

func TestEmptySlices(t *testing.T) {
	registry := codec.NewCodecRegistry()
	registry.RegisterTypes(SliceTypes{}, NestedStruct{})

	original := SliceTypes{
		SliceInt:    []int{},
		SliceString: []string{},
		SliceNested: []NestedStruct{},
	}

	// Encode
	encoded, err := registry.Encode(&original)
	require.NoError(t, err)

	// Verify empty slices are encoded as empty slices (not nil)
	assert.Equal(t, []any{}, encoded["slice_int"])
	assert.Equal(t, []any{}, encoded["slice_string"])
	assert.Equal(t, []any{}, encoded["slice_nested"])

	// Decode
	var decoded SliceTypes
	err = registry.Decode(encoded, &decoded)
	require.NoError(t, err)

	assert.Len(t, decoded.SliceInt, 0)
	assert.Len(t, decoded.SliceString, 0)
	assert.Len(t, decoded.SliceNested, 0)
}

// Test complex nested structures
type Address struct {
	Street string `db:"street"`
	City   string `db:"city"`
	Zip    string `db:"zip"`
}

type Company struct {
	Name    string `db:"name"`
	Address Address `db:"address"`
}

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

// Test slice of primitive types with various values
func TestSlicePrimitives(t *testing.T) {
	registry := codec.NewCodecRegistry()
	
	type SliceContainer struct {
		Ints    []int    `db:"ints"`
		Strings []string `db:"strings"`
		Floats  []float64 `db:"floats"`
		Bools   []bool   `db:"bools"`
	}
	registry.RegisterTypes(SliceContainer{})

	original := SliceContainer{
		Ints:    []int{-1, 0, 1, 42, 999},
		Strings: []string{"a", "b", "c", "hello world"},
		Floats:  []float64{0.0, 3.14, -2.71, 1.41},
		Bools:   []bool{true, false, true},
	}

	// Encode
	encoded, err := registry.Encode(&original)
	require.NoError(t, err)

	ints := encoded["ints"].([]any)
	strings := encoded["strings"].([]any)
	floats := encoded["floats"].([]any)
	bools := encoded["bools"].([]any)

	assert.Equal(t, int64(-1), ints[0])
	assert.Equal(t, int64(999), ints[4])
	assert.Equal(t, "hello world", strings[3])
	assert.InDelta(t, 3.14, floats[1], 0.01)
	assert.Equal(t, false, bools[1])

	// Decode
	var decoded SliceContainer
	err = registry.Decode(encoded, &decoded)
	require.NoError(t, err)

	assert.Equal(t, original, decoded)
}

// Test pointer fields with values
func TestPointerFields(t *testing.T) {
	registry := codec.NewCodecRegistry()

	type WithPointers struct {
		PtrString *string `db:"ptr_string"`
		PtrInt    *int    `db:"ptr_int"`
		PtrFloat  *float64 `db:"ptr_float"`
		PtrBool   *bool   `db:"ptr_bool"`
	}
	registry.RegisterTypes(WithPointers{})

	str := "hello"
	i := 42
	f := 3.14
	b := true

	original := WithPointers{
		PtrString: &str,
		PtrInt:    &i,
		PtrFloat:  &f,
		PtrBool:   &b,
	}

	// Encode
	encoded, err := registry.Encode(&original)
	require.NoError(t, err)

	assert.Equal(t, "hello", encoded["ptr_string"])
	assert.Equal(t, int64(42), encoded["ptr_int"])
	assert.InDelta(t, 3.14, encoded["ptr_float"], 0.01)
	assert.Equal(t, true, encoded["ptr_bool"])

	// Decode
	var decoded WithPointers
	err = registry.Decode(encoded, &decoded)
	require.NoError(t, err)

	assert.Equal(t, *original.PtrString, *decoded.PtrString)
	assert.Equal(t, *original.PtrInt, *decoded.PtrInt)
	assert.InDelta(t, *original.PtrFloat, *decoded.PtrFloat, 0.01)
	assert.Equal(t, *original.PtrBool, *decoded.PtrBool)
}

// Test deeply nested structures
type Level3 struct {
	Value string `db:"value"`
}

type Level2 struct {
	Level3 Level3 `db:"level3"`
	Name   string `db:"name"`
}

type Level1 struct {
	Level2 Level2 `db:"level2"`
	ID     int    `db:"id"`
}

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

// Test multiple embedded structs
type Metadata struct {
	CreatedAt time.Time `db:"created_at"`
	UpdatedAt time.Time `db:"updated_at"`
}

type Status struct {
	Active bool `db:"active"`
}

type MultiEmbed struct {
	Name     string `db:"name"`
	Metadata `db:",embed"`
	Status   `db:",embed"`
}

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
	assert.Equal(t, original.Status.Active, decoded.Status.Active)
	assert.Equal(t, original.Metadata.CreatedAt, decoded.Metadata.CreatedAt)
	assert.Equal(t, original.Metadata.UpdatedAt, decoded.Metadata.UpdatedAt)
}

// Test EncodeValue with various types
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

// Test decode errors
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

// Test lazily registered types
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