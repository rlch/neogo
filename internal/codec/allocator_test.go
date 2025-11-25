package codec_test

import (
	"reflect"
	"testing"
	"unsafe"

	"github.com/rlch/neogo/internal/codec"
	"github.com/stretchr/testify/assert"
)

// TestMakeSliceAllocator tests the makeSliceAllocator function
func TestMakeSliceAllocator(t *testing.T) {
	t.Run("AllocateIntSlice", func(t *testing.T) {
		intType := reflect.TypeOf(int(0))
		allocator := codec.MakeSliceAllocator(intType)

		ptr, len_, cap := allocator(10)

		assert.NotNil(t, ptr)
		assert.Equal(t, 10, cap)
		assert.Equal(t, 0, len_)
	})

	t.Run("AllocateStringSlice", func(t *testing.T) {
		strType := reflect.TypeOf("")
		allocator := codec.MakeSliceAllocator(strType)

		ptr, len_, cap := allocator(5)

		assert.NotNil(t, ptr)
		assert.Equal(t, 5, cap)
		assert.Equal(t, 0, len_)
	})

	t.Run("AllocateZeroCapacity", func(t *testing.T) {
		intType := reflect.TypeOf(int(0))
		allocator := codec.MakeSliceAllocator(intType)

		ptr, len_, cap := allocator(0)

		assert.Nil(t, ptr)
		assert.Equal(t, 0, cap)
		assert.Equal(t, 0, len_)
	})

	t.Run("MultipleAllocationsIndependent", func(t *testing.T) {
		intType := reflect.TypeOf(int(0))
		allocator := codec.MakeSliceAllocator(intType)

		ptr1, _, cap1 := allocator(10)
		ptr2, _, cap2 := allocator(20)

		assert.NotEqual(t, ptr1, ptr2, "multiple allocations should have different pointers")
		assert.Equal(t, 10, cap1)
		assert.Equal(t, 20, cap2)
	})

	t.Run("LargeAllocation", func(t *testing.T) {
		intType := reflect.TypeOf(int(0))
		allocator := codec.MakeSliceAllocator(intType)

		ptr, _, cap := allocator(1000000)

		assert.NotNil(t, ptr)
		assert.Equal(t, 1000000, cap)
	})

	t.Run("StructSliceAllocation", func(t *testing.T) {
		structType := reflect.TypeOf(TestStruct{})
		allocator := codec.MakeSliceAllocator(structType)

		ptr, _, cap := allocator(50)

		assert.NotNil(t, ptr)
		assert.Equal(t, 50, cap)
	})
}

// TestMakePtrAllocator tests the makePtrAllocator function
func TestMakePtrAllocator(t *testing.T) {
	t.Run("AllocateIntPtr", func(t *testing.T) {
		intType := reflect.TypeOf(int(0))
		allocator := codec.MakePtrAllocator(intType)

		ptr := allocator()

		assert.NotNil(t, ptr)
		// Should be able to dereference as int
		*((*int)(ptr)) = 42
		assert.Equal(t, 42, *((*int)(ptr)))
	})

	t.Run("AllocateStringPtr", func(t *testing.T) {
		strType := reflect.TypeOf("")
		allocator := codec.MakePtrAllocator(strType)

		ptr := allocator()

		assert.NotNil(t, ptr)
		// Should be able to dereference as string
		*((*string)(ptr)) = "hello"
		assert.Equal(t, "hello", *((*string)(ptr)))
	})

	t.Run("AllocateStructPtr", func(t *testing.T) {
		structType := reflect.TypeOf(TestStruct{})
		allocator := codec.MakePtrAllocator(structType)

		ptr := allocator()

		assert.NotNil(t, ptr)
		// Should be able to dereference as struct
		s := (*TestStruct)(ptr)
		s.Int = 99
		assert.Equal(t, 99, (*TestStruct)(ptr).Int)
	})

	t.Run("MultipleAllocationsIndependent", func(t *testing.T) {
		intType := reflect.TypeOf(int(0))
		allocator := codec.MakePtrAllocator(intType)

		ptr1 := allocator()
		ptr2 := allocator()

		assert.NotEqual(t, ptr1, ptr2, "multiple allocations should have different pointers")

		*((*int)(ptr1)) = 11
		*((*int)(ptr2)) = 22

		assert.Equal(t, 11, *((*int)(ptr1)))
		assert.Equal(t, 22, *((*int)(ptr2)))
	})

	t.Run("NestedStructAllocation", func(t *testing.T) {
		structType := reflect.TypeOf(NestedStruct{})
		allocator := codec.MakePtrAllocator(structType)

		ptr := allocator()

		assert.NotNil(t, ptr)
		s := (*NestedStruct)(ptr)
		s.Val = "test"
		assert.Equal(t, "test", (*NestedStruct)(ptr).Val)
	})
}

// TestMakeMapAllocator tests the makeMapAllocator function
func TestMakeMapAllocator(t *testing.T) {
	t.Run("AllocateStringStringMap", func(t *testing.T) {
		mapType := reflect.TypeOf(map[string]string{})
		allocator := codec.MakeMapAllocator(mapType)

		ptr := allocator()

		assert.NotNil(t, ptr)
		// Verify it's a valid map pointer
		assert.NotZero(t, unsafe.Pointer(uintptr(ptr)))
	})

	t.Run("AllocateStringIntMap", func(t *testing.T) {
		mapType := reflect.TypeOf(map[string]int{})
		allocator := codec.MakeMapAllocator(mapType)

		ptr := allocator()

		assert.NotNil(t, ptr)
	})

	t.Run("MultipleAllocationsIndependent", func(t *testing.T) {
		mapType := reflect.TypeOf(map[string]string{})
		allocator := codec.MakeMapAllocator(mapType)

		ptr1 := allocator()
		ptr2 := allocator()

		assert.NotEqual(t, ptr1, ptr2, "multiple allocations should have different pointers")
	})

	t.Run("StringInterfaceMap", func(t *testing.T) {
		mapType := reflect.TypeOf(map[string]interface{}{})
		allocator := codec.MakeMapAllocator(mapType)

		ptr := allocator()

		assert.NotNil(t, ptr)
	})
}

// TestAllocatorConsistency tests that allocators are consistent with reflect
func TestAllocatorConsistency(t *testing.T) {
	t.Run("SliceAllocatorElementSize", func(t *testing.T) {
		intType := reflect.TypeOf(int(0))
		expectedSize := intType.Size()

		allocator := codec.MakeSliceAllocator(intType)
		ptr, _, cap := allocator(100)

		// We can't directly test element size, but we can verify the allocation works
		assert.NotNil(t, ptr)
		assert.Equal(t, 100, cap)
		
		// Write to different offsets and verify they don't overwrite
		*((*int)(unsafe.Pointer(uintptr(ptr)))) = 1
		offset := unsafe.Pointer(uintptr(ptr) + expectedSize)
		*((*int)(offset)) = 2
		
		assert.Equal(t, 1, *((*int)(unsafe.Pointer(uintptr(ptr)))))
		assert.Equal(t, 2, *((*int)(offset)))
	})

	t.Run("PtrAllocatorMemoryInitialization", func(t *testing.T) {
		intType := reflect.TypeOf(int(0))
		allocator := codec.MakePtrAllocator(intType)

		ptr := allocator()
		// Memory should be allocated (might be zero-initialized)
		assert.NotNil(t, ptr)
	})
}
