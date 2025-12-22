package internal

import (
	"errors"
	"reflect"
	"testing"
	"time"

	"github.com/neo4j/neo4j-go-driver/v5/neo4j"
	"github.com/stretchr/testify/require"
)

type Organism interface {
	IAbstract
}

type BaseOrganism struct {
	Node
	Abstract `neo4j:"Organism"`
	Alive    bool `neo4j:"alive"`
}

func (b BaseOrganism) Implementers() []IAbstract {
	return []IAbstract{
		&Human{},
		&Dog{},
	}
}

type Person struct {
	Node `neo4j:"Person"`
	Name string `neo4j:"name"`
}

type ActedIn struct {
	Relationship `neo4j:"ACTED_IN"`
	Role         string `neo4j:"role"`
}

type Human struct {
	BaseOrganism `neo4j:"Human"`
	Name         string `neo4j:"name"`
}

type Dog struct {
	BaseOrganism `neo4j:"Dog"`
	Borfs        bool `neo4j:"borfs"`
}

type (
	simpleValuer[T neo4j.RecordValue] struct {
		Value     T
		shouldErr bool
	}
	nodeValuer struct {
		Value     map[string]any
		shouldErr bool
	}
	relationshipValuer struct {
		Value     map[string]any
		shouldErr bool
	}
)

var (
	_ Valuer[bool]               = (*simpleValuer[bool])(nil)
	_ Valuer[neo4j.Node]         = (*nodeValuer)(nil)
	_ Valuer[neo4j.Relationship] = (*relationshipValuer)(nil)
)

func (b simpleValuer[T]) Marshal() (*T, error) {
	if b.shouldErr {
		return nil, errors.New("intentional error")
	}
	return &b.Value, nil
}

func (b *simpleValuer[T]) Unmarshal(v *T) error {
	if b.shouldErr {
		return errors.New("intentional error")
	}
	b.Value = *v
	return nil
}

func (b nodeValuer) Marshal() (*neo4j.Node, error) {
	if b.shouldErr {
		return nil, errors.New("intentional error")
	}
	return &neo4j.Node{Props: b.Value}, nil
}

func (b *nodeValuer) Unmarshal(v *neo4j.Node) error {
	if b.shouldErr {
		return errors.New("intentional error")
	}
	b.Value = v.Props
	return nil
}

func (b relationshipValuer) Marshal() (*neo4j.Relationship, error) {
	if b.shouldErr {
		return nil, errors.New("intentional error")
	}
	return &neo4j.Relationship{Props: b.Value}, nil
}

func (b *relationshipValuer) Unmarshal(v *neo4j.Relationship) error {
	if b.shouldErr {
		return errors.New("intentional error")
	}
	b.Value = v.Props
	return nil
}

func TestBindValuer(t *testing.T) {
	t.Run("err nil when not implemented", func(t *testing.T) {
		ok, err := bindValuer(false, reflect.ValueOf(10))
		require.False(t, ok)
		require.NoError(t, err)
	})

	t.Run("err when unmarshal fails", func(t *testing.T) {
		v := &simpleValuer[bool]{
			shouldErr: true,
		}
		ok, err := bindValuer(false, reflect.ValueOf(v))
		require.False(t, ok)
		require.Error(t, err)
	})

	t.Run("unmarshals to bindTo", func(t *testing.T) {
		v := &simpleValuer[bool]{}
		ok, err := bindValuer(true, reflect.ValueOf(v))
		require.True(t, ok)
		require.NoError(t, err)
		require.True(t, v.Value)
	})
}

func TestBind(t *testing.T) {
	r := NewRegistry()
	r.RegisterTypes(&BaseOrganism{})

	t.Run("binds primitive to pointer", func(t *testing.T) {
		var target string
		err := r.Bind("hello", &target)
		require.NoError(t, err)
		require.Equal(t, "hello", target)
	})

	t.Run("binds node to struct pointer", func(t *testing.T) {
		target := &Person{}
		err := r.Bind(neo4j.Node{
			Labels: []string{"Person"},
			Props:  map[string]any{"name": "Alice"},
		}, target)
		require.NoError(t, err)
		require.Equal(t, "Alice", target.Name)
	})

	t.Run("fails when target is not a pointer", func(t *testing.T) {
		target := "hello"
		err := r.Bind("world", target)
		require.Error(t, err)
		require.Contains(t, err.Error(), "pointer target")
	})
}

func TestBindNil(t *testing.T) {
	r := NewRegistry()

	t.Run("nil to slice creates single element slice with zero", func(t *testing.T) {
		var target []string
		err := r.BindValue(nil, reflect.ValueOf(&target).Elem())
		require.NoError(t, err)
		require.Len(t, target, 1)
		require.Equal(t, "", target[0])
	})

	t.Run("nil to pointer sets to nil", func(t *testing.T) {
		target := &Person{Name: "test"}
		targetPtr := &target
		err := r.BindValue(nil, reflect.ValueOf(targetPtr).Elem())
		require.NoError(t, err)
		require.Nil(t, target)
	})

	t.Run("nil to struct sets zero value", func(t *testing.T) {
		target := Person{Name: "test"}
		err := r.BindValue(nil, reflect.ValueOf(&target).Elem())
		require.NoError(t, err)
		require.Equal(t, Person{}, target)
	})

	t.Run("nil to nested pointer slice", func(t *testing.T) {
		var slice []*string
		err := r.BindValue(nil, reflect.ValueOf(&slice).Elem())
		require.NoError(t, err)
		require.Len(t, slice, 1)
		require.Nil(t, slice[0])
	})
}

func TestBindSliceDepthMismatch(t *testing.T) {
	r := NewRegistry()

	t.Run("wraps slice in outer slice when depth differs by 1", func(t *testing.T) {
		// []Person -> [][]Person (depth 1 -> depth 2)
		input := []any{
			neo4j.Node{Props: map[string]any{"name": "Alice"}},
			neo4j.Node{Props: map[string]any{"name": "Bob"}},
		}
		var target [][]Person
		err := r.bindSliceDepthMismatch(input, reflect.ValueOf(&target).Elem(), 1, 2)
		require.NoError(t, err)
		require.Len(t, target, 1)
		require.Len(t, target[0], 2)
		require.Equal(t, "Alice", target[0][0].Name)
		require.Equal(t, "Bob", target[0][1].Name)
	})

	t.Run("fails when depth mismatch is greater than 1", func(t *testing.T) {
		input := []any{"a", "b"}
		var target [][][]string
		err := r.bindSliceDepthMismatch(input, reflect.ValueOf(&target).Elem(), 1, 3)
		require.Error(t, err)
		require.Contains(t, err.Error(), "cannot bind slice of depth 1 to slice of depth 3")
	})

	t.Run("fails when target is not a slice", func(t *testing.T) {
		var target string
		err := r.bindSliceDepthMismatch([]any{"a"}, reflect.ValueOf(&target).Elem(), 1, 0)
		require.Error(t, err)
		require.Contains(t, err.Error(), "cannot bind slice to non-slice type")
	})
}

func TestWrapInSlice(t *testing.T) {
	r := NewRegistry()
	r.RegisterTypes(&BaseOrganism{})

	t.Run("wraps single node in slice of abstract", func(t *testing.T) {
		input := neo4j.Node{
			Labels: []string{"Organism", "Human"},
			Props:  map[string]any{"name": "Alice"},
		}
		var target []Organism
		err := r.wrapInSlice(input, reflect.ValueOf(&target).Elem())
		require.NoError(t, err)
		require.Len(t, target, 1)
		human, ok := target[0].(*Human)
		require.True(t, ok)
		require.Equal(t, "Alice", human.Name)
	})

	t.Run("wraps single value in pointer slice", func(t *testing.T) {
		input := neo4j.Node{Props: map[string]any{"name": "Bob"}}
		slice := new([]Person)
		err := r.wrapInSlice(input, reflect.ValueOf(&slice).Elem())
		require.NoError(t, err)
		require.Len(t, *slice, 1)
		require.Equal(t, "Bob", (*slice)[0].Name)
	})
}

func TestBindSliceWithAbstractElements(t *testing.T) {
	r := NewRegistry()
	r.RegisterTypes(&BaseOrganism{})

	t.Run("binds slice of nodes to slice of abstract", func(t *testing.T) {
		input := []any{
			neo4j.Node{
				Labels: []string{"Organism", "Human"},
				Props:  map[string]any{"name": "Alice", "alive": true},
			},
			neo4j.Node{
				Labels: []string{"Organism", "Dog"},
				Props:  map[string]any{"borfs": true, "alive": false},
			},
		}
		var target []Organism
		err := r.bindSliceWithAbstractElements(input, reflect.ValueOf(&target).Elem())
		require.NoError(t, err)
		require.Len(t, target, 2)

		human, ok := target[0].(*Human)
		require.True(t, ok)
		require.Equal(t, "Alice", human.Name)
		require.True(t, human.Alive)

		dog, ok := target[1].(*Dog)
		require.True(t, ok)
		require.True(t, dog.Borfs)
		require.False(t, dog.Alive)
	})

	t.Run("handles pointer to slice", func(t *testing.T) {
		input := []any{
			neo4j.Node{
				Labels: []string{"Organism", "Human"},
				Props:  map[string]any{"name": "Bob"},
			},
		}
		target := new([]Organism)
		err := r.bindSliceWithAbstractElements(input, reflect.ValueOf(&target).Elem())
		require.NoError(t, err)
		require.Len(t, *target, 1)
	})
}

func TestBindValue(t *testing.T) {
	r := NewRegistry()
	r.RegisterTypes(&BaseOrganism{})

	t.Run("Primitive via codec", func(t *testing.T) {
		// The codec handles direct type matching (int64 -> int64, string -> string, etc.)
		// Type coercion (string -> int) is no longer supported directly - use proper types

		t.Run("bool", func(t *testing.T) {
			bindTo := false
			err := r.BindValue(true, reflect.ValueOf(&bindTo).Elem())
			require.NoError(t, err)
			require.True(t, bindTo)
		})

		t.Run("string", func(t *testing.T) {
			bindTo := ""
			err := r.BindValue("hello", reflect.ValueOf(&bindTo).Elem())
			require.NoError(t, err)
			require.Equal(t, "hello", bindTo)
		})

		t.Run("int64", func(t *testing.T) {
			bindTo := int64(0)
			err := r.BindValue(int64(100), reflect.ValueOf(&bindTo).Elem())
			require.NoError(t, err)
			require.Equal(t, int64(100), bindTo)
		})

		t.Run("float64", func(t *testing.T) {
			bindTo := float64(0)
			err := r.BindValue(3.14, reflect.ValueOf(&bindTo).Elem())
			require.NoError(t, err)
			require.Equal(t, 3.14, bindTo)
		})

		t.Run("time.Time from neo4j.Time", func(t *testing.T) {
			inputTime := time.Date(2023, time.August, 4, 12, 0, 0, 0, time.UTC)
			bindTo := time.Time{}
			err := r.BindValue(inputTime, reflect.ValueOf(&bindTo).Elem())
			require.NoError(t, err)
			require.Equal(t, inputTime, bindTo)
		})
	})

	t.Run("Valuer", func(t *testing.T) {
		t.Run("bool", func(t *testing.T) {
			bindTo := &simpleValuer[bool]{}
			err := r.BindValue(true, reflect.ValueOf(bindTo))
			require.NoError(t, err)
			require.True(t, bindTo.Value)
		})

		t.Run("int64", func(t *testing.T) {
			bindTo := &simpleValuer[int64]{}
			err := r.BindValue(int64(100), reflect.ValueOf(bindTo))
			require.NoError(t, err)
			require.Equal(t, int64(100), bindTo.Value)
		})

		t.Run("string", func(t *testing.T) {
			bindTo := &simpleValuer[string]{}
			err := r.BindValue("hello", reflect.ValueOf(bindTo))
			require.NoError(t, err)
			require.Equal(t, "hello", bindTo.Value)
		})

		t.Run("float64", func(t *testing.T) {
			bindTo := &simpleValuer[float64]{}
			err := r.BindValue(3.14, reflect.ValueOf(bindTo))
			require.NoError(t, err)
			require.Equal(t, 3.14, bindTo.Value)
		})

		t.Run("time.Time", func(t *testing.T) {
			inputTime := time.Date(2023, time.August, 4, 12, 0, 0, 0, time.UTC)
			bindTo := &simpleValuer[time.Time]{}
			err := r.BindValue(inputTime, reflect.ValueOf(bindTo))
			require.NoError(t, err)
			require.Equal(t, inputTime, bindTo.Value)
		})

		t.Run("[]byte", func(t *testing.T) {
			input := []byte{0x68, 0x65, 0x6c, 0x6c, 0x6f}
			bindTo := &simpleValuer[[]byte]{}
			err := r.BindValue(input, reflect.ValueOf(bindTo))
			require.NoError(t, err)
			require.Equal(t, input, bindTo.Value)
		})

		t.Run("[]any", func(t *testing.T) {
			input := []any{1, "hello", true}
			bindTo := &simpleValuer[[]any]{}
			err := r.BindValue(input, reflect.ValueOf(bindTo))
			require.NoError(t, err)
			require.Equal(t, input, bindTo.Value)
		})

		t.Run("map[string]any", func(t *testing.T) {
			input := map[string]any{"name": "John", "age": 30}
			bindTo := &simpleValuer[map[string]any]{}
			err := r.BindValue(input, reflect.ValueOf(bindTo))
			require.NoError(t, err)
			require.Equal(t, input, bindTo.Value)
		})

		t.Run("Node", func(t *testing.T) {
			input := neo4j.Node{
				Props: map[string]any{
					"name": "Richard",
				},
			}
			bindTo := &nodeValuer{}
			err := r.BindValue(input, reflect.ValueOf(bindTo))
			require.NoError(t, err)
			require.Equal(t, map[string]any{
				"name": "Richard",
			}, bindTo.Value)
		})

		t.Run("Relationship", func(t *testing.T) {
			input := neo4j.Relationship{
				Props: map[string]any{
					"weight": 0.5,
				},
			}
			bindTo := &relationshipValuer{}
			err := r.BindValue(input, reflect.ValueOf(bindTo))
			require.NoError(t, err)
			require.Equal(t, map[string]any{
				"weight": 0.5,
			}, bindTo.Value)
		})
	})

	t.Run("Node", func(t *testing.T) {
		to := &Person{}
		err := r.BindValue(neo4j.Node{
			Labels: []string{"Person"},
			Props: map[string]any{
				"name":    "Richard",
				"surname": "Mathieson",
				"age":     24,
			},
		}, reflect.ValueOf(to))
		require.NoError(t, err)
		require.Equal(t, Person{
			Name: "Richard",
		}, *to)
	})

	t.Run("Relationship", func(t *testing.T) {
		to := &ActedIn{}
		err := r.BindValue(neo4j.Node{
			Labels: []string{"ACTED_IN"},
			Props: map[string]any{
				"role": "Stuntman",
			},
		}, reflect.ValueOf(to))
		require.NoError(t, err)
		require.Equal(t, ActedIn{
			Role: "Stuntman",
		}, *to)
	})

	t.Run("Abstract using base type", func(t *testing.T) {
		var to Organism = &BaseOrganism{}
		err := r.BindValue(neo4j.Node{
			Labels: []string{"Human", "Organism"},
			Props: map[string]any{
				"name": "bruh",
			},
		}, reflect.ValueOf(&to))
		require.NoError(t, err)
		require.Equal(t, &Human{
			Name: "bruh",
		}, to)
	})

	t.Run("Abstract using registered types", func(t *testing.T) {
		rWithAbstract := NewRegistry()
		rWithAbstract.RegisterTypes(
			&BaseOrganism{},
		)

		var to Organism
		err := rWithAbstract.BindValue(neo4j.Node{
			Labels: []string{"Human", "Organism"},
			Props: map[string]any{
				"alive": true,
				"name":  "Raqeeb",
			},
		}, reflect.ValueOf(&to))
		require.NoError(t, err)
		require.Equal(t, &Human{
			BaseOrganism: BaseOrganism{
				Alive: true,
			},
			Name: "Raqeeb",
		}, to)
	})

	t.Run("Abstract using registered concrete types", func(t *testing.T) {
		rWithAbstract := NewRegistry()
		rWithAbstract.RegisterTypes(
			&Human{},
			&Dog{},
		)
		var to Organism
		err := rWithAbstract.BindValue(neo4j.Node{
			Labels: []string{"Human", "Organism"},
			Props: map[string]any{
				"alive": true,
				"name":  "Raqeeb",
			},
		}, reflect.ValueOf(&to))
		require.NoError(t, err)
		require.Equal(t, &Human{
			BaseOrganism: BaseOrganism{
				Alive: true,
			},
			Name: "Raqeeb",
		}, to)
	})

	t.Run("Any", func(t *testing.T) {
		to := new(any)
		r.RegisterTypes(&ActedIn{})
		err := r.BindValue(neo4j.Relationship{
			Type: "ACTED_IN",
			Props: map[string]any{
				"role": "Stuntman",
			},
		}, reflect.ValueOf(to))
		require.NoError(t, err)
		require.Equal(t, neo4j.Relationship{
			Type: "ACTED_IN",
			Props: map[string]any{
				"role": "Stuntman",
			},
		}, *to)
	})

	t.Run("Slice binding", func(t *testing.T) {
		t.Run("[]int from []any", func(t *testing.T) {
			var bindTo []int64
			err := r.BindValue([]any{int64(1), int64(2), int64(3)}, reflect.ValueOf(&bindTo).Elem())
			require.NoError(t, err)
			require.Equal(t, []int64{1, 2, 3}, bindTo)
		})

		t.Run("[]string from []any", func(t *testing.T) {
			var bindTo []string
			err := r.BindValue([]any{"a", "b", "c"}, reflect.ValueOf(&bindTo).Elem())
			require.NoError(t, err)
			require.Equal(t, []string{"a", "b", "c"}, bindTo)
		})

		t.Run("[][]any nested", func(t *testing.T) {
			input1 := []any{1.0, "hello", true}
			input2 := []any{2.0, "bye", false}
			var bindTo [][]any
			err := r.BindValue([]any{input1, input2}, reflect.ValueOf(&bindTo).Elem())
			require.NoError(t, err)
			require.Len(t, bindTo, 2)
			require.Equal(t, input1, bindTo[0])
			require.Equal(t, input2, bindTo[1])
		})
	})
}
