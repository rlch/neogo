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
| `internal/binding.go` | 🟢 OPTIMIZED | Reflection still used for special cases (Valuer, abstract), but fast path skips it |
| `internal/binding_plan.go` | 🟢 NEW | Pre-compiles binding metadata at query compile time, provides zero-reflection DecodeSingle/DecodeMultiple |
| `client_impl.go` | 🟢 OPTIMIZED | Uses BindingPlan for fast path, falls back to Bind() for special cases |

### Medium Priority (Query Building)

| File | Status | Notes |
|------|--------|-------|
| `internal/scope.go` | ✅ DONE | Changed from `reflect.Value` to `map[string]any` (pointers) |
| `internal/cypher.go` | ✅ DONE | `Bindings map[string]any` - pointers to user's binding targets |
| `internal/cypher_client.go` | ✅ DONE | Uses bindings from cypher.go |

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
- [x] Phase 4: Scope bindings migration from reflect.Value to any
  - Changed `bindings` from `map[string]reflect.Value` to `map[string]any` (pointers)
  - Changed `names` from `map[reflect.Value]string` to `map[uintptr]string` (pointer addresses)
  - Added `ptrAddr()` helper for pointer address extraction
  - Fixed field/name lookup priority for zero-size embedded types
    - Primitive pointers (string, int, etc.) check fields first
    - Struct/interface pointers check names first
  - Updated `propertyIdentifier` and `valueIdentifier` with same logic
  - Fixed WHERE clause identifier to use name string instead of raw identifier
  - Delete field entry when aliasing to prevent stale lookups
- [x] Phase 5: BindingPlan - Pre-compile binding metadata
  - Added `BindingPlan` struct in `internal/binding_plan.go`
  - Pre-computes at query compile time (not per-record):
    - IsSlice, IsAbstract, IsSliceAbstract flags
    - SliceDepth, PointerDepth
    - Decoder (pre-compiled from codec)
    - SliceAllocator, ElemAllocator functions
  - Added `Plans map[string]*BindingPlan` to `CompiledCypher`
  - Plans built in `CypherRunner.Compile()` via `BuildBindingPlans()`
  - `unmarshalRecords` uses plan flags instead of runtime reflection
- [x] Phase 6: Use plan.Decoder directly in hot path (skip BindValue)
  - Added `DecodeSingle(value any)` method to BindingPlan
    - Uses pre-compiled decoder directly via unsafe pointer
    - ZERO reflection for non-abstract, non-Valuer types
    - Falls through to Bind() for special cases
  - Added `DecodeMultiple(values []any)` method to BindingPlan
    - Uses pre-compiled SliceAllocator + decoder
    - Batch decodes multiple records efficiently
  - Updated `unmarshalRecord()` in client_impl.go:
    - Uses `plan.DecodeSingle()` for non-slice bindings
    - Falls back to `Bind()` only for abstract/special types
  - Updated `unmarshalRecords()` in client_impl.go:
    - Uses `plan.DecodeMultiple()` first for slice bindings
    - Simplified: removed legacy path (plans always available)
  - Updated `unmarshalRecordsFallback()`:
    - Uses `plan.SliceAllocator` instead of `reflect.MakeSlice`
    - Uses `plan.ElemAllocator` instead of `reflect.New` for pointer elements
  - Added `getTargetPtr()` helper for unwrapping pointer chains
  - **Removed legacy code**:
    - Deleted `normalizeSliceBinding()` - replaced by plan.DecodeMultiple
    - Deleted `isAbstractSliceBinding()` - replaced by plan.IsSliceAbstract flag
    - Simplified `unmarshalRecords()` control flow
- [ ] Phase 7: Add HasValuer detection at plan creation time (future work)
  - Would allow skipping Valuer interface check per-record
  - Low priority since Valuer is rarely used
- [x] Phase 8: Further simplify binding.go
  - Consolidated `computeDepth` and `computeSliceDepth` (removed duplicate)
  - Consolidated `isAbstractTarget` to delegate to `isAbstractType`
  - Removed redundant `rAbstract` local variable (uses package-level from registry.go)
  - Updated comments to accurately describe reflection boundaries
- [x] Phase 9: Evaluate remaining reflection in query building
  - **Conclusion**: Query building reflection is ACCEPTABLE (not hot path)
  - Added documentation headers to scope.go, binding.go, binding_plan.go
  - Clarified reflection boundaries in comments
  - No changes needed - query building happens once per query, not per-record
- [x] Phase 10: Cache target pointer at plan creation time
  - Added `CachedTargetPtr` field to BindingPlan
  - Renamed `getTargetPtr()` to `computeTargetPtr()` - called once at plan creation
  - `DecodeSingle()` now uses cached pointer directly - ZERO reflection per-record
  - `DecodeMultiple()` uses cached pointer for slice header
  - Added `TestBindingPlan_CachedTargetPtr` test suite
  - **Result**: Eliminated `reflect.ValueOf()` calls from hot path

## Code Stats

**Before refactor:**
- `binding.go`: ~300 lines with heavy reflection
- `scope.go`: Used `reflect.Value` for bindings map
- Used `cast` library for type coercion
- Every record decode used reflection per-field

**After Phase 8-9 cleanup:**
- `binding.go`: ~340 lines, reflection only for special cases (Valuer, abstract, slice depth)
- `binding_plan.go`: ~270 lines, pre-compiled binding plans with accurate doc comments
- `scope.go`: Uses `map[string]any` with `map[uintptr]string` for reverse lookup
- `client_impl.go`: Uses BindingPlan fast path, falls back to Bind() for special cases
- All files have clear documentation of reflection boundaries

**Hot path improvements:**
- Non-abstract, non-Valuer bindings: ZERO reflection per-record
- Target pointer: Cached at plan creation (no `reflect.ValueOf` per-record)
- Slice allocations: Pre-compiled allocators (per-batch, not per-record)
- Pointer element allocations: Pre-compiled allocators (fallback path only)
- Type checking: Plan flags instead of `reflect.Kind()`, `reflect.Implements()`
- Decoder lookup: Pre-compiled at query compile time (not per-record)

## Final Architecture

```
Query Building (reflection OK - once per query)
    scope.go          -> Tracks bindings, generates Cypher
    cypher.go         -> Builds query string
    cypher_client.go  -> Compiles query, builds BindingPlans

Record Decoding (zero reflection - per record)
    binding_plan.go   -> Pre-compiled plans with flags & decoders
    codec/decoder.go  -> Zero-reflection decoders using unsafe
    codec/registry.go -> Decoder lookup (cached)

Fallback Path (reflection - special cases only)
    binding.go        -> Valuer, abstract nodes, slice depth mismatch
```
