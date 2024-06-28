package neogo

import (
	"context"
	"reflect"
	"testing"

	"github.com/neo4j/neo4j-go-driver/v5/neo4j"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/rlch/neogo/db"
	"github.com/rlch/neogo/internal"
	"github.com/rlch/neogo/internal/tests"
	"github.com/rlch/neogo/query"
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
			s := &session{
				registry: registry{
					abstractNodes: []IAbstract{
						&tests.BaseOrganism{},
						&tests.BasePet{},
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
			Stream(ctx, func(r query.Result) error {
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
			Stream(ctx, func(r query.Result) error {
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
	var d Driver
	if testing.Short() {
		m := NewMock()
		m.BindRecords([]map[string]any{{"i": 1}})
		d = m
	} else {
		neo4jDriver, cancel := startNeo4J(ctx)
		d = New(neo4jDriver)
		t.Cleanup(func() {
			if err := cancel(ctx); err != nil {
				t.Fatal(err)
			}
		})
	}

	t.Run("unmarshals slice of length 1", func(t *testing.T) {
		var is []int
		err := d.Exec().
			Unwind("range(1, 1)", "i").
			Return(db.Qual(&is, "i")).Run(ctx)
		assert.NoError(t, err)
		assert.Equal(t, []int{1}, is)
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
			Stream(ctx, func(r query.Result) error {
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
			Stream(ctx, func(r query.Result) error {
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
				Stream(ctx, func(r query.Result) error {
					return r.Err()
				})
			assert.NoError(t, err)
		})

		t.Run("should throw error when there is error in resultWithContext", func(t *testing.T) {
			var n []any
			c := internal.NewCypherClient()
			cy, err := c.
				Match(db.Node(db.Var(n, db.Name("n")))).
				Return(n).
				Compile()
			assert.NoError(t, err)
			params, err := canonicalizeParams(cy.Parameters)
			assert.NoError(t, err)

			r := runnerImpl{session: session}
			err = r.executeTransaction(ctx, cy, func(tx neo4j.ManagedTransaction) (any, error) {
				var result neo4j.ResultWithContext
				result, err = tx.Run(ctx, cy.Cypher, params)
				assert.NoError(t, err)
				_, resultErr := result.Single(ctx)
				assert.Error(t, resultErr)

				var res query.Result = &resultImpl{
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
				Stream(ctx, func(r query.Result) error {
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
				Stream(ctx, func(r query.Result) error {
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
			Subquery(func(c Query) query.Runner {
				return c.Union(
					func(c Query) query.Runner {
						return c.Return("n")
					},
					func(c Query) query.Runner {
						return c.Use("graph").Return("n")
					},
				)
			}).
			Subquery(func(c Query) query.Runner {
				return c.UnionAll(
					func(c Query) query.Runner {
						return c.Call("aff")
					},
					func(c Query) query.Runner {
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
			ForEach("a", "m", func(c query.Updater[any]) {
				c.Set()
			}).

			// All Reader methods
			OptionalMatch(db.Node("p")).
			Match(db.Node("o")).
			With("n").
			Call("call").
			Yield("yield").
			Show("").
			Subquery(func(c Query) query.Runner {
				return c.Match(db.Node("m"))
			}).
			Cypher("").
			Unwind("a", "a").
			Print().
			Run(context.Background())
		require.NoError(t, err)
	})
}
