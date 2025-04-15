package internal

import (
	"errors"
	"reflect"
	"testing"
	"time"

	"github.com/neo4j/neo4j-go-driver/v5/neo4j"
	"github.com/spf13/cast"
	"github.com/stretchr/testify/require"
)

type Organism interface {
	IAbstract
}

type BaseOrganism struct {
	Node
	Abstract `neo4j:"Organism"`
	Alive    bool `json:"alive"`
}

func (b BaseOrganism) Implementers() []IAbstract {
	return []IAbstract{
		&Human{},
		&Dog{},
	}
}

type Person struct {
	Node `neo4j:"Person"`
	Name string `json:"name"`
}

type ActedIn struct {
	Relationship `neo4j:"ACTED_IN"`
	Role         string `json:"role"`
}

type Human struct {
	BaseOrganism `neo4j:"Human"`
	Name         string `json:"name"`
}

type Dog struct {
	BaseOrganism `neo4j:"Dog"`
	Borfs        bool `json:"borfs"`
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
	} else {
		return &b.Value, nil
	}
}

func (b *simpleValuer[T]) Unmarshal(v *T) error {
	if b.shouldErr {
		return errors.New("intentional error")
	} else {
		b.Value = *v
	}
	return nil
}

func (b nodeValuer) Marshal() (*neo4j.Node, error) {
	if b.shouldErr {
		return nil, errors.New("intentional error")
	} else {
		return &neo4j.Node{Props: b.Value}, nil
	}
}

func (b *nodeValuer) Unmarshal(v *neo4j.Node) error {
	if b.shouldErr {
		return errors.New("intentional error")
	} else {
		b.Value = v.Props
	}
	return nil
}

func (b relationshipValuer) Marshal() (*neo4j.Relationship, error) {
	if b.shouldErr {
		return nil, errors.New("intentional error")
	} else {
		return &neo4j.Relationship{Props: b.Value}, nil
	}
}

func (b *relationshipValuer) Unmarshal(v *neo4j.Relationship) error {
	if b.shouldErr {
		return errors.New("intentional error")
	} else {
		b.Value = v.Props
	}
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

func TestBindCasted(t *testing.T) {
	t.Run("err when cast fails", func(t *testing.T) {
		bindTo := false
		err := bindCasted(cast.ToBoolE, "not a bool", reflect.ValueOf(&bindTo).Elem())
		require.Error(t, err)
	})

	t.Run("unmarshals to bindTo", func(t *testing.T) {
		bindTo := false
		err := bindCasted(cast.ToBoolE, "true", reflect.ValueOf(&bindTo).Elem())
		require.NoError(t, err)
		require.True(t, bindTo)
	})
}

func TestBindValue(t *testing.T) {
	r := NewRegistry()
	r.RegisterTypes(&BaseOrganism{})

	t.Run("Primitive coercion", func(t *testing.T) {
		t.Run("bool", func(t *testing.T) {
			bindTo := false
			err := r.BindValue(true, reflect.ValueOf(&bindTo).Elem())
			require.NoError(t, err)
			require.True(t, bindTo)
		})

		t.Run("string", func(t *testing.T) {
			bindTo := "no"
			err := r.BindValue(2.3, reflect.ValueOf(&bindTo).Elem())
			require.NoError(t, err)
			require.Equal(t, "2.3", bindTo)
		})

		t.Run("int", func(t *testing.T) {
			bindTo := 0
			err := r.BindValue("10", reflect.ValueOf(&bindTo).Elem())
			require.NoError(t, err)
			require.Equal(t, 10, bindTo)
		})

		t.Run("int8", func(t *testing.T) {
			bindTo := int8(0)
			err := r.BindValue("100", reflect.ValueOf(&bindTo).Elem())
			require.NoError(t, err)
			require.Equal(t, int8(100), bindTo)
		})

		t.Run("int16", func(t *testing.T) {
			bindTo := int16(0)
			err := r.BindValue("20000", reflect.ValueOf(&bindTo).Elem())
			require.NoError(t, err)
			require.Equal(t, int16(20000), bindTo)
		})

		t.Run("int32", func(t *testing.T) {
			bindTo := int32(0)
			err := r.BindValue("3000000", reflect.ValueOf(&bindTo).Elem())
			require.NoError(t, err)
			require.Equal(t, int32(3000000), bindTo)
		})

		t.Run("int64", func(t *testing.T) {
			bindTo := int64(0)
			err := r.BindValue("40000000000", reflect.ValueOf(&bindTo).Elem())
			require.NoError(t, err)
			require.Equal(t, int64(40000000000), bindTo)
		})

		t.Run("uint", func(t *testing.T) {
			bindTo := uint(0)
			err := r.BindValue("500", reflect.ValueOf(&bindTo).Elem())
			require.NoError(t, err)
			require.Equal(t, uint(500), bindTo)
		})

		t.Run("uint8", func(t *testing.T) {
			bindTo := uint8(0)
			err := r.BindValue("200", reflect.ValueOf(&bindTo).Elem())
			require.NoError(t, err)
			require.Equal(t, uint8(200), bindTo)
		})

		t.Run("uint16", func(t *testing.T) {
			bindTo := uint16(0)
			err := r.BindValue("60000", reflect.ValueOf(&bindTo).Elem())
			require.NoError(t, err)
			require.Equal(t, uint16(60000), bindTo)
		})

		t.Run("uint32", func(t *testing.T) {
			bindTo := uint32(0)
			err := r.BindValue("7000000", reflect.ValueOf(&bindTo).Elem())
			require.NoError(t, err)
			require.Equal(t, uint32(7000000), bindTo)
		})

		t.Run("uint64", func(t *testing.T) {
			bindTo := uint64(0)
			err := r.BindValue("80000000000", reflect.ValueOf(&bindTo).Elem())
			require.NoError(t, err)
			require.Equal(t, uint64(80000000000), bindTo)
		})

		t.Run("float32", func(t *testing.T) {
			bindTo := float32(0)
			err := r.BindValue("3.14", reflect.ValueOf(&bindTo).Elem())
			require.NoError(t, err)
			require.Equal(t, float32(3.14), bindTo)
		})

		t.Run("float64", func(t *testing.T) {
			bindTo := float64(0)
			err := r.BindValue("2.718", reflect.ValueOf(&bindTo).Elem())
			require.NoError(t, err)
			require.Equal(t, float64(2.718), bindTo)
		})

		t.Run("[]int", func(t *testing.T) {
			bindTo := []int{}
			err := r.BindValue([]any{1, 2, 3}, reflect.ValueOf(&bindTo).Elem())
			require.NoError(t, err)
			require.Equal(t, []int{1, 2, 3}, bindTo)
		})

		t.Run("[]string", func(t *testing.T) {
			bindTo := []string{}
			err := r.BindValue([]any{"a", "b", "c"}, reflect.ValueOf(&bindTo).Elem())
			require.NoError(t, err)
			require.Equal(t, []string{"a", "b", "c"}, bindTo)
		})

		t.Run("time.Time", func(t *testing.T) {
			bindTo := time.Time{}
			err := r.BindValue("2023-08-04T12:00:00Z", reflect.ValueOf(&bindTo).Elem())
			require.NoError(t, err)
			expected, _ := time.Parse(time.RFC3339, "2023-08-04T12:00:00Z")
			require.Equal(t, expected, bindTo)
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

		t.Run("[][]any", func(t *testing.T) {
			input1 := []any{1.0, "hello", true}
			input2 := []any{2.0, "bye", false}
			var bindTo [][]any
			err := r.BindValue([][]any{input1, input2}, reflect.ValueOf(&bindTo))
			require.NoError(t, err)
			require.Equal(t, input1, bindTo[0])
			require.Equal(t, input2, bindTo[1])
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
}
