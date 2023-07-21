package neogo

import (
	"reflect"
	"testing"

	"github.com/neo4j/neo4j-go-driver/v5/neo4j"
	"github.com/stretchr/testify/assert"

	"github.com/rlch/neogo/internal"
	"github.com/rlch/neogo/internal/tests"
)

func TestUnmarshalResult(t *testing.T) {
	s := &session{}
	t.Run("single", func(t *testing.T) {
		t.Run("err on non-existent key", func(t *testing.T) {
			n := tests.Person{}
			cy := &internal.CompiledCypher{
				Bindings: map[string]reflect.Value{
					"m": reflect.ValueOf(&n),
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
			n := tests.Person{}
			cy := &internal.CompiledCypher{
				Bindings: map[string]reflect.Value{
					"n": reflect.ValueOf(&n),
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
			assert.Equal(t, tests.Person{
				Name: "Jessie", Surname: "Pinkman",
			}, n)
		})

		t.Run("binds to abstract node", func(t *testing.T) {
			var n tests.Organism = &tests.BaseOrganism{}
			cy := &internal.CompiledCypher{
				Bindings: map[string]reflect.Value{
					"n": reflect.ValueOf(&n),
				},
			}
			record := &neo4j.Record{
				Keys: []string{"n"},
				Values: []any{
					neo4j.Node{
						Labels: []string{
							"Organism",
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
			assert.Equal(t, &tests.Dog{
				BaseOrganism: tests.BaseOrganism{
					Node: internal.Node{
						ID: "dog",
					},
					Alive: true,
				},
				Borfs: true,
			}, n)
		})
	})

	t.Run("collection", func(t *testing.T) {
		t.Run("err on non-existent key", func(t *testing.T) {
			n1 := tests.Person{}
			cy := &internal.CompiledCypher{
				Bindings: map[string]reflect.Value{
					"n": reflect.ValueOf(&n1),
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
			var n []*tests.Person
			cy := &internal.CompiledCypher{
				Bindings: map[string]reflect.Value{
					"n": reflect.ValueOf(&n),
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
			assert.Equal(t, tests.Person{
				Name: "Jessie", Surname: "Pinkman",
			}, *n[0])
			assert.Equal(t, tests.Person{
				Name: "Walter", Surname: "White",
			}, *n[1])
		})

		t.Run("binds to []any", func(t *testing.T) {
			var n []any
			cy := &internal.CompiledCypher{
				Bindings: map[string]reflect.Value{
					"n": reflect.ValueOf(&n),
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

		t.Run("binds to abstract nodes", func(t *testing.T) {
			s := &session{
				registry: registry{
					abstractNodes: []IAbstract{
						&tests.BaseOrganism{},
					},
				},
			}
			var n []tests.Organism
			cy := &internal.CompiledCypher{
				Bindings: map[string]reflect.Value{
					"n": reflect.ValueOf(&n),
				},
			}
			records := []*neo4j.Record{
				{
					Keys: []string{"n"},
					Values: []any{
						neo4j.Node{
							Labels: []string{
								"Organism",
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
			assert.Equal(t, &tests.Dog{
				BaseOrganism: tests.BaseOrganism{
					Node: internal.Node{
						ID: "dog",
					},
					Alive: true,
				},
				Borfs: true,
			}, n[0])
			assert.Equal(t, &tests.Human{
				BaseOrganism: tests.BaseOrganism{
					Node: internal.Node{
						ID: "human",
					},
					Alive: true,
				},
				Name: "Jesse Pinkman",
			}, n[1])
		})
	})
}
