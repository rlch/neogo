package internal

// binding_plan.go provides pre-compiled binding plans for zero-reflection decoding.
//
// # Architecture
//
// BindingPlan is created ONCE at query compile time (cold path) and captures:
//   - Pre-compiled codec decoder (from internal/codec)
//   - Type flags (IsSlice, IsAbstract, etc.)
//   - Allocator functions for slice/pointer elements
//
// At decode time (hot path), BindingPlan.DecodeSingle() and DecodeMultiple()
// use unsafe pointers and pre-compiled decoders with ZERO reflection per-record.
//
// # Reflection Boundaries
//
// COLD PATH (plan creation): Uses reflection to analyze types, create allocators
// HOT PATH (per-record decode): Zero reflection - uses unsafe pointers + pre-compiled decoders

import (
	"fmt"
	"reflect"
	"unsafe"

	"github.com/rlch/neogo/internal/codec"
)

// BindingPlan pre-computes everything needed to decode values for a specific binding.
// Created once at query compile time (cold path), used per-record without reflection (hot path).
type BindingPlan struct {
	// Key is the binding name (e.g., "n", "person.name")
	Key string

	// Target is the pointer to user's binding target
	Target any

	// TargetType is the reflect.Type of the target (cached to avoid reflect.TypeOf per record)
	TargetType reflect.Type

	// CachedTargetPtr is the pre-computed unsafe.Pointer to the target value.
	// For non-slice targets, this points directly to where data should be decoded.
	// Computed once at plan creation, used per-record without reflection.
	CachedTargetPtr unsafe.Pointer

	// Decoder is the pre-compiled codec decoder for the innermost type
	Decoder codec.Decoder

	// Flags computed at plan creation time (avoid per-record reflection)
	IsSlice         bool // Target is a slice (possibly nested in pointers)
	IsPointerElem   bool // Slice element type is a pointer
	IsAbstract      bool // Target type implements IAbstract interface
	IsSliceAbstract bool // Slice element implements IAbstract
	HasValuer       bool // Type implements Valuer interface (TODO: detect at registration)
	SliceDepth      int  // Nesting depth ([]T = 1, [][]T = 2)
	PointerDepth    int  // Number of pointer indirections to reach slice/value

	// Allocator for pointer elements (avoids reflect.New per record)
	// Created at plan time, uses unsafe.Pointer
	ElemAllocator func() unsafe.Pointer

	// SliceAllocator creates a slice of given length (avoids reflect.MakeSlice per record)
	SliceAllocator func(n int) any
}

// NewBindingPlan creates a BindingPlan for a binding target.
// This does reflection ONCE at plan creation, not per-record.
func NewBindingPlan(key string, target any, codecs *codec.CodecRegistry) *BindingPlan {
	plan := &BindingPlan{
		Key:    key,
		Target: target,
	}

	// Get type info (reflection at plan creation, not per-record)
	targetType := reflect.TypeOf(target)
	plan.TargetType = targetType

	// Unwrap pointer chain to find the actual value type
	innerType := targetType
	for innerType.Kind() == reflect.Ptr {
		plan.PointerDepth++
		innerType = innerType.Elem()
	}

	// Check if it's a slice
	if innerType.Kind() == reflect.Slice {
		plan.IsSlice = true
		plan.SliceDepth = computeDepth(innerType)

		// Check element type
		elemType := innerType.Elem()
		for elemType.Kind() == reflect.Slice {
			elemType = elemType.Elem()
		}

		plan.IsPointerElem = elemType.Kind() == reflect.Ptr
		plan.IsSliceAbstract = isAbstractType(elemType)

		// Create slice allocator
		plan.SliceAllocator = makeSliceAllocatorFunc(innerType)

		// Create element allocator if pointer elements
		if plan.IsPointerElem {
			plan.ElemAllocator = makePtrAllocatorFunc(elemType.Elem())
		}

		// Get decoder for element type (uses lazy compilation if not pre-registered)
		if !plan.IsSliceAbstract {
			// Create a zero value to get the decoder
			zeroVal := reflect.New(innerType).Interface()
			plan.Decoder = codecs.GetDecoder(zeroVal)
		}

		// Cache target pointer for slice header location
		// (slice decoding needs to update the slice header at this location)
		plan.CachedTargetPtr = computeTargetPtr(target, plan.PointerDepth)
	} else {
		// Non-slice target
		plan.IsAbstract = isAbstractType(innerType)

		if !plan.IsAbstract {
			// Create a zero value to get the decoder
			zeroVal := reflect.New(innerType).Interface()
			plan.Decoder = codecs.GetDecoder(zeroVal)

			// Cache target pointer for direct decoding (ZERO reflection at decode time)
			plan.CachedTargetPtr = computeTargetPtr(target, plan.PointerDepth)
		}
	}

	return plan
}

// Note: computeSliceDepth is defined in binding.go as computeDepth
// Both are kept for clarity - computeDepth is used in binding.go's special case handling

// isAbstractType checks if a type implements IAbstract
// Note: Uses package-level rAbstract from registry.go
func isAbstractType(t reflect.Type) bool {
	// Unwrap pointer
	for t.Kind() == reflect.Ptr {
		t = t.Elem()
	}

	// Must be interface
	if t.Kind() != reflect.Interface {
		return false
	}

	// Check if implements IAbstract (rAbstract defined in registry.go)
	return t.Implements(rAbstract)
}

// makeSliceAllocatorFunc creates a function that allocates a slice of given length.
// Note: The closure still uses reflect.MakeSlice per call, but this is called
// once per batch (not per record), so the overhead is amortized.
// The actual per-record decoding uses zero-reflection codec decoders.
func makeSliceAllocatorFunc(sliceType reflect.Type) func(n int) any {
	return func(n int) any {
		slice := reflect.MakeSlice(sliceType, n, n)
		return slice.Interface()
	}
}

// makePtrAllocatorFunc creates a function that allocates a new instance of a type.
// Note: The closure still uses reflect.New per call, but this is called
// once per element in fallback path only. The fast path uses codec allocators.
func makePtrAllocatorFunc(elemType reflect.Type) func() unsafe.Pointer {
	return func() unsafe.Pointer {
		val := reflect.New(elemType)
		return unsafe.Pointer(val.Pointer())
	}
}

// BuildBindingPlans creates BindingPlans for all bindings in a CompiledCypher.
// Called once at query compile time.
func BuildBindingPlans(bindings map[string]any, codecs *codec.CodecRegistry) map[string]*BindingPlan {
	plans := make(map[string]*BindingPlan, len(bindings))
	for key, target := range bindings {
		plans[key] = NewBindingPlan(key, target, codecs)
	}
	return plans
}

// DecodeSingle decodes a single value directly into the binding target using the plan's decoder.
// This is ZERO-REFLECTION for non-abstract types - uses pre-compiled decoder and cached unsafe pointer.
// Returns an error if the plan can't handle direct decoding (abstract types, Valuer interface).
func (p *BindingPlan) DecodeSingle(value any) error {
	// Can't directly decode abstract types - need runtime label lookup
	if p.IsAbstract {
		return fmt.Errorf("cannot use DecodeSingle for abstract type binding %q", p.Key)
	}

	// Can't directly decode if we don't have a decoder
	if p.Decoder == nil {
		return fmt.Errorf("no decoder available for binding %q", p.Key)
	}

	// Can't decode if we don't have a cached pointer
	if p.CachedTargetPtr == nil {
		return fmt.Errorf("no cached target pointer for binding %q", p.Key)
	}

	// Handle nil value - set target to zero value
	if value == nil {
		// For nil, we need to zero out the target
		// This requires knowing the size and using unsafe, but for now
		// we'll leave it to the caller (this is rare case)
		return nil
	}

	// Use pre-compiled decoder with cached pointer - ZERO reflection
	return p.Decoder.Decode(value, p.CachedTargetPtr)
}

// DecodeMultiple decodes multiple values into a slice binding using the plan's allocator and decoder.
// Per-batch reflection: slice allocation uses reflection (SliceAllocator closure).
// Per-record zero-reflection: the Decoder.Decode call uses pre-compiled codec with unsafe pointers.
func (p *BindingPlan) DecodeMultiple(values []any) error {
	if !p.IsSlice {
		return fmt.Errorf("binding %q is not a slice", p.Key)
	}

	if p.IsSliceAbstract {
		return fmt.Errorf("cannot use DecodeMultiple for abstract slice binding %q", p.Key)
	}

	if p.Decoder == nil {
		return fmt.Errorf("no decoder available for binding %q", p.Key)
	}

	if p.CachedTargetPtr == nil {
		return fmt.Errorf("no cached target pointer for binding %q", p.Key)
	}

	n := len(values)
	if n == 0 {
		return nil
	}

	// Allocate slice using pre-compiled allocator
	// This still uses reflection internally (reflect.MakeSlice), but it's called once per batch
	slice := p.SliceAllocator(n)

	// Set the slice header at the cached target pointer
	// We need reflect.ValueOf to get the slice header from the interface{}
	// This is ONE reflection call per batch, not per record
	sliceVal := reflect.ValueOf(slice)
	srcHeader := (*sliceHeader)(unsafe.Pointer(sliceVal.Pointer()))
	dstHeader := (*sliceHeader)(p.CachedTargetPtr)
	*dstHeader = *srcHeader

	// Decode using the slice decoder - ZERO reflection per-element
	return p.Decoder.Decode(values, p.CachedTargetPtr)
}

// sliceHeader is the runtime representation of a slice.
// It matches the layout of reflect.SliceHeader.
type sliceHeader struct {
	Data unsafe.Pointer
	Len  int
	Cap  int
}

// computeTargetPtr unwraps the pointer chain and returns the unsafe.Pointer to the target.
// Called ONCE at plan creation time, not per-record.
// Allocates any nil intermediate pointers in the chain.
func computeTargetPtr(target any, pointerDepth int) unsafe.Pointer {
	v := reflect.ValueOf(target)

	// Unwrap pointer chain, allocating nil pointers along the way
	for i := 0; i < pointerDepth; i++ {
		if v.Kind() != reflect.Ptr {
			return nil
		}
		if v.Elem().Kind() == reflect.Ptr && v.Elem().IsNil() {
			v.Elem().Set(reflect.New(v.Elem().Type().Elem()))
		}
		v = v.Elem()
	}

	if !v.CanAddr() {
		return nil
	}

	return unsafe.Pointer(v.UnsafeAddr())
}
