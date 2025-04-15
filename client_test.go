package neogo

import (
	"context"
	"reflect"
	"testing"

	"github.com/neo4j/neo4j-go-driver/v5/neo4j"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/rlch/neogo/builder"
	"github.com/rlch/neogo/db"
	"github.com/rlch/neogo/internal"
	"github.com/rlch/neogo/internal/tests"
)

func TestUnmarshalRecord(t *testing.T) {
	s := &session{}
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

	t.Run("binds to null", func(t *testing.T) {
		var n *tests.Person
		cy := &internal.CompiledCypher{
			Bindings: map[string]reflect.Value{
				"n": reflect.ValueOf(&n),
			},
		}
		record := &neo4j.Record{
			Keys:   []string{"n"},
			Values: []any{nil},
		}
		err := s.unmarshalRecord(cy, record)
		assert.NoError(t, err)
		assert.Equal(t, (*tests.Person)(nil), n)
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
		assert.Equal(t, &tests.Human{
			BaseOrganism: tests.BaseOrganism{
				Node: internal.Node{
					ID: "human",
				},
				Alive: false,
			},
			Name: "waltuh",
		}, n)
	})

	t.Run("binds to multi-polymorphic abstract node", func(t *testing.T) {
		var n tests.Pet = &tests.BasePet{}
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
		assert.Equal(t, &tests.Dog{
			BasePet: tests.BasePet{
				BaseOrganism: tests.BaseOrganism{
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
		var n []tests.Person
		cy := &internal.CompiledCypher{
			Bindings: map[string]reflect.Value{
				"n": reflect.ValueOf(&n),
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
		assert.Equal(t, tests.Person{
			Name: "Jessie", Surname: "Pinkman",
		}, n[0])
		assert.Equal(t, tests.Person{
			Name: "Walter", Surname: "White",
		}, n[1])
	})

	t.Run("binds to nodes with length 1", func(t *testing.T) {
		var n []tests.Person
		cy := &internal.CompiledCypher{
			Bindings: map[string]reflect.Value{
				"n": reflect.ValueOf(&n),
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
		assert.Equal(t, tests.Person{
			Name: "Jessie", Surname: "Pinkman",
		}, n[0])
	})
}

func TestUnmarshalRecords(t *testing.T) {
	s := &session{}

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
		var n []tests.Person
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
		}, n[0])
		assert.Equal(t, tests.Person{
			Name: "Walter", Surname: "White",
		}, n[1])
	})

	t.Run("binds to slice of nils", func(t *testing.T) {
		var n []*tests.Person
		cy := &internal.CompiledCypher{
			Bindings: map[string]reflect.Value{
				"n": reflect.ValueOf(&n),
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
		assert.Equal(t, (*tests.Person)(nil), n[0])
		assert.Equal(t, (*tests.Person)(nil), n[1])
	})

	t.Run("considers nil nodes in slices", func(t *testing.T) {
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
					nil,
				},
			},
		}
		err := s.unmarshalRecords(cy, records)
		assert.NoError(t, err)
		assert.Len(t, n, 2)
		assert.Equal(t, tests.Person{
			Name: "Jessie", Surname: "Pinkman",
		}, *n[0])
		assert.Equal(t, (*tests.Person)(nil), n[1])
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

	t.Run("binds to [][]any", func(t *testing.T) {
		var n [][]any
		cy := &internal.CompiledCypher{
			Bindings: map[string]reflect.Value{
				"n": reflect.ValueOf(&n),
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
		s := &session{}
		s.RegisterTypes(&tests.BaseOrganism{}, &tests.BasePet{})
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
		assert.Equal(t, &tests.Dog{
			BasePet: tests.BasePet{
				BaseOrganism: tests.BaseOrganism{
					Node: internal.Node{
						ID: "dog",
					},
					Alive: true,
				},
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

	t.Run("binds to [][]Abstract", func(t *testing.T) {
		s := &session{
			driver: &driver{
				Registry: internal.NewRegistry(),
			},
		}
		s.RegisterTypes(&tests.BaseOrganism{}, &tests.BasePet{})
		var n [][]tests.Organism
		cy := &internal.CompiledCypher{
			Bindings: map[string]reflect.Value{
				"n": reflect.ValueOf(&n),
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
			{
				Keys: []string{"n"},
				Values: []any{
					[]any{
						neo4j.Node{
							Labels: []string{
								"Organism",
								"Human",
							},
							Props: map[string]any{
								"id":    "human",
								"alive": true,
							},
						},
					},
				},
			},
		}
		err := s.unmarshalRecords(cy, records)
		assert.NoError(t, err)
		assert.Equal(t, &tests.BasePet{
			BaseOrganism: tests.BaseOrganism{
				Node: internal.Node{
					ID: "pet",
				},
			},
			Cute: true,
		}, n[0][0])
		assert.Equal(t, &tests.Human{
			BaseOrganism: tests.BaseOrganism{
				Node: internal.Node{
					ID: "human",
				},
				Alive: true,
			},
		}, n[1][0])
	})

	t.Run("binds to [][]Concrete where Concrete is an implementation of Abstract", func(t *testing.T) {
		s := &session{}
		s.RegisterTypes(&tests.BaseOrganism{}, &tests.BasePet{})
		var n [][]tests.BasePet
		cy := &internal.CompiledCypher{
			Bindings: map[string]reflect.Value{
				"n": reflect.ValueOf(&n),
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
		assert.Equal(t, tests.BasePet{
			BaseOrganism: tests.BaseOrganism{
				Node: internal.Node{
					ID: "pet",
				},
			},
			Cute: true,
		}, n[0][0])
	})

	t.Run("unmarshalling slices", func(t *testing.T) {
		require := require.New(t)
		s := &session{
			driver: &driver{
				Registry: internal.NewRegistry(),
			},
		}

		type Person struct {
			ID int `json:"id"`
		}

		// UNWIND [1, 2, 3] AS id
		// WITH {id: id} AS person
		// WITH collect(person) AS persons
		var persons [][]*Person
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
			Bindings: map[string]reflect.Value{
				"persons": reflect.ValueOf(&persons),
			},
		}, record)
		require.NoError(err)

		// (*db.Record)(0xc0005a34a0)({
		//  Values: ([]interface {}) (len=1 cap=1) {
		//   ([]interface {}) (len=3 cap=3) {
		//    (map[string]interface {}) (len=1) {
		//     (string) (len=2) "id": (int64) 1
		//    },
		//    (map[string]interface {}) (len=1) {
		//     (string) (len=2) "id": (int64) 2
		//    },
		//    (map[string]interface {}) (len=1) {
		//     (string) (len=2) "id": (int64) 3
		//    }
		//   }
		//  },
		//  Keys: ([]string) (len=1 cap=1) {
		//   (string) (len=7) "persons"
		//  }
		// })

		require.NoError(err)
		require.Len(persons, 1)
	})
}

func TestStream(t *testing.T) {
	ctx := context.Background()

	t.Run("should fail when invalid parameters passed", func(t *testing.T) {
		d, m := newHybridDriver(t, ctx)
		m.Bind(nil)

		var nums []chan int
		err := d.Exec().
			Unwind(db.NamedParam(nums, "nums"), "i").
			Return(db.Qual(&nums, "i")).
			Stream(ctx, func(r builder.Result) error {
				return nil
			})
		assert.Error(t, err)
	})

	t.Run("should stream when valid query", func(t *testing.T) {
		records := make([]map[string]any, 11)
		for i := range records {
			records[i] = map[string]any{"i": i}
		}
		d, m := newHybridDriver(t, ctx)
		m.BindRecords(records)

		expectedOut := []int{0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10}
		var num int
		err := d.Exec().
			Unwind("range(0, 10)", "i").
			Return(db.Qual(&num, "i")).
			Stream(ctx, func(r builder.Result) error {
				n := 0
				for r.Next(ctx) {
					if err := r.Read(); err != nil {
						return err
					}
					assert.Equal(t, expectedOut[n], num)
					n++
				}
				assert.Equal(t, len(expectedOut), n)
				return nil
			})
		assert.NoError(t, err)
	})
}

func TestRun(t *testing.T) {
	ctx := context.Background()
	neo4jDriver, cancel := startNeo4J(ctx)
	d := New(neo4jDriver)
	t.Cleanup(func() {
		if err := cancel(ctx); err != nil {
			t.Fatal(err)
		}
	})

	t.Run("unmarshals slice of length 1", func(t *testing.T) {
		var is []int
		err := d.Exec().
			Unwind("range(1, 1)", "i").
			Return(db.Qual(&is, "i")).
			Run(ctx)
		assert.NoError(t, err)
		assert.Equal(t, []int{1}, is)
	})

	t.Run("non-existent nil property nil pointer", func(t *testing.T) {
		// Create a test node first
		err := d.Exec().
			Create(
				db.Node(
					db.Var(
						"t",
						db.Label("TestNode"),
					),
				),
			).
			Run(ctx)
		assert.NoError(t, err)

		// Try to query a non-existent property
		var listOfVal []string
		err = d.Exec().
			Cypher(`MATCH (t:TestNode)`).
			Return(db.Qual(&listOfVal, "t.someNonExistentProp")).
			Run(ctx)

		// Should not error, but return an empty list since the property doesn't exist
		assert.NoError(t, err)
		assert.Empty(t, listOfVal, "Expected empty list when querying non-existent property")
	})
}

func TestRunSummary(t *testing.T) {
	// TODO: Setup mocks
	if testing.Short() {
		return
	}
	ctx := context.Background()
	neo4jDriver, cancel := startNeo4J(ctx)
	d := New(neo4jDriver)
	t.Cleanup(func() {
		if err := cancel(ctx); err != nil {
			t.Fatal(err)
		}
	})

	t.Run("reports correct summary", func(t *testing.T) {
		var p Person
		p.ID = "Jessie"
		summary, err := d.Exec().
			Create(db.Node(&p)).
			Set(db.SetPropValue(&p.Name, &p.ID)).
			Return(&p).
			RunSummary(ctx)
		assert.NoError(t, err)
		assert.Equal(t, p.ID, p.Name)
		assert.Equal(t, 1, summary.Counters().NodesCreated())
	})
}

func TestResultImpl(t *testing.T) {
	// TODO: Setup mocks
	if testing.Short() {
		return
	}

	ctx := context.Background()
	neo4jDriver, cancel := startNeo4J(ctx)
	d := New(neo4jDriver)
	readSession := d.ReadSession(ctx)
	session := &session{session: readSession.Session()}

	t.Cleanup(func() {
		if err := readSession.Close(ctx); err != nil {
			t.Fatal(err)
		}
		if err := cancel(ctx); err != nil {
			t.Fatal(err)
		}
	})

	t.Run("Peek", func(t *testing.T) {
		var num int
		err := d.Exec().
			Unwind("range(0, 1)", "i").
			Return(db.Qual(&num, "i")).
			Stream(ctx, func(r builder.Result) error {
				assert.True(t, r.Next(ctx))
				assert.True(t, r.Peek(ctx), "should be true when there is one record to process after current record")
				assert.True(t, r.Next(ctx))
				assert.False(t, r.Peek(ctx), "should be false when there is no record to process after current record")
				return nil
			})
		assert.NoError(t, err)
	})

	t.Run("Next", func(t *testing.T) {
		var num int
		err := d.Exec().
			Unwind("range(0, 0)", "i").
			Return(db.Qual(&num, "i")).
			Stream(ctx, func(r builder.Result) error {
				assert.True(t, r.Next(ctx), "should be true when there is one record to process")
				assert.False(t, r.Next(ctx), "should be false when there is no record to process")
				return nil
			})
		assert.NoError(t, err)
	})

	t.Run("Err", func(t *testing.T) {
		t.Run("should not throw error for valid resultWithContext", func(t *testing.T) {
			var num int
			err := d.Exec().
				Unwind("range(0, 0)", "i").
				Return(db.Qual(&num, "i")).
				Stream(ctx, func(r builder.Result) error {
					return r.Err()
				})
			assert.NoError(t, err)
		})

		t.Run("should throw error when there is error in resultWithContext", func(t *testing.T) {
			var n []any
			c := internal.NewCypherClient(internal.NewRegistry())
			cy, err := c.
				Match(db.Node(db.Var(n, db.Name("n")))).
				Return(n).
				Compile()
			assert.NoError(t, err)
			params, err := canonicalizeParams(cy.Parameters)
			assert.NoError(t, err)

			r := runnerImpl{session: session}
			_, err = r.executeTransaction(ctx, cy, func(tx neo4j.ManagedTransaction) (any, error) {
				var result neo4j.ResultWithContext
				result, err = tx.Run(ctx, cy.Cypher, params)
				assert.NoError(t, err)
				_, resultErr := result.Single(ctx)
				assert.Error(t, resultErr)

				var res builder.Result = &resultImpl{
					ResultWithContext: result,
					compiled:          cy,
				}
				assert.ErrorIs(t, res.Err(), resultErr)
				return nil, res.Err()
			})
			assert.Error(t, err)
		})
	})

	t.Run("Read", func(t *testing.T) {
		t.Run("should read values for valid query", func(t *testing.T) {
			var num int
			err := d.Exec().Unwind("range(0, 5)", "i").
				Return(db.Qual(&num, "i")).
				Stream(ctx, func(r builder.Result) error {
					for i := 0; r.Next(ctx); i++ {
						err := r.Read()
						assert.NoError(t, err)
						assert.Equal(t, i, num)
					}
					return nil
				})
			assert.NoError(t, err)
		})

		t.Run("should fail read for invalid variable type", func(t *testing.T) {
			var num string
			err := d.Exec().Unwind("range(0, 5)", "i").
				Return(db.Qual(&num, "i")).
				Stream(ctx, func(r builder.Result) error {
					assert.True(t, r.Next(ctx))
					return r.Read()
				})
			assert.Error(t, err)
		})
	})
}

func TestClient(t *testing.T) {
	t.Run("all methods", func(t *testing.T) {
		// This is simply to test the clientImpl wrapper around CypherClient to
		// ensure no nil dereferences etc. Obviously syntax is not tested here.
		c := NewMock()
		c.Bind(nil)
		err := c.Exec().
			// All Client methods
			Subquery(func(c Query) builder.Runner {
				return c.Union(
					func(c Query) builder.Runner {
						return c.Return("n")
					},
					func(c Query) builder.Runner {
						return c.Use("graph").Return("n")
					},
				)
			}).
			Subquery(func(c Query) builder.Runner {
				return c.UnionAll(
					func(c Query) builder.Runner {
						return c.Call("aff")
					},
					func(c Query) builder.Runner {
						return c.Return("n")
					},
				)
			}).

			// All Querier methods
			Where(db.Cond("x", "=", "2")).

			// All Updater[Querier] methods
			Create(db.Node("n")).
			Merge(db.Node("m")).
			Delete().
			DetachDelete().
			Set().
			Remove().
			ForEach("a", "m", func(c builder.Updater[any]) {
				c.Set()
			}).

			// All Reader methods
			OptionalMatch(db.Node("p")).
			Match(db.Node("o")).
			With("n").
			Call("call").
			Yield("yield").
			Show("").
			Subquery(func(c Query) builder.Runner {
				return c.Match(db.Node("m"))
			}).
			Cypher("").
			Unwind("a", "a").
			Print().
			Run(context.Background())
		require.NoError(t, err)
	})
}
