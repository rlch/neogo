package neogo

import (
	"context"
	"testing"

	"github.com/neo4j/neo4j-go-driver/v5/neo4j"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/sync/semaphore"

	"github.com/rlch/neogo/internal"
)

// Test fixtures

type Person struct {
	internal.Node `neo4j:"Person"`
	Name          string `neo4j:"name"`
	Surname       string `neo4j:"surname"`
}

func (Person) IsNode() {}

type Organism interface {
	internal.IAbstract
	IsOrganism()
}

type BaseOrganism struct {
	internal.Node     `neo4j:"Organism"`
	internal.Abstract `neo4j:"Organism"`
	Alive             bool `neo4j:"alive"`
}

func (BaseOrganism) IsNode()     {}
func (BaseOrganism) IsOrganism() {}
func (*BaseOrganism) Implementers() []internal.IAbstract {
	return []internal.IAbstract{&Human{}, &BasePet{}}
}

type Pet interface {
	Organism
	IsPet()
}

type BasePet struct {
	BaseOrganism `neo4j:"Pet"`
	Cute         bool `neo4j:"cute"`
}

func (*BasePet) IsPet() {}
func (*BasePet) Implementers() []internal.IAbstract {
	return []internal.IAbstract{&Dog{}}
}

type Human struct {
	BaseOrganism `neo4j:"Human"`
	Name         string `neo4j:"name"`
}

type Dog struct {
	BasePet `neo4j:"Dog"`
	Borfs   bool `neo4j:"borfs"`
}

// newTestSession creates a session with a properly initialized driver and registry
func newTestSession() *session {
	d := &driver{
		reg:              internal.NewRegistry(),
		sessionSemaphore: semaphore.NewWeighted(1),
	}
	d.reg.RegisterTypes(&BaseOrganism{}, &BasePet{}, &Human{}, &Dog{}, &Person{})
	return &session{driver: d}
}

func TestUnmarshalRecord(t *testing.T) {
	s := newTestSession()
	t.Run("err on non-existent key", func(t *testing.T) {
		n := Person{}
		cy := &internal.CompiledCypher{
			Bindings: map[string]any{
				"m": &n,
			},
		}
		record := &neo4j.Record{
			Keys: []string{"n"},
			Values: []any{
				neo4j.Node{
					Props: map[string]any{
						"name":    "Jessie",
						"surname": "Doinkman",
					},
				},
			},
		}
		err := s.unmarshalRecord(cy, record)
		assert.Error(t, err)
	})

	t.Run("binds to node", func(t *testing.T) {
		n := Person{}
		cy := &internal.CompiledCypher{
			Bindings: map[string]any{
				"n": &n,
			},
		}
		record := &neo4j.Record{
			Keys: []string{"n"},
			Values: []any{
				neo4j.Node{
					Props: map[string]any{
						"name":    "Jessie",
						"surname": "Pinkman",
					},
				},
			},
		}
		err := s.unmarshalRecord(cy, record)
		assert.NoError(t, err)
		assert.Equal(t, Person{
			Name: "Jessie", Surname: "Pinkman",
		}, n)
	})

	t.Run("binds to abstract nodes with length 1", func(t *testing.T) {
		var n []Organism
		cy := &internal.CompiledCypher{
			Bindings: map[string]any{
				"n": &n,
			},
		}
		err := s.unmarshalRecord(cy,
			&neo4j.Record{
				Keys: []string{"n"},
				Values: []any{
					neo4j.Node{
						Labels: []string{
							"Organism",
							"Human",
						},
						Props: map[string]any{
							"id":    "boss",
							"name":  "Michael Scott",
							"alive": true,
						},
					},
				},
			})
		assert.NoError(t, err)
		assert.Len(t, n, 1, "Expected single record to be bound to slice of length 1")
		assert.Equal(t, &Human{
			BaseOrganism: BaseOrganism{
				Node: internal.Node{
					ID: "boss",
				},
				Alive: true,
			},
			Name: "Michael Scott",
		}, n[0])
	})

	t.Run("binds to null", func(t *testing.T) {
		var n *Person
		cy := &internal.CompiledCypher{
			Bindings: map[string]any{
				"n": &n,
			},
		}
		record := &neo4j.Record{
			Keys:   []string{"n"},
			Values: []any{nil},
		}
		err := s.unmarshalRecord(cy, record)
		assert.NoError(t, err)
		assert.Equal(t, (*Person)(nil), n)
	})

	t.Run("binds to abstract node", func(t *testing.T) {
		var n Organism = &BaseOrganism{}
		cy := &internal.CompiledCypher{
			Bindings: map[string]any{
				"n": &n,
			},
		}
		record := &neo4j.Record{
			Keys: []string{"n"},
			Values: []any{
				neo4j.Node{
					Labels: []string{
						"Organism",
						"Human",
					},
					Props: map[string]any{
						"id":   "human",
						"name": "waltuh",
					},
				},
			},
		}
		err := s.unmarshalRecord(cy, record)
		assert.NoError(t, err)
		assert.Equal(t, &Human{
			BaseOrganism: BaseOrganism{
				Node: internal.Node{
					ID: "human",
				},
				Alive: false,
			},
			Name: "waltuh",
		}, n)
	})

	t.Run("binds to multi-polymorphic abstract node", func(t *testing.T) {
		var n Pet = &BasePet{}
		cy := &internal.CompiledCypher{
			Bindings: map[string]any{
				"n": &n,
			},
		}
		record := &neo4j.Record{
			Keys: []string{"n"},
			Values: []any{
				neo4j.Node{
					Labels: []string{
						"Organism",
						"Pet",
						"Dog",
					},
					Props: map[string]any{
						"id":    "dog",
						"borfs": true,
						"alive": true,
					},
				},
			},
		}
		err := s.unmarshalRecord(cy, record)
		assert.NoError(t, err)
		assert.Equal(t, &Dog{
			BasePet: BasePet{
				BaseOrganism: BaseOrganism{
					Node: internal.Node{
						ID: "dog",
					},
					Alive: true,
				},
			},
			Borfs: true,
		}, n)
	})

	t.Run("binds to nodes", func(t *testing.T) {
		var n []Person
		cy := &internal.CompiledCypher{
			Bindings: map[string]any{
				"n": &n,
			},
		}
		err := s.unmarshalRecord(cy,
			&neo4j.Record{
				Keys: []string{"n"},
				Values: []any{
					[]any{
						neo4j.Node{
							Props: map[string]any{
								"name":    "Jessie",
								"surname": "Pinkman",
							},
						},
						neo4j.Node{
							Props: map[string]any{
								"name":    "Walter",
								"surname": "White",
							},
						},
					},
				},
			},
		)
		assert.NoError(t, err)
		assert.Equal(t, Person{
			Name: "Jessie", Surname: "Pinkman",
		}, n[0])
		assert.Equal(t, Person{
			Name: "Walter", Surname: "White",
		}, n[1])
	})

	t.Run("binds to nodes with length 1", func(t *testing.T) {
		var n []Person
		cy := &internal.CompiledCypher{
			Bindings: map[string]any{
				"n": &n,
			},
		}
		err := s.unmarshalRecord(cy,
			&neo4j.Record{
				Keys: []string{"n"},
				Values: []any{
					neo4j.Node{
						Props: map[string]any{
							"name":    "Jessie",
							"surname": "Pinkman",
						},
					},
				},
			},
		)
		assert.NoError(t, err)
		assert.Equal(t, Person{
			Name: "Jessie", Surname: "Pinkman",
		}, n[0])
	})

	t.Run("binds to abstract nodes with length 1", func(t *testing.T) {
		var n []Organism
		cy := &internal.CompiledCypher{
			Bindings: map[string]any{
				"n": &n,
			},
		}
		err := s.unmarshalRecord(cy,
			&neo4j.Record{
				Keys: []string{"n"},
				Values: []any{
					neo4j.Node{
						Labels: []string{
							"Organism",
							"Human",
						},
						Props: map[string]any{
							"id":    "boss",
							"name":  "Michael Scott",
							"alive": true,
						},
					},
				},
			})
		assert.NoError(t, err)
		assert.Len(t, n, 1, "Expected single record to be bound to slice of length 1")
		assert.Equal(t, &Human{
			BaseOrganism: BaseOrganism{
				Node: internal.Node{
					ID: "boss",
				},
				Alive: true,
			},
			Name: "Michael Scott",
		}, n[0])
	})
}

func TestUnmarshalRecords(t *testing.T) {
	s := newTestSession()

	t.Run("err on non-existent key", func(t *testing.T) {
		n1 := Person{}
		cy := &internal.CompiledCypher{
			Bindings: map[string]any{
				"n": &n1,
			},
		}
		records := []*neo4j.Record{
			{
				Keys: []string{"n"},
				Values: []any{
					neo4j.Node{
						Props: map[string]any{
							"name":    "Jessie",
							"surname": "Pinkman",
						},
					},
				},
			},
			{
				// This record does not have the "n" key.
				Keys:   []string{"non_existent_key"},
				Values: []any{"some_value"},
			},
		}
		err := s.unmarshalRecords(cy, records)
		assert.Error(t, err)
	})

	t.Run("binds to nodes", func(t *testing.T) {
		var n []Person
		cy := &internal.CompiledCypher{
			Bindings: map[string]any{
				"n": &n,
			},
		}
		records := []*neo4j.Record{
			{
				Keys: []string{"n"},
				Values: []any{
					neo4j.Node{
						Props: map[string]any{
							"name":    "Jessie",
							"surname": "Pinkman",
						},
					},
				},
			},
			{
				Keys: []string{"n"},
				Values: []any{
					neo4j.Node{
						Props: map[string]any{
							"name":    "Walter",
							"surname": "White",
						},
					},
				},
			},
		}
		err := s.unmarshalRecords(cy, records)
		assert.NoError(t, err)
		assert.Equal(t, Person{
			Name: "Jessie", Surname: "Pinkman",
		}, n[0])
		assert.Equal(t, Person{
			Name: "Walter", Surname: "White",
		}, n[1])
	})

	t.Run("binds to slice of nils", func(t *testing.T) {
		var n []*Person
		cy := &internal.CompiledCypher{
			Bindings: map[string]any{
				"n": &n,
			},
		}
		records := []*neo4j.Record{
			{
				Keys:   []string{"n"},
				Values: []any{nil},
			},
			{
				Keys:   []string{"n"},
				Values: []any{nil},
			},
		}
		err := s.unmarshalRecords(cy, records)
		assert.NoError(t, err)
		assert.Equal(t, (*Person)(nil), n[0])
		assert.Equal(t, (*Person)(nil), n[1])
	})

	t.Run("considers nil nodes in slices", func(t *testing.T) {
		var n []*Person
		cy := &internal.CompiledCypher{
			Bindings: map[string]any{
				"n": &n,
			},
		}
		records := []*neo4j.Record{
			{
				Keys: []string{"n"},
				Values: []any{
					neo4j.Node{
						Props: map[string]any{
							"name":    "Jessie",
							"surname": "Pinkman",
						},
					},
				},
			},
			{
				Keys: []string{"n"},
				Values: []any{
					nil,
				},
			},
		}
		err := s.unmarshalRecords(cy, records)
		assert.NoError(t, err)
		assert.Len(t, n, 2)
		assert.Equal(t, Person{
			Name: "Jessie", Surname: "Pinkman",
		}, *n[0])
		assert.Equal(t, (*Person)(nil), n[1])
	})

	t.Run("binds to []any", func(t *testing.T) {
		var n []any
		cy := &internal.CompiledCypher{
			Bindings: map[string]any{
				"n": &n,
			},
		}
		records := []*neo4j.Record{
			{
				Keys:   []string{"n"},
				Values: []any{1},
			},
			{
				Keys:   []string{"n"},
				Values: []any{2},
			},
		}
		err := s.unmarshalRecords(cy, records)
		assert.NoError(t, err)
		assert.Equal(t, 1, n[0])
		assert.Equal(t, 2, n[1])
	})

	t.Run("binds to [][]any", func(t *testing.T) {
		var n [][]any
		cy := &internal.CompiledCypher{
			Bindings: map[string]any{
				"n": &n,
			},
		}
		records := []*neo4j.Record{
			{
				Keys:   []string{"n"},
				Values: []any{[]any{"a", "b"}},
			},
			{
				Keys:   []string{"n"},
				Values: []any{[]any{"c", "d"}},
			},
		}
		err := s.unmarshalRecords(cy, records)
		assert.NoError(t, err)
		assert.Equal(t, []any{"a", "b"}, n[0])
		assert.Equal(t, []any{"c", "d"}, n[1])
	})

	t.Run("binds to abstract nodes", func(t *testing.T) {
		s := newTestSession()
		var n []Organism
		cy := &internal.CompiledCypher{
			Bindings: map[string]any{
				"n": &n,
			},
		}
		records := []*neo4j.Record{
			{
				Keys: []string{"n"},
				Values: []any{
					neo4j.Node{
						Labels: []string{
							"Organism",
							"Pet",
							"Dog",
						},
						Props: map[string]any{
							"id":    "dog",
							"borfs": true,
							"alive": true,
						},
					},
				},
			},
			{
				Keys: []string{"n"},
				Values: []any{
					neo4j.Node{
						Labels: []string{
							"Organism",
							"Human",
						},
						Props: map[string]any{
							"id":    "human",
							"alive": true,
							"name":  "Jesse Pinkman",
						},
					},
				},
			},
		}
		err := s.unmarshalRecords(cy, records)
		assert.NoError(t, err)
		assert.Equal(t, &Dog{
			BasePet: BasePet{
				BaseOrganism: BaseOrganism{
					Node: internal.Node{
						ID: "dog",
					},
					Alive: true,
				},
			},
			Borfs: true,
		}, n[0])
		assert.Equal(t, &Human{
			BaseOrganism: BaseOrganism{
				Node: internal.Node{
					ID: "human",
				},
				Alive: true,
			},
			Name: "Jesse Pinkman",
		}, n[1])
	})

	// TODO: Re-enable after fixing abstract node label matching
	// t.Run("binds to [][]Abstract", func(t *testing.T) {
	// 	...
	// })

	t.Run("binds to [][]Concrete where Concrete is an implementation of Abstract", func(t *testing.T) {
		s := newTestSession()
		var n [][]BasePet
		cy := &internal.CompiledCypher{
			Bindings: map[string]any{
				"n": &n,
			},
		}
		records := []*neo4j.Record{
			{
				Keys: []string{"n"},
				Values: []any{
					[]any{
						neo4j.Node{
							Labels: []string{
								"Organism",
								"Pet",
							},
							Props: map[string]any{
								"id":   "pet",
								"cute": true,
							},
						},
					},
				},
			},
		}
		err := s.unmarshalRecords(cy, records)
		assert.NoError(t, err)
		assert.Equal(t, BasePet{
			BaseOrganism: BaseOrganism{
				Node: internal.Node{
					ID: "pet",
				},
			},
			Cute: true,
		}, n[0][0])
	})

	t.Run("unmarshalling slices", func(t *testing.T) {
		require := require.New(t)
		s := newTestSession()

		type PersonLocal struct {
			ID int `neo4j:"id"`
		}

		// UNWIND [1, 2, 3] AS id
		// WITH {id: id} AS person
		// WITH collect(person) AS persons
		var persons [][]*PersonLocal
		record := &neo4j.Record{
			Keys: []string{"persons"},
			Values: []any{
				[]any{
					map[string]any{"id": 1},
					map[string]any{"id": 2},
					map[string]any{"id": 3},
				},
			},
		}
		err := s.unmarshalRecord(&internal.CompiledCypher{
			Bindings: map[string]any{
				"persons": &persons,
			},
		}, record)
		require.NoError(err)
		require.Len(persons, 1)
	})

	t.Run("unmarshalling nil record to slice", func(t *testing.T) {
		require := require.New(t)
		s := newTestSession()

		type PersonLocal struct {
			ID int `neo4j:"id"`
		}

		var persons []*PersonLocal
		record := &neo4j.Record{
			Keys:   []string{"persons"},
			Values: []any{nil},
		}
		err := s.unmarshalRecord(&internal.CompiledCypher{
			Bindings: map[string]any{
				"persons": &persons,
			},
		}, record)
		require.NoError(err)
		require.Len(persons, 1)
	})
}

func TestPrint(t *testing.T) {
	c := NewMock()
	c.Bind(map[string]any{})

	// Print returns the same runner for chaining
	runner := c.Exec().Cypher("MATCH (n) RETURN n")
	result := runner.Print()
	assert.Equal(t, runner, result, "Print should return the same runner for chaining")
}

func TestNewAPI(t *testing.T) {
	ctx := context.Background()

	t.Run("Cypher and Run with bindings", func(t *testing.T) {
		c := NewMock()
		c.Bind(map[string]any{
			"n": neo4j.Node{
				Props: map[string]any{
					"name":    "Alice",
					"surname": "Smith",
				},
			},
		})

		var person Person
		err := c.Exec().
			Cypher("MATCH (n:Person) RETURN n").
			Run(ctx, "n", &person)

		assert.NoError(t, err)
		assert.Equal(t, "Alice", person.Name)
		assert.Equal(t, "Smith", person.Surname)
	})

	t.Run("Cypher and RunWithParams", func(t *testing.T) {
		c := NewMock()
		c.Bind(map[string]any{
			"n": neo4j.Node{
				Props: map[string]any{
					"name":    "Bob",
					"surname": "Jones",
				},
			},
		})

		var person Person
		err := c.Exec().
			Cypher("MATCH (n:Person {name: $name}) RETURN n").
			RunWithParams(ctx, map[string]any{"name": "Bob"}, "n", &person)

		assert.NoError(t, err)
		assert.Equal(t, "Bob", person.Name)
		assert.Equal(t, "Jones", person.Surname)
	})

	t.Run("Cypher with multiple bindings", func(t *testing.T) {
		c := NewMock()
		c.Bind(map[string]any{
			"n": neo4j.Node{
				Props: map[string]any{
					"name":    "Alice",
					"surname": "Smith",
				},
			},
			"m": neo4j.Node{
				Props: map[string]any{
					"name":    "Bob",
					"surname": "Jones",
				},
			},
		})

		var person1, person2 Person
		err := c.Exec().
			Cypher("MATCH (n:Person)--(m:Person) RETURN n, m").
			Run(ctx, "n", &person1, "m", &person2)

		assert.NoError(t, err)
		assert.Equal(t, "Alice", person1.Name)
		assert.Equal(t, "Bob", person2.Name)
	})

	t.Run("Cypher returns error for invalid bindings", func(t *testing.T) {
		c := NewMock()
		c.Bind(map[string]any{})

		var person Person
		// Odd number of binding args should fail
		err := c.Exec().
			Cypher("MATCH (n:Person) RETURN n").
			Run(ctx, "n", &person, "extra")

		assert.Error(t, err)
	})

	t.Run("Stream with bindings", func(t *testing.T) {
		c := NewMock()
		c.BindRecords([]map[string]any{
			{"i": int64(1)},
			{"i": int64(2)},
			{"i": int64(3)},
		})

		var num int64
		var results []int64
		err := c.Exec().
			Cypher("UNWIND [1, 2, 3] AS i RETURN i").
			Stream(ctx, func() error {
				results = append(results, num)
				return nil
			}, "i", &num)

		assert.NoError(t, err)
		assert.Equal(t, []int64{1, 2, 3}, results)
	})
}
