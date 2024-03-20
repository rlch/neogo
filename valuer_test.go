package neogo

import (
	"errors"
	"reflect"
	"testing"
	"time"

	"github.com/neo4j/neo4j-go-driver/v5/neo4j"
	"github.com/spf13/cast"
	"github.com/stretchr/testify/assert"

	"github.com/rlch/neogo/internal/tests"
)

type (
	simpleValuer[T neo4j.RecordValue] struct {
		shouldErr bool
		Value     T
	}
	nodeValuer struct {
		shouldErr bool
		Value     map[string]any
	}
	relationshipValuer struct {
		shouldErr bool
		Value     map[string]any
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

func TestUnwindValue(t *testing.T) {
	t.Run("pointers to values", func(t *testing.T) {
		n := 10
		v := unwindValue(reflect.ValueOf(&n))
		assert.Equal(t, v.Interface(), n)
	})

	t.Run("values", func(t *testing.T) {
		n := 10
		v := unwindValue(reflect.ValueOf(n))
		assert.Equal(t, v.Interface(), n)
	})
}

func TestBindValuer(t *testing.T) {
	t.Run("err nil when not implemented", func(t *testing.T) {
		ok, err := bindValuer(false, reflect.ValueOf(10))
		assert.False(t, ok)
		assert.NoError(t, err)
	})

	t.Run("err when unmarshal fails", func(t *testing.T) {
		v := &simpleValuer[bool]{
			shouldErr: true,
		}
		ok, err := bindValuer(false, reflect.ValueOf(v))
		assert.False(t, ok)
		assert.Error(t, err)
	})

	t.Run("unmarshals to bindTo", func(t *testing.T) {
		v := &simpleValuer[bool]{}
		ok, err := bindValuer(true, reflect.ValueOf(v))
		assert.True(t, ok)
		assert.NoError(t, err)
		assert.True(t, v.Value)
	})
}

func TestBindCasted(t *testing.T) {
	t.Run("err when cast fails", func(t *testing.T) {
		bindTo := false
		err := bindCasted[bool](cast.ToBoolE, "not a bool", reflect.ValueOf(&bindTo).Elem())
		assert.Error(t, err)
	})

	t.Run("unmarshals to bindTo", func(t *testing.T) {
		bindTo := false
		err := bindCasted[bool](cast.ToBoolE, "true", reflect.ValueOf(&bindTo).Elem())
		assert.NoError(t, err)
		assert.True(t, bindTo)
	})
}

func TestBindValue(t *testing.T) {
	r := &registry{}

	t.Run("Primitive coercion", func(t *testing.T) {
		t.Run("bool", func(t *testing.T) {
			bindTo := false
			err := r.bindValue(true, reflect.ValueOf(&bindTo).Elem())
			assert.NoError(t, err)
			assert.True(t, bindTo)
		})

		t.Run("string", func(t *testing.T) {
			bindTo := "no"
			err := r.bindValue(2.3, reflect.ValueOf(&bindTo).Elem())
			assert.NoError(t, err)
			assert.Equal(t, "2.3", bindTo)
		})

		t.Run("int", func(t *testing.T) {
			bindTo := 0
			err := r.bindValue("10", reflect.ValueOf(&bindTo).Elem())
			assert.NoError(t, err)
			assert.Equal(t, 10, bindTo)
		})

		t.Run("int8", func(t *testing.T) {
			bindTo := int8(0)
			err := r.bindValue("100", reflect.ValueOf(&bindTo).Elem())
			assert.NoError(t, err)
			assert.Equal(t, int8(100), bindTo)
		})

		t.Run("int16", func(t *testing.T) {
			bindTo := int16(0)
			err := r.bindValue("20000", reflect.ValueOf(&bindTo).Elem())
			assert.NoError(t, err)
			assert.Equal(t, int16(20000), bindTo)
		})

		t.Run("int32", func(t *testing.T) {
			bindTo := int32(0)
			err := r.bindValue("3000000", reflect.ValueOf(&bindTo).Elem())
			assert.NoError(t, err)
			assert.Equal(t, int32(3000000), bindTo)
		})

		t.Run("int64", func(t *testing.T) {
			bindTo := int64(0)
			err := r.bindValue("40000000000", reflect.ValueOf(&bindTo).Elem())
			assert.NoError(t, err)
			assert.Equal(t, int64(40000000000), bindTo)
		})

		t.Run("uint", func(t *testing.T) {
			bindTo := uint(0)
			err := r.bindValue("500", reflect.ValueOf(&bindTo).Elem())
			assert.NoError(t, err)
			assert.Equal(t, uint(500), bindTo)
		})

		t.Run("uint8", func(t *testing.T) {
			bindTo := uint8(0)
			err := r.bindValue("200", reflect.ValueOf(&bindTo).Elem())
			assert.NoError(t, err)
			assert.Equal(t, uint8(200), bindTo)
		})

		t.Run("uint16", func(t *testing.T) {
			bindTo := uint16(0)
			err := r.bindValue("60000", reflect.ValueOf(&bindTo).Elem())
			assert.NoError(t, err)
			assert.Equal(t, uint16(60000), bindTo)
		})

		t.Run("uint32", func(t *testing.T) {
			bindTo := uint32(0)
			err := r.bindValue("7000000", reflect.ValueOf(&bindTo).Elem())
			assert.NoError(t, err)
			assert.Equal(t, uint32(7000000), bindTo)
		})

		t.Run("uint64", func(t *testing.T) {
			bindTo := uint64(0)
			err := r.bindValue("80000000000", reflect.ValueOf(&bindTo).Elem())
			assert.NoError(t, err)
			assert.Equal(t, uint64(80000000000), bindTo)
		})

		t.Run("float32", func(t *testing.T) {
			bindTo := float32(0)
			err := r.bindValue("3.14", reflect.ValueOf(&bindTo).Elem())
			assert.NoError(t, err)
			assert.Equal(t, float32(3.14), bindTo)
		})

		t.Run("float64", func(t *testing.T) {
			bindTo := float64(0)
			err := r.bindValue("2.718", reflect.ValueOf(&bindTo).Elem())
			assert.NoError(t, err)
			assert.Equal(t, float64(2.718), bindTo)
		})

		t.Run("[]int", func(t *testing.T) {
			bindTo := []int{}
			err := r.bindValue([]string{"1", "2", "3"}, reflect.ValueOf(&bindTo).Elem())
			assert.NoError(t, err)
			assert.Equal(t, []int{1, 2, 3}, bindTo)
		})

		t.Run("[]string", func(t *testing.T) {
			bindTo := []string{}
			err := r.bindValue([]int{10, 20, 30}, reflect.ValueOf(&bindTo).Elem())
			assert.NoError(t, err)
			assert.Equal(t, []string{"10", "20", "30"}, bindTo)
		})

		t.Run("time.Time", func(t *testing.T) {
			bindTo := time.Time{}
			err := r.bindValue("2023-08-04T12:00:00Z", reflect.ValueOf(&bindTo).Elem())
			assert.NoError(t, err)
			expected, _ := time.Parse(time.RFC3339, "2023-08-04T12:00:00Z")
			assert.Equal(t, expected, bindTo)
		})
	})

	t.Run("Valuer", func(t *testing.T) {
		t.Run("bool", func(t *testing.T) {
			bindTo := &simpleValuer[bool]{}
			err := r.bindValue(true, reflect.ValueOf(bindTo))
			assert.NoError(t, err)
			assert.True(t, bindTo.Value)
		})

		t.Run("int64", func(t *testing.T) {
			bindTo := &simpleValuer[int64]{}
			err := r.bindValue(int64(100), reflect.ValueOf(bindTo))
			assert.NoError(t, err)
			assert.Equal(t, int64(100), bindTo.Value)
		})

		t.Run("string", func(t *testing.T) {
			bindTo := &simpleValuer[string]{}
			err := r.bindValue("hello", reflect.ValueOf(bindTo))
			assert.NoError(t, err)
			assert.Equal(t, "hello", bindTo.Value)
		})

		t.Run("float64", func(t *testing.T) {
			bindTo := &simpleValuer[float64]{}
			err := r.bindValue(3.14, reflect.ValueOf(bindTo))
			assert.NoError(t, err)
			assert.Equal(t, 3.14, bindTo.Value)
		})

		t.Run("time.Time", func(t *testing.T) {
			inputTime := time.Date(2023, time.August, 4, 12, 0, 0, 0, time.UTC)
			bindTo := &simpleValuer[time.Time]{}
			err := r.bindValue(inputTime, reflect.ValueOf(bindTo))
			assert.NoError(t, err)
			assert.Equal(t, inputTime, bindTo.Value)
		})

		t.Run("[]byte", func(t *testing.T) {
			input := []byte{0x68, 0x65, 0x6c, 0x6c, 0x6f}
			bindTo := &simpleValuer[[]byte]{}
			err := r.bindValue(input, reflect.ValueOf(bindTo))
			assert.NoError(t, err)
			assert.Equal(t, input, bindTo.Value)
		})

		t.Run("[]any", func(t *testing.T) {
			input := []any{1, "hello", true}
			bindTo := &simpleValuer[[]any]{}
			err := r.bindValue(input, reflect.ValueOf(bindTo))
			assert.NoError(t, err)
			assert.Equal(t, input, bindTo.Value)
		})

		t.Run("map[string]any", func(t *testing.T) {
			input := map[string]any{"name": "John", "age": 30}
			bindTo := &simpleValuer[map[string]any]{}
			err := r.bindValue(input, reflect.ValueOf(bindTo))
			assert.NoError(t, err)
			assert.Equal(t, input, bindTo.Value)
		})

		t.Run("Node", func(t *testing.T) {
			input := neo4j.Node{
				Props: map[string]any{
					"name": "Richard",
				},
			}
			bindTo := &nodeValuer{}
			err := r.bindValue(input, reflect.ValueOf(bindTo))
			assert.NoError(t, err)
			assert.Equal(t, map[string]any{
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
			err := r.bindValue(input, reflect.ValueOf(bindTo))
			assert.NoError(t, err)
			assert.Equal(t, map[string]any{
				"weight": 0.5,
			}, bindTo.Value)
		})
	})

	t.Run("Node", func(t *testing.T) {
		to := &tests.Person{}
		err := r.bindValue(neo4j.Node{
			Labels: []string{"Person"},
			Props: map[string]any{
				"name":    "Richard",
				"surname": "Mathieson",
				"age":     24,
			},
		}, reflect.ValueOf(to))
		assert.NoError(t, err)
		assert.Equal(t, tests.Person{
			Name:    "Richard",
			Surname: "Mathieson",
			Age:     24,
		}, *to)
	})

	t.Run("Relationship", func(t *testing.T) {
		to := &tests.ActedIn{}
		err := r.bindValue(neo4j.Node{
			Labels: []string{"ACTED_IN"},
			Props: map[string]any{
				"role": "Stuntman",
			},
		}, reflect.ValueOf(to))
		assert.NoError(t, err)
		assert.Equal(t, tests.ActedIn{
			Role: "Stuntman",
		}, *to)
	})

	t.Run("Abstract using base type", func(t *testing.T) {
		var to tests.Organism = &tests.BaseOrganism{}
		err := r.bindValue(neo4j.Node{
			Labels: []string{"Human", "Organism"},
			Props: map[string]any{
				"name": "bruh",
			},
		}, reflect.ValueOf(&to))
		assert.NoError(t, err)
		assert.Equal(t, &tests.Human{
			Name: "bruh",
		}, to)
	})

	t.Run("Abstract using registered types", func(t *testing.T) {
		rWithAbstract := &registry{
			abstractNodes: []IAbstract{
				&tests.BaseOrganism{},
			},
		}
		var to tests.Organism
		err := rWithAbstract.bindValue(neo4j.Node{
			Labels: []string{"Human", "Organism"},
			Props: map[string]any{
				"alive": true,
				"name":  "Raqeeb",
			},
		}, reflect.ValueOf(&to))
		assert.NoError(t, err)
		assert.Equal(t, &tests.Human{
			BaseOrganism: tests.BaseOrganism{
				Alive: true,
			},
			Name: "Raqeeb",
		}, to)
	})

	t.Run("Abstract using registered concrete types", func(t *testing.T) {
		rWithAbstract := &registry{
			abstractNodes: []IAbstract{
				&tests.Human{},
				&tests.Dog{},
			},
		}
		var to tests.Organism
		err := rWithAbstract.bindValue(neo4j.Node{
			Labels: []string{"Human", "Organism"},
			Props: map[string]any{
				"alive": true,
				"name":  "Raqeeb",
			},
		}, reflect.ValueOf(&to))
		assert.NoError(t, err)
		assert.Equal(t, &tests.Human{
			BaseOrganism: tests.BaseOrganism{
				Alive: true,
			},
			Name: "Raqeeb",
		}, to)
	})

	t.Run("Any", func(t *testing.T) {
		to := new(any)
		err := r.bindValue(neo4j.Node{
			Labels: []string{"ACTED_IN"},
			Props: map[string]any{
				"role": "Stuntman",
			},
		}, reflect.ValueOf(to))
		assert.NoError(t, err)
		assert.Equal(t, neo4j.Node{
			Labels: []string{"ACTED_IN"},
			Props: map[string]any{
				"role": "Stuntman",
			},
		}, *to)
	})
}
