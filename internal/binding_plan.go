package internal

import (
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
		plan.SliceDepth = computeSliceDepth(innerType)

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
	} else {
		// Non-slice target
		plan.IsAbstract = isAbstractType(innerType)

		if !plan.IsAbstract {
			// Create a zero value to get the decoder
			zeroVal := reflect.New(innerType).Interface()
			plan.Decoder = codecs.GetDecoder(zeroVal)
		}
	}

	return plan
}

// computeSliceDepth returns the nesting depth of a slice type
func computeSliceDepth(t reflect.Type) int {
	depth := 0
	for t.Kind() == reflect.Slice {
		depth++
		t = t.Elem()
	}
	return depth
}

// isAbstractType checks if a type implements IAbstract
func isAbstractType(t reflect.Type) bool {
	// Unwrap pointer
	for t.Kind() == reflect.Ptr {
		t = t.Elem()
	}

	// Must be interface
	if t.Kind() != reflect.Interface {
		return false
	}

	// Check if implements IAbstract
	rAbstract := reflect.TypeOf((*IAbstract)(nil)).Elem()
	return t.Implements(rAbstract)
}

// makeSliceAllocatorFunc creates a function that allocates a slice of given length
// Uses reflection ONCE at creation, returns a closure that doesn't use reflection
func makeSliceAllocatorFunc(sliceType reflect.Type) func(n int) any {
	// Capture the type at creation time
	return func(n int) any {
		slice := reflect.MakeSlice(sliceType, n, n)
		return slice.Interface()
	}
}

// makePtrAllocatorFunc creates a function that allocates a new instance of a type
// Uses reflection ONCE at creation, returns a closure
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
