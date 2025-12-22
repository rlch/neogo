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

// BenchmarkDecodeSingle compares plan-based vs reflection-based decoding
func BenchmarkDecodeSingle(b *testing.B) {
	codecs := codec.NewCodecRegistry()
	codecs.RegisterTypes(testPerson{})

	props := map[string]any{
		"name": "Alice",
		"age":  int64(30),
	}

	b.Run("plan-based", func(b *testing.B) {
		var person testPerson
		plan := NewBindingPlan("person", &person, codecs)

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			person = testPerson{} // Reset
			_ = plan.DecodeSingle(props)
		}
	})

	b.Run("codec-direct", func(b *testing.B) {
		var person testPerson

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			person = testPerson{} // Reset
			_ = codecs.Decode(props, &person)
		}
	})
}

// BenchmarkDecodeMultiple compares batch decoding approaches
func BenchmarkDecodeMultiple(b *testing.B) {
	codecs := codec.NewCodecRegistry()
	codecs.RegisterTypes(testPerson{})

	values := make([]any, 100)
	for i := 0; i < 100; i++ {
		values[i] = map[string]any{
			"name": "Person",
			"age":  int64(i),
		}
	}

	b.Run("plan-based", func(b *testing.B) {
		var people []testPerson
		plan := NewBindingPlan("people", &people, codecs)

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			people = nil // Reset
			_ = plan.DecodeMultiple(values)
		}
	})

	b.Run("codec-DecodeMultiple", func(b *testing.B) {
		var people []testPerson

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			people = nil // Reset
			_ = codecs.DecodeMultiple(values, &people)
		}
	})
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
			assert.Equal(t, tt.depth, computeDepth(tt.typ))
		})
	}
}

func TestBindingPlan_DecodeSingle(t *testing.T) {
	codecs := codec.NewCodecRegistry()
	codecs.RegisterTypes(testPerson{})

	t.Run("decode struct from map", func(t *testing.T) {
		var person testPerson
		plan := NewBindingPlan("person", &person, codecs)

		props := map[string]any{
			"name": "Alice",
			"age":  int64(30),
		}

		err := plan.DecodeSingle(props)
		require.NoError(t, err)
		assert.Equal(t, "Alice", person.Name)
		assert.Equal(t, 30, person.Age)
	})

	t.Run("decode primitive", func(t *testing.T) {
		var name string
		plan := NewBindingPlan("name", &name, codecs)

		err := plan.DecodeSingle("Bob")
		require.NoError(t, err)
		assert.Equal(t, "Bob", name)
	})

	t.Run("decode int", func(t *testing.T) {
		var count int
		plan := NewBindingPlan("count", &count, codecs)

		err := plan.DecodeSingle(int64(42))
		require.NoError(t, err)
		assert.Equal(t, 42, count)
	})

	t.Run("decode nil is noop", func(t *testing.T) {
		name := "initial"
		plan := NewBindingPlan("name", &name, codecs)

		err := plan.DecodeSingle(nil)
		require.NoError(t, err)
		// nil doesn't change the value (noop)
		assert.Equal(t, "initial", name)
	})

	t.Run("reject abstract type", func(t *testing.T) {
		var abstract IAbstract
		plan := NewBindingPlan("abstract", &abstract, codecs)

		err := plan.DecodeSingle(map[string]any{})
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "abstract type")
	})
}

func TestBindingPlan_DecodeMultiple(t *testing.T) {
	codecs := codec.NewCodecRegistry()
	codecs.RegisterTypes(testPerson{})

	t.Run("decode slice of structs", func(t *testing.T) {
		var people []testPerson
		plan := NewBindingPlan("people", &people, codecs)

		values := []any{
			map[string]any{"name": "Alice", "age": int64(30)},
			map[string]any{"name": "Bob", "age": int64(25)},
		}

		err := plan.DecodeMultiple(values)
		require.NoError(t, err)
		require.Len(t, people, 2)
		assert.Equal(t, "Alice", people[0].Name)
		assert.Equal(t, 30, people[0].Age)
		assert.Equal(t, "Bob", people[1].Name)
		assert.Equal(t, 25, people[1].Age)
	})

	t.Run("decode slice of primitives", func(t *testing.T) {
		var names []string
		plan := NewBindingPlan("names", &names, codecs)

		values := []any{"Alice", "Bob", "Charlie"}

		err := plan.DecodeMultiple(values)
		require.NoError(t, err)
		require.Len(t, names, 3)
		assert.Equal(t, []string{"Alice", "Bob", "Charlie"}, names)
	})

	t.Run("reject non-slice", func(t *testing.T) {
		var person testPerson
		plan := NewBindingPlan("person", &person, codecs)

		err := plan.DecodeMultiple([]any{map[string]any{}})
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "not a slice")
	})

	t.Run("empty values noop", func(t *testing.T) {
		var people []testPerson
		plan := NewBindingPlan("people", &people, codecs)

		err := plan.DecodeMultiple([]any{})
		require.NoError(t, err)
		// Empty input should be a noop
	})
}

func TestBindingPlan_CachedTargetPtr(t *testing.T) {
	codecs := codec.NewCodecRegistry()
	codecs.RegisterTypes(testPerson{})

	t.Run("non-slice target has cached pointer", func(t *testing.T) {
		var person testPerson
		plan := NewBindingPlan("person", &person, codecs)

		assert.NotNil(t, plan.CachedTargetPtr, "CachedTargetPtr should be set for non-slice targets")
		assert.NotNil(t, plan.Decoder, "Decoder should be set")
	})

	t.Run("slice target has cached pointer", func(t *testing.T) {
		var people []testPerson
		plan := NewBindingPlan("people", &people, codecs)

		assert.NotNil(t, plan.CachedTargetPtr, "CachedTargetPtr should be set for slice targets")
		assert.NotNil(t, plan.Decoder, "Decoder should be set")
	})

	t.Run("primitive target has cached pointer", func(t *testing.T) {
		var name string
		plan := NewBindingPlan("name", &name, codecs)

		assert.NotNil(t, plan.CachedTargetPtr, "CachedTargetPtr should be set for primitive targets")
	})

	t.Run("abstract target has nil cached pointer", func(t *testing.T) {
		var abstract IAbstract
		plan := NewBindingPlan("abstract", &abstract, codecs)

		// Abstract types need runtime label lookup, so no cached pointer
		assert.Nil(t, plan.CachedTargetPtr, "CachedTargetPtr should be nil for abstract targets")
		assert.True(t, plan.IsAbstract)
	})

	t.Run("DecodeSingle uses cached pointer without additional reflection", func(t *testing.T) {
		var person testPerson
		plan := NewBindingPlan("person", &person, codecs)

		// Verify the plan is set up correctly
		require.NotNil(t, plan.CachedTargetPtr)
		require.NotNil(t, plan.Decoder)

		// DecodeSingle should succeed and decode directly to the cached pointer
		props := map[string]any{
			"name": "Alice",
			"age":  int64(30),
		}

		err := plan.DecodeSingle(props)
		require.NoError(t, err)

		// Verify the data was written to the original variable
		assert.Equal(t, "Alice", person.Name)
		assert.Equal(t, 30, person.Age)
	})

	t.Run("DecodeMultiple uses cached pointer for slice header", func(t *testing.T) {
		var people []testPerson
		plan := NewBindingPlan("people", &people, codecs)

		// Verify the plan is set up correctly
		require.NotNil(t, plan.CachedTargetPtr)
		require.NotNil(t, plan.Decoder)

		values := []any{
			map[string]any{"name": "Alice", "age": int64(30)},
			map[string]any{"name": "Bob", "age": int64(25)},
		}

		err := plan.DecodeMultiple(values)
		require.NoError(t, err)

		// Verify the slice was updated through the cached pointer
		require.Len(t, people, 2)
		assert.Equal(t, "Alice", people[0].Name)
		assert.Equal(t, "Bob", people[1].Name)
	})

	t.Run("nested pointer target allocates intermediate pointers", func(t *testing.T) {
		var personPtr *testPerson
		plan := NewBindingPlan("person", &personPtr, codecs)

		// For **T targets, the plan should allocate intermediate pointers
		// and CachedTargetPtr should point to where the T should be decoded
		assert.NotNil(t, plan.CachedTargetPtr)
		assert.Equal(t, 2, plan.PointerDepth) // **testPerson

		props := map[string]any{
			"name": "Alice",
			"age":  int64(30),
		}

		err := plan.DecodeSingle(props)
		require.NoError(t, err)

		// The intermediate pointer should have been allocated
		require.NotNil(t, personPtr)
		assert.Equal(t, "Alice", personPtr.Name)
		assert.Equal(t, 30, personPtr.Age)
	})
}
