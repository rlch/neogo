# Codec Package - Zero-Reflection Serialization

The codec package implements a high-performance, zero-reflection serialization/deserialization system inspired by `goccy/go-json`. It compiles type information into optimized opcode sequences and decoder trees, enabling fast encode/decode operations without runtime reflection.

## Architecture

### 1. **Encoding Path** (Opcode VM)

```
Type Registration
    ↓
Compiler.Compile(type)
    ↓
Opcode Tree (linked-list VM)
    ↓
EncodeStruct() → Iterate Opcodes
    ↓
EncodeAny() → Execute Opcode
    ↓
map[string]any (Neo4j-compatible output)
```

**Key Components:**
- **`compiler.go`**: Recursively compiles Go types into opcode sequences
- **`opcodes.go`**: Defines opcode types and execution logic
  - `OpFieldInt`, `OpFieldString`, `OpFieldBool`, etc. (primitives)
  - `OpFieldSlice`, `OpFieldMap`, `OpFieldStruct` (complex types)
  - `OpFieldPtr` (pointer dereferencing)
  - `OpFieldEmbed` (embedded struct flattening)
  - `OpFieldTime`, `OpFieldPoint2D` (Neo4j types)
- **`EncodeStruct(opcode, ptr, result)`**: Main encoding entry point
  - Uses `unsafe` pointers to directly read struct field values
  - Zero allocations for primitives
  - Recursive handling of nested structures

### 2. **Decoding Path** (Decoder Tree)

```
Type Registration
    ↓
Compiler.CompileDecoder(type)
    ↓
Decoder Tree (interface-based)
    ↓
structDecoder → fieldDecoder → elementDecoder
    ↓
Decode(data, ptr) → Recursively decode
    ↓
Populate struct fields using unsafe pointers
```

**Key Components:**
- **`decoder.go`**: Implements decoder interfaces for all types
  - `structDecoder`: Iterates over field map, calls field decoders
  - `sliceDecoder`: Allocates slice, decodes elements
  - `mapDecoder`: Creates map, decodes key-value pairs
  - `ptrDecoder`: Allocates memory, decodes into it
  - Primitive decoders: `intDecoder`, `stringDecoder`, `boolDecoder`, etc.
  - Special decoders: `bytesDecoder`, `timeDecoder`

### 3. **Type Metadata**

**Compiler Caching:**
- Encoders cached by `TypePtr` (runtime type pointer) → `*Opcode`
- Decoders cached by `TypePtr` → `Decoder`
- Circular references handled with early cache registration

**Field Information:**
- `FieldInfo`: Parsed from struct tags (`neo4j:"..."`)
  - `DBName`: Database field name
  - `IsSkip`: Field excluded from serialization
  - `IsEmbed`: Embedded struct flattened into parent
  - Options: `embed`, `skip`, custom codecs (future)

## Key Features

### Zero-Reflection Performance
- No runtime reflection during encode/decode
- Type inspection happens once at registration
- Direct memory access via `unsafe` pointers
- Opcode execution is tight loop with minimal allocations

### Embedded Struct Handling
```go
type Address struct {
    Street string `neo4j:"street"`
    City   string `neo4j:"city"`
}

type Person struct {
    Name    string `neo4j:"name"`
    Address Address `neo4j:",embed"`  // Fields merged at top level
}

// Encoded as: {"name": "...", "street": "...", "city": "..."}
```

Decoder flattens embedded struct fields by:
1. Recursively compiling embedded type
2. Extracting field map from nested decoder
3. Adjusting field offsets for parent struct
4. Merging into parent field map

### Bytes Special Handling
`[]byte` is treated as a primitive (not a generic slice):
- Encoded as `[]byte` directly
- Decoded with `bytesDecoder` (accepts `[]byte` or `string`)
- Avoids slice allocations for byte sequences

### Type Coercion
Decoders handle flexible type conversion:
- `int64` ↔ `int`, `int8`, `int16`, `int32` (with bounds checking)
- `float64` ↔ `float32` (with precision loss)
- `string` ↔ `[]byte`
- Neo4j types ↔ Go types

## Testing

**23 comprehensive test functions** covering:

### Core Functionality
- `TestRoundTrip`: Complete struct with all field types
- `TestSliceOfStructs`: Nested struct slices
- `TestPrimitiveTypes`: All int/uint/float variants
- `TestNestedStructs`: Multi-level nesting
- `TestMultipleEmbedded`: Multiple embedded structs

### Type Coverage
- `TestDeeplyNestedStructs`: 3-level nesting
- `TestSlicePrimitives`: Slices of primitives
- `TestPointerFields`: Pointers with dereferencing
- `TestSliceOfPointers`: Slices of pointers
- `TestNeo4jTypes`: Point2D encoding/decoding
- `TestInterfaceField`: `interface{}` fields

### Edge Cases
- `TestNilValues`: Nil pointers and slices
- `TestEmptySlices`: Empty slice handling
- `TestNumericBoundaries`: Max/min int64, float64
- `TestTypeCoercion`: Int type conversions
- `TestFieldSkip`: Fields marked `neo4j:"-"`
- `TestDecodeErrors`: Error handling
- `TestLazyRegistration`: Unregistered types

### Encoding/Decoding
- `TestEncodeValue`: Various primitive types
- `TestEncodeValueSlices`: Slice encoding
- `TestEncodeValueZeros`: Zero values
- `TestPointerToStruct`: Nested pointers
- `TestBoolFields`: Boolean encoding

**Coverage**: 48.7% of statements

## File Structure

```
codec/
├── opcodes.go              # Opcode definition and encoding VM
├── compiler.go             # Type compiler for opcodes/decoders
├── decoder.go              # Decoder implementations
├── registry.go             # Type registry and API
├── tags.go                 # Struct tag parsing (neo4j:"...")
├── typeptr.go              # Type information and conversions
├── codec_test.go           # Comprehensive test suite
├── neo4j_extractor.go      # Neo4j metadata extraction (legacy)
├── relationship_extractor.go # Relationship handling (legacy)
├── abstract_extractor.go   # Abstract type handling (legacy)
├── reflection_helpers.go   # Reflection utilities (legacy)
└── README.md               # This file
```

## Usage

```go
// Register types
registry := codec.NewCodecRegistry()
registry.RegisterTypes(MyStruct{}, NestedStruct{})

// Encode
data := &MyStruct{...}
encoded, err := registry.Encode(data)

// Decode
var decoded MyStruct
err = registry.Decode(encoded, &decoded)

// Encode arbitrary values
val, err := registry.EncodeValue([]string{"a", "b"})
```

## Future Improvements

1. **Reach 100% coverage**: Test remaining paths
   - Error conditions in decoders
   - Registry lazy compilation
   - Neo4j type decoders

2. **Custom codec support**: `neo4j:"codec:mycodec"` tag option

3. **Map encoding**: Support `map[string]T` encoding

4. **Performance benchmarks**: Compare with reflection-based approaches

## References

- Inspired by: [goccy/go-json](https://github.com/goccy/go-json)
- Runtime pointer safety: [unsafe.Pointer rules](https://pkg.go.dev/unsafe)
