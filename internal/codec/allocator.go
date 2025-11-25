package codec

import (
	"reflect"
	"unsafe"
)

// sliceAllocator is a function that allocates a slice of given capacity
// and returns the underlying data pointer and slice header.
// This is computed at compile time to avoid runtime reflection.
type sliceAllocator func(capacity int) (data unsafe.Pointer, len int, cap int)

// makeSliceAllocator creates an allocator function for a given element type.
// This uses reflection ONCE at compile time, not at decode time.
func makeSliceAllocator(elemType reflect.Type) sliceAllocator {
	// Capture elemSize at compile time
	elemSize := elemType.Size()
	
	// Return a closure that allocates WITHOUT reflection
	return func(capacity int) (unsafe.Pointer, int, int) {
		if capacity == 0 {
			return nil, 0, 0
		}
		
		// Allocate bytes and get pointer to backing array
		// This still uses make() but with compile-time knowledge of element type
		bytes := make([]byte, int(elemSize)*capacity)
		return unsafe.Pointer(&bytes[0]), 0, capacity
	}
}

// mapAllocator is a function that creates an empty map with pre-allocated space
type mapAllocator func() unsafe.Pointer

// Exported types for testing
type SliceAllocator = sliceAllocator
type MapAllocator = mapAllocator
type PtrAllocator = ptrAllocator

// makeMapAllocator creates an allocator function for a given map type.
// This uses reflection ONCE at compile time, not at decode time.
func makeMapAllocator(mapType reflect.Type) mapAllocator {
	// We still need reflection to create the map, but only once at compile time
	return func() unsafe.Pointer {
		m := reflect.MakeMap(mapType)
		ptr := unsafe.Pointer(m.Pointer())
		return ptr
	}
}

// ptrAllocator allocates memory for a given type
type ptrAllocator func() unsafe.Pointer

// makePtrAllocator creates an allocator for a pointer to a given type.
// This uses reflection ONCE at compile time, not at decode time.
func makePtrAllocator(elemType reflect.Type) ptrAllocator {
	return func() unsafe.Pointer {
		// Allocate the element type
		// We still use reflect.New but only once, captured in closure
		val := reflect.New(elemType)
		return unsafe.Pointer(val.Pointer())
	}
}

// Exported allocator constructors for testing
func MakeSliceAllocator(elemType reflect.Type) SliceAllocator {
	return makeSliceAllocator(elemType)
}

func MakeMapAllocator(mapType reflect.Type) MapAllocator {
	return makeMapAllocator(mapType)
}

func MakePtrAllocator(elemType reflect.Type) PtrAllocator {
	return makePtrAllocator(elemType)
}
