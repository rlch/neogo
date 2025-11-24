# Neogo Internal Architecture: Serialization & Deserialization

This document outlines the design for `neogo`'s zero-reflection runtime, inspired by [goccy/go-json](https://github.com/goccy/go-json) but adapted for Neo4j's specific data types and structures.

## 1. Design Philosophy

To achieve high performance, we avoid Go's `reflect` package in hot paths. Instead, we perform reflection **once** during the registration phase to build optimized "programs" (opcodes) that are executed at runtime using `unsafe` pointer arithmetic.

### Key Components
1.  **Registry**: Stores pre-compiled encoders and decoders for registered types.
2.  **Encoder**: A linked-list virtual machine (VM) that converts Go structs to `map[string]any` (for Neo4j parameters).
3.  **Decoder**: A compiled set of field handlers that populates Go structs from `map[string]any` or `dbtype.Node`.

---

## 2. Neo4j-Go-Driver Data Mapping

The `neo4j-go-driver` uses the Bolt protocol (PackStream). Our ORM must map between Go structs and the driver's supported runtime types.

| Go Type | Neo4j Driver Type | Neogo Opcode Strategy |
|:--- |:--- |:--- |
| `int`, `int8`...`int64` | `int64` | Cast & assign |
| `float32`, `float64` | `float64` | Cast & assign |
| `bool` | `bool` | Direct assignment |
| `string` | `string` | Direct assignment |
| `[]byte` | `[]byte` | Direct assignment |
| `[]T` | `[]any` | Loop & recurse |
| `map[K]V` | `map[string]any` | Loop & recurse (Keys must be strings) |
| `time.Time` | `dbtype.Time`, `dbtype.Date`, etc. | Wrap/Unwrap driver types |
| `struct` | `map[string]any` (Properties) | Recursive encoding/decoding |

---

## 3. Encoder Architecture (Struct -> Map)

Following `goccy/go-json`, the encoder uses a linked list of **Opcodes**. This allows us to "run" a struct serialization as a sequence of CPU-friendly instructions without lookups.

### Opcode Structure

```go
type Opcode struct {
    Op         OpType       // The operation (e.g., OpInt, OpString)
    Offset     uintptr      // Memory offset of the field in the struct
    Key        string       // The key name to use in the output map
    Next       *Opcode      // Next operation in the chain
    End        *Opcode      // For nested structures/lists, where to jump after
    SubOpcodes *Opcode      // For nested structs/slices
}
```

### Required Encoder Opcodes

We need a simplified set compared to a JSON library, as we output a `map` rather than a byte stream.

#### Primitives
- `OpInt`, `OpInt8`, `OpInt16`, `OpInt32`, `OpInt64`
- `OpUint`, `OpUint8`... (Note: Neo4j only supports signed 64-bit integers; large uint64s may overflow)
- `OpFloat32`, `OpFloat64`
- `OpBool`
- `OpString`
- `OpBytes`

#### Complex Types
- `OpSlice`: Iterates a slice header, applying `SubOpcodes` to each element.
- `OpMap`: Iterates a map, encoding keys and values.
- `OpStruct`: Recurses into an embedded struct or nested object.
- `OpPtr`: Handles nil checks. If not nil, dereferences and proceeds.
- `OpInterface`: Fallback that passes the value directly (letting the driver handle it).

#### Neo4j Specifics
- `OpTime`, `OpDate`, `OpDuration`: Handles conversion to `dbtype.*` structs.
- `OpPoint2D`, `OpPoint3D`: Handles spatial types.

### Execution Loop (Pseudo-code)

```go
func Encode(head *Opcode, ptr unsafe.Pointer) map[string]any {
    out := make(map[string]any)
    for op := head; op != nil; op = op.Next {
        val := op.Execute(ptr + op.Offset)
        out[op.Key] = val
    }
    return out
}
```

---

## 4. Decoder Architecture (Map/Node -> Struct)

For decoding, we don't need a linked list. We receive a map (random access) and need to populate a struct. A **Field Map** approach is most efficient.

### Structure

Instead of a VM, we compile a `Decoder` for each type which holds a mapping of `Property Name -> FieldHandler`.

```go
type StructDecoder struct {
    Fields map[string]FieldHandler
}

type FieldHandler struct {
    Offset  uintptr     // Where to write in the struct
    Decoder Decoder     // How to decode the value (IntDecoder, StringDecoder, etc.)
}
```

### Required Decoders

#### Primitives
- `IntDecoder`: Reads `int64`, casts to specific int type, writes to memory.
- `FloatDecoder`: Reads `float64`, casts, writes.
- `StringDecoder`: Reads `string`, writes.
- `BoolDecoder`: Reads `bool`, writes.

#### Complex Types
- `SliceDecoder`: Expects `[]any`. Allocates slice of correct size, iterates, and uses a generic `ElemDecoder` for items.
- `StructDecoder`: Expects `map[string]any` (or `dbtype.Node`). Recurses using the pre-compiled `Fields` map.
- `PtrDecoder`: Allocates new memory if the map value is not nil.

#### Neo4j Specifics
- `NodeDecoder`: Handles `dbtype.Node`. Extracts `Props` map and `ElementId`.
- `RelationshipDecoder`: Handles `dbtype.Relationship`.
- `TimeDecoder`: Converts `dbtype.Time` back to `time.Time`.

### Execution Flow (Pseudo-code)

```go
func (d *StructDecoder) Decode(data map[string]any, ptr unsafe.Pointer) error {
    for key, handler := range d.Fields {
        if val, ok := data[key]; ok {
            // Fast path: value exists
            if err := handler.Decoder.Decode(val, ptr + handler.Offset); err != nil {
                return err
            }
        }
    }
    return nil
}
```

---

## 5. Implementation Roadmap

1.  **Goccy Vendoring**: We have vendored `goccy/go-json` to reference its optimized `unsafe` handling patterns.
2.  **Opcodes Package**: Create `internal/codec/opcodes.go` to define the simplified instruction set.
3.  **Compiler**: Create `internal/codec/compiler.go` to inspect types via reflection and generate opcodes.
4.  **Runtime**: Implement the `Encode` and `Decode` hot paths using `unsafe`.
5.  **Integration**: Hook this system into `client_impl.go` to replace the current `json.Marshal` hack.
