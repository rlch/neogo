# Reflection Removal Progress

This document tracks progress on removing runtime reflection from hot paths in neogo.

## Goals

1. **Hot path (per-query/record)**: Zero reflection - use pre-compiled codecs with unsafe pointers
2. **Registration phase (startup)**: Reflection OK - happens once at startup
3. **Lookup phase (once per type)**: Minimal reflection OK - cache lookup uses `reflect.TypeOf`

## Current State

The codec system (`internal/codec/`) already implements zero-reflection decoding via:
- Pre-compiled decoders (`CompileDecoder`)
- Unsafe pointer arithmetic for field access
- Type-specific allocators that avoid runtime `reflect.New()`

However, `internal/binding.go` still uses heavy reflection on the hot path.

## Architecture Challenge

The current binding system has two phases:
1. **Query building** (`scope.go`): Tracks `map[string]reflect.Value` bindings
2. **Unmarshaling** (`client_impl.go`): Uses those bindings to populate user structs

The challenge is that `client_impl.go` does significant reflection work:
- `unmarshalRecords()`: Creates slice, gets index, allocates elements
- `unmarshalRecord()`: Passes `reflect.Value` to `BindValue`

### Incremental Approach

Rather than rewriting everything at once, we can:

1. **Keep bindings as `reflect.Value`** for now (query building isn't hot path)
2. **Optimize `BindValue`** to delegate to codec ASAP (reduce reflection per-record)
3. **Later**: Refactor bindings to use typed wrappers that pre-compile allocation

## Files to Refactor

### High Priority (Hot Path)

| File | Status | Notes |
|------|--------|-------|
| `internal/binding.go` | 🔴 TODO | 300 lines of reflection-heavy code. Main `BindValue` function called per-record |
| `internal/binding_test.go` | 🔴 TODO | Tests for binding.go - update to test new API |

### Medium Priority (Query Building)

| File | Status | Notes |
|------|--------|-------|
| `internal/scope.go` | 🟡 REVIEW | Uses `reflect.Value` for bindings map. May need to change to `unsafe.Pointer` or `any` |
| `internal/cypher.go` | 🟡 REVIEW | `Bindings map[string]reflect.Value` - evaluate if can use `any` instead |
| `internal/cypher_client.go` | 🟡 REVIEW | Uses bindings from cypher.go |
| `client_impl.go` | 🟡 REVIEW | Calls `BindValue` - needs to adapt to new API |

### Low Priority (Registration Only)

| File | Status | Notes |
|------|--------|-------|
| `internal/registry.go` | ✅ OK | Registration phase - reflection acceptable |
| `internal/labels.go` | ✅ OK | `ExtractNodeLabels`/`ExtractRelationshipType` - registration phase |
| `internal/helpers.go` | ✅ OK | `WalkStruct` - used at registration, not hot path |
| `internal/codec/neo4j_extractor.go` | ✅ OK | Registration phase metadata extraction |
| `internal/codec/compiler.go` | ✅ OK | Compile-time only |
| `internal/codec/registry.go` | ✅ OK | Registration + lookup (cached) |

## Plan for binding.go Removal

### Current API
```go
func (r *Registry) BindValue(from any, to reflect.Value) error
```

### Target API
```go
func (r *Registry) Bind(from any, to any) error  // to must be pointer
```

### Migration Steps

1. **Phase 1**: Add new `Bind(from any, to any)` method that uses codec.Decode
2. **Phase 2**: Handle special cases in codec:
   - `neo4j.Node` → extract Props, decode to struct
   - `neo4j.Relationship` → extract Props, decode to struct  
   - Slice depth matching (single record to slice)
   - Abstract node binding (polymorphic decode)
   - Valuer interface support
3. **Phase 3**: Update callers (`client_impl.go`) to use new API
4. **Phase 4**: Remove old `BindValue` and related reflection code

## Special Cases to Handle

### 1. Neo4j Node/Relationship Unwrapping
Currently handled in binding.go:
```go
case neo4j.Node:
    return r.BindValue(fromVal.Props, to)
```
**Solution**: Codec decoder already handles this in `structDecoder.Decode()`

### 2. Slice Depth Matching
```go
// If depth of from is 1 lower than to, wrap in slice
if fromDepth+1 == toDepth {
    to.Set(reflect.MakeSlice(toT, 1, 1))
    return r.BindValue(from, to.Index(0))
}
```
**Solution**: Add to slice decoder or handle in top-level Bind()

### 3. Abstract Node Binding
```go
func (r *Registry) BindAbstractNode(node neo4j.Node, to reflect.Value) error
```
**Solution**: Need label-based type lookup in codec registry

### 4. Valuer Interface
```go
type Valuer[V neo4j.RecordValue] interface {
    Marshal() (*V, error)
    Unmarshal(*V) error
}
```
**Solution**: Check for interface at registration, compile custom decoder

### 5. Primitive Coercion (via cast library)
```go
case int:
    return true, bindCasted(cast.ToIntE, from, value)
```
**Solution**: Codec decoders already handle type conversion

## Scope/Bindings Refactor

The `map[string]reflect.Value` in scope.go is used for:
1. Tracking bound variables during query building
2. Passing bindings to unmarshaling

Options:
- **Option A**: Keep `reflect.Value` in scope (query building isn't hot path)
- **Option B**: Change to `map[string]any` and use type assertions
- **Option C**: Change to `map[string]unsafe.Pointer` with type info

**Recommendation**: Option A for now - query building happens once per query, not per record.

## Progress Log

- [x] Phase 1: Simplify binding.go to delegate to codec ASAP
  - Removed ~100 lines of reflection code
  - Kept Valuer interface support
  - Added primitive binding helper
  - Fixed nil handling
- [x] Phase 2: Handle special cases
  - Neo4j Node/Relationship unwrapping (codec handles this)
  - Slice depth matching (kept in binding.go for now)
  - Single value to slice wrapping
  - Abstract node binding (polymorphic)
  - Nil pointer preservation in slices
- [x] Phase 3: Update client_impl.go
  - Fixed nil pre-allocation issue
- [ ] Phase 4: Further simplify binding.go (future work)
- [ ] Phase 5: Evaluate scope.go bindings map (future work)

## Code Stats

**Before refactor:**
- `binding.go`: ~300 lines with heavy reflection
- Used `cast` library for type coercion

**After refactor:**
- `binding.go`: ~350 lines but cleaner structure
- Delegates to codec for struct decoding (zero reflection hot path)
- Primitives handled with direct type matching
- Removed dependency on `cast` library for most paths
