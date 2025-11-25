package codec_test

import (
	"time"

	"github.com/neo4j/neo4j-go-driver/v5/neo4j"
)

// Shared test fixtures used across all codec tests

// Basic nested struct
type NestedStruct struct {
	Val string `neo4j:"val"`
}

// Comprehensive struct with all field types
type TestStruct struct {
	Int    int     `neo4j:"int"`
	String string  `neo4j:"string"`
	Bool   bool    `neo4j:"bool"`
	Float  float64 `neo4j:"float"`
	PtrInt *int    `neo4j:"ptr_int"`
	SliceInt []int `neo4j:"slice_int"`
	SliceStr []string `neo4j:"slice_str"`
	Nested   NestedStruct  `neo4j:"nested"`
	PtrNested *NestedStruct `neo4j:"ptr_nested"`
	EmbeddedStruct `neo4j:",embed"`
	Time  time.Time     `neo4j:"time"`
	Point neo4j.Point2D `neo4j:"point"`
}

type EmbeddedStruct struct {
	EmbedVal int `neo4j:"embed_val"`
}

type PrimitiveTypes struct {
	Int8    int8    `neo4j:"int8"`
	Int16   int16   `neo4j:"int16"`
	Int32   int32   `neo4j:"int32"`
	Int64   int64   `neo4j:"int64"`
	Uint    uint    `neo4j:"uint"`
	Uint8   uint8   `neo4j:"uint8"`
	Uint16  uint16  `neo4j:"uint16"`
	Uint32  uint32  `neo4j:"uint32"`
	Uint64  uint64  `neo4j:"uint64"`
	Float32 float32 `neo4j:"float32"`
	Bytes   []byte  `neo4j:"bytes"`
}

type NilableTypes struct {
	PtrString *string       `neo4j:"ptr_string"`
	PtrInt    *int          `neo4j:"ptr_int"`
	SliceVal  []string      `neo4j:"slice_val"`
	NestedPtr *NestedStruct `neo4j:"nested_ptr"`
}

type SliceTypes struct {
	SliceInt    []int          `neo4j:"slice_int"`
	SliceString []string       `neo4j:"slice_string"`
	SliceNested []NestedStruct `neo4j:"slice_nested"`
}

type Address struct {
	Street string `neo4j:"street"`
	City   string `neo4j:"city"`
	Zip    string `neo4j:"zip"`
}

type Company struct {
	Name    string  `neo4j:"name"`
	Address Address `neo4j:"address"`
}

type Level3 struct {
	Value string `neo4j:"value"`
}

type Level2 struct {
	Level3 Level3 `neo4j:"level3"`
	Name   string `neo4j:"name"`
}

type Level1 struct {
	Level2 Level2 `neo4j:"level2"`
	ID     int    `neo4j:"id"`
}

type Metadata struct {
	CreatedAt time.Time `neo4j:"created_at"`
	UpdatedAt time.Time `neo4j:"updated_at"`
}

type Status struct {
	Active bool `neo4j:"active"`
}

type MultiEmbed struct {
	Name     string `neo4j:"name"`
	Metadata `neo4j:",embed"`
	Status   `neo4j:",embed"`
}

type WithPointers struct {
	PtrString *string  `neo4j:"ptr_string"`
	PtrInt    *int     `neo4j:"ptr_int"`
	PtrFloat  *float64 `neo4j:"ptr_float"`
	PtrBool   *bool    `neo4j:"ptr_bool"`
}

type SliceContainer struct {
	Ints    []int     `neo4j:"ints"`
	Strings []string  `neo4j:"strings"`
	Floats  []float64 `neo4j:"floats"`
	Bools   []bool    `neo4j:"bools"`
}

type Wrapper struct {
	Nested *NestedStruct `neo4j:"nested"`
}

type Container struct {
	Items []*NestedStruct `neo4j:"items"`
}

type AllNeo4jTypes struct {
	Point neo4j.Point2D `neo4j:"point"`
}

type WithInterface struct {
	Data any `neo4j:"data"`
}

type WithBools struct {
	True  bool `neo4j:"true"`
	False bool `neo4j:"false"`
}

type Boundaries struct {
	MaxInt64   int64   `neo4j:"max_int64"`
	MinInt64   int64   `neo4j:"min_int64"`
	MaxFloat64 float64 `neo4j:"max_float64"`
	MinFloat64 float64 `neo4j:"min_float64"`
}

type IntTypes struct {
	Int    int    `neo4j:"int"`
	Int64  int64  `neo4j:"int64"`
	Uint   uint   `neo4j:"uint"`
	Uint32 uint32 `neo4j:"uint32"`
}

type WithSkip struct {
	Name     string `neo4j:"name"`
	Internal string `neo4j:"-"`
	Other    string `neo4j:"other"`
}

// NOTE: Maps are no longer supported for Neo4j codec
// The data model only supports scalar properties
// Keeping these types for reference but they cannot be used with the codec
// type WithMaps struct {
// 	StringMap map[string]string `neo4j:"string_map"`
// 	IntMap    map[string]int    `neo4j:"int_map"`
// }

// Nil pointer fields
type AllNilPtrs struct {
	PtrStr *string    `neo4j:"ptr_str"`
	PtrInt *int       `neo4j:"ptr_int"`
	PtrFlt *float64   `neo4j:"ptr_flt"`
	PtrBol *bool      `neo4j:"ptr_bol"`
	PtrNst *NestedStruct `neo4j:"ptr_nst"`
}

// Recursive/circular reference handling
type TreeNode struct {
	Value    int        `neo4j:"value"`
	Children []*TreeNode `neo4j:"children"`
}

// Complex nested with all features
type ComplexStruct struct {
	Simple    int            `neo4j:"simple"`
	Nested    NestedStruct   `neo4j:"nested"`
	PtrNested *NestedStruct  `neo4j:"ptr_nested"`
	SliceNest []NestedStruct `neo4j:"slice_nested"`
	Status    `neo4j:",embed"`
}

// Test overflow cases
type NumericOverflow struct {
	SmallInt  int8   `neo4j:"small_int"`
	SmallUint uint8  `neo4j:"small_uint"`
}

// Test special float values
type FloatSpecial struct {
	NormalFloat float64 `neo4j:"normal_float"`
}

// Mutliple fields of same primitive type
type SamePrimitive struct {
	A int `neo4j:"a"`
	B int `neo4j:"b"`
	C int `neo4j:"c"`
}

// Empty struct
type EmptyStruct struct {
}

// Only embedded
type OnlyEmbedded struct {
	Status `neo4j:",embed"`
}

// Mix of multiple embedded with overlapping names (should merge)
type MultiEmbedWithMeta struct {
	Name     string    `neo4j:"name"`
	Metadata `neo4j:",embed"`
	Status   `neo4j:",embed"`
}
