package codec_test

import (
	"time"

	"github.com/neo4j/neo4j-go-driver/v5/neo4j"
)

// Common test struct types reused across multiple test files

// Basic nested struct
type NestedStruct struct {
	Val string `db:"val"`
}

// Comprehensive struct with all field types
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

// Embedded struct for flattening tests
type EmbeddedStruct struct {
	EmbedVal int `db:"embed_val"`
}

// Primitive types test struct
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

// Nullable types
type NilableTypes struct {
	PtrString *string       `db:"ptr_string"`
	PtrInt    *int          `db:"ptr_int"`
	SliceVal  []string      `db:"slice_val"`
	NestedPtr *NestedStruct `db:"nested_ptr"`
}

// Slice types
type SliceTypes struct {
	SliceInt    []int          `db:"slice_int"`
	SliceString []string       `db:"slice_string"`
	SliceNested []NestedStruct `db:"slice_nested"`
}

// Nested structures
type Address struct {
	Street string `db:"street"`
	City   string `db:"city"`
	Zip    string `db:"zip"`
}

type Company struct {
	Name    string  `db:"name"`
	Address Address `db:"address"`
}

// Deeply nested (3 levels)
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

// Multiple embedded structs
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

// Pointer types
type WithPointers struct {
	PtrString *string  `db:"ptr_string"`
	PtrInt    *int     `db:"ptr_int"`
	PtrFloat  *float64 `db:"ptr_float"`
	PtrBool   *bool    `db:"ptr_bool"`
}

// Slice containers
type SliceContainer struct {
	Ints    []int     `db:"ints"`
	Strings []string  `db:"strings"`
	Floats  []float64 `db:"floats"`
	Bools   []bool    `db:"bools"`
}

// Pointer to struct wrapper
type Wrapper struct {
	Nested *NestedStruct `db:"nested"`
}

// Slice of pointers
type Container struct {
	Items []*NestedStruct `db:"items"`
}

// Neo4j types
type AllNeo4jTypes struct {
	Point neo4j.Point2D `db:"point"`
}

// Interface field
type WithInterface struct {
	Data any `db:"data"`
}

// Bool fields
type WithBools struct {
	True  bool `db:"true"`
	False bool `db:"false"`
}

// Numeric boundaries
type Boundaries struct {
	MaxInt64   int64   `db:"max_int64"`
	MinInt64   int64   `db:"min_int64"`
	MaxFloat64 float64 `db:"max_float64"`
	MinFloat64 float64 `db:"min_float64"`
}

// Integer types for coercion
type IntTypes struct {
	Int    int    `db:"int"`
	Int64  int64  `db:"int64"`
	Uint   uint   `db:"uint"`
	Uint32 uint32 `db:"uint32"`
}

// Field skip test
type WithSkip struct {
	Name     string `db:"name"`
	Internal string `db:"-"`
	Other    string `db:"other"`
}
