package internal

import (
	"reflect"
	"testing"

	"github.com/rlch/neogo/internal/codec"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type testPerson struct {
	Node `neo4j:"Person"`
	Name string `neo4j:"name"`
	Age  int    `neo4j:"age"`
}

func (testPerson) IsNode()         {}
func (p testPerson) GetID() string { return p.ID }

func TestNewBindingPlan(t *testing.T) {
	codecs := codec.NewCodecRegistry()
	codecs.RegisterTypes(testPerson{})

	t.Run("slice of structs", func(t *testing.T) {
		var people []testPerson
		plan := NewBindingPlan("people", &people, codecs)

		assert.Equal(t, "people", plan.Key)
		assert.True(t, plan.IsSlice)
		assert.False(t, plan.IsPointerElem)
		assert.False(t, plan.IsAbstract)
		assert.False(t, plan.IsSliceAbstract)
		assert.Equal(t, 1, plan.SliceDepth)
		assert.Equal(t, 1, plan.PointerDepth) // *[]T
		assert.NotNil(t, plan.SliceAllocator)
		assert.NotNil(t, plan.Decoder)
	})

	t.Run("slice of pointers to structs", func(t *testing.T) {
		var people []*testPerson
		plan := NewBindingPlan("people", &people, codecs)

		assert.True(t, plan.IsSlice)
		assert.True(t, plan.IsPointerElem)
		assert.False(t, plan.IsAbstract)
		assert.Equal(t, 1, plan.SliceDepth)
		assert.NotNil(t, plan.ElemAllocator)
	})

	t.Run("nested slice", func(t *testing.T) {
		var people [][]testPerson
		plan := NewBindingPlan("people", &people, codecs)

		assert.True(t, plan.IsSlice)
		assert.Equal(t, 2, plan.SliceDepth)
	})

	t.Run("single struct pointer", func(t *testing.T) {
		var person testPerson
		plan := NewBindingPlan("person", &person, codecs)

		assert.False(t, plan.IsSlice)
		assert.False(t, plan.IsAbstract)
		assert.Equal(t, 1, plan.PointerDepth)
		assert.NotNil(t, plan.Decoder)
	})

	t.Run("double pointer to slice", func(t *testing.T) {
		var people *[]testPerson
		plan := NewBindingPlan("people", &people, codecs)

		assert.True(t, plan.IsSlice)
		assert.Equal(t, 2, plan.PointerDepth) // **[]T
		assert.Equal(t, 1, plan.SliceDepth)
	})
}

func TestBuildBindingPlans(t *testing.T) {
	codecs := codec.NewCodecRegistry()
	codecs.RegisterTypes(testPerson{})

	var people []testPerson
	var count int

	bindings := map[string]any{
		"people": &people,
		"count":  &count,
	}

	plans := BuildBindingPlans(bindings, codecs)

	require.Len(t, plans, 2)
	assert.NotNil(t, plans["people"])
	assert.NotNil(t, plans["count"])

	assert.True(t, plans["people"].IsSlice)
	assert.False(t, plans["count"].IsSlice)
}

func TestSliceAllocator(t *testing.T) {
	sliceType := reflect.TypeOf([]testPerson{})
	allocator := makeSliceAllocatorFunc(sliceType)

	result := allocator(5)
	slice, ok := result.([]testPerson)
	require.True(t, ok)
	assert.Len(t, slice, 5)
	assert.Equal(t, 5, cap(slice))
}

func TestComputeSliceDepth(t *testing.T) {
	tests := []struct {
		name  string
		typ   reflect.Type
		depth int
	}{
		{"[]int", reflect.TypeOf([]int{}), 1},
		{"[][]int", reflect.TypeOf([][]int{}), 2},
		{"[][][]int", reflect.TypeOf([][][]int{}), 3},
		{"int", reflect.TypeOf(0), 0},
		{"struct", reflect.TypeOf(testPerson{}), 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.depth, computeSliceDepth(tt.typ))
		})
	}
}
