package tests

import (
	"reflect"
	"testing"

	neo4jgorm "github.com/rlch/neo4j-gorm"
	"github.com/rlch/neo4j-gorm/db"
	"github.com/rlch/neo4j-gorm/internal"
)

func TestUnwind(t *testing.T) {
	t.Run("Unwinding a list", func(t *testing.T) {
		var (
			x []any
			y []string
		)
		c := internal.NewCypherClient()
		cy, err := c.
			Unwind(db.Expr("[1, 2, 3, null]"), "x").
			Find(db.Bind("x", &x), db.Qual(db.Bind(db.Expr("'val'"), &y), "y")).Compile()

		check(t, cy, err, internal.CompiledCypher{
			Cypher: `
					UNWIND [1, 2, 3, null] AS x
					RETURN x, 'val' AS y
					`,
			Bindings: map[string]reflect.Value{
				"x": reflect.ValueOf(&x),
				"y": reflect.ValueOf(&y),
			},
		})
	})

	t.Run("Creating a distinct list", func(t *testing.T) {
		var setOfVals []any
		c := internal.NewCypherClient()
		cy, err := c.
			With(db.Qual("[1, 1, 2, 2]", "coll")).
			Unwind("coll", "x").
			With(db.With("x", db.Distinct)).
			Find(db.Qual(&setOfVals, "collect(x)", db.Name("setOfVals"))).Compile()

		check(t, cy, err, internal.CompiledCypher{
			Cypher: `
					WITH [1, 1, 2, 2] AS coll
					UNWIND coll AS x
					WITH DISTINCT x
					RETURN collect(x) AS setOfVals
					`,
			Bindings: map[string]reflect.Value{
				"setOfVals": reflect.ValueOf(&setOfVals),
			},
		})
	})

	t.Run("Using UNWIND with any expression returning a list", func(t *testing.T) {
		var x []float64
		c := internal.NewCypherClient()
		cy, err := c.
			With(
				db.Qual("[1, 2]", "a"),
				db.Qual("[3, 4]", "b"),
			).
			Unwind("(a + b)", "x").
			Find(db.Qual(&x, "x")).Compile()

		check(t, cy, err, internal.CompiledCypher{
			Cypher: `
					WITH [1, 2] AS a, [3, 4] AS b
					UNWIND (a + b) AS x
					RETURN x
					`,
			Bindings: map[string]reflect.Value{
				"x": reflect.ValueOf(&x),
			},
		})
	})

	t.Run("Using UNWIND with a list of lists", func(t *testing.T) {
		var y []float64
		c := internal.NewCypherClient()
		cy, err := c.
			With(db.Qual("[[1, 2], [3, 4], 5]", "nested")).
			Unwind("nested", "x").
			Unwind("x", "y").
			Find(db.Qual(&y, "y")).Compile()

		check(t, cy, err, internal.CompiledCypher{
			Cypher: `
					WITH [[1, 2], [3, 4], 5] AS nested
					UNWIND nested AS x
					UNWIND x AS y
					RETURN y
					`,
			Bindings: map[string]reflect.Value{
				"y": reflect.ValueOf(&y),
			},
		})
	})

	t.Run("Using UNWIND with an empty list", func(t *testing.T) {
		c := internal.NewCypherClient()
		cy, err := c.
			Unwind("[]", "empty").
			Find("'literal_that_is_not_returned'").Compile()

		check(t, cy, err, internal.CompiledCypher{
			Cypher: `
					UNWIND [] AS empty
					RETURN 'literal_that_is_not_returned'
					`,
			Bindings: map[string]reflect.Value{},
		})
	})

	t.Run("Using UNWIND with an expression that is not a list", func(t *testing.T) {
		c := internal.NewCypherClient()
		cy, err := c.
			Unwind("null", "x").
			Find("x", "'some_literal'").Compile()

		check(t, cy, err, internal.CompiledCypher{
			Cypher: `
					UNWIND null AS x
					RETURN x, 'some_literal'
					`,
			Bindings: map[string]reflect.Value{},
		})
	})

	t.Run("Creating nodes from a list parameter", func(t *testing.T) {
		events := map[string]any{
			"events": []map[string]any{
				{
					"year": 2014,
					"id":   1,
				},
				{
					"year": 2014,
					"id":   2,
				},
			},
		}
		type Year struct {
			neo4jgorm.Node `neo4j:"Year"`

			Year int `json:"year"`
		}
		type Event struct {
			neo4jgorm.Node `neo4j:"Event"`

			ID   int `json:"id"`
			Year int `json:"year"`
		}
		type In struct {
			neo4jgorm.Relationship `neo4j:"IN"`
		}
		var (
			y Year
			e Event
		)
		c := internal.NewCypherClient()
		cy, err := c.
			Unwind(db.Qual(&events, "events"), "event").
			Merge(
				c.Node(db.Qual(&y, "y", db.Props{"year": "event.year"})),
			).
			Merge(
				c.Node(&y).
					From(In{}, db.Qual(&e, "e", db.Props{"id": "event.id"})),
			).
			Find(db.Return(db.Qual(&e.ID, "x"), db.OrderBy("", true))).
			Compile()

		check(t, cy, err, internal.CompiledCypher{
			Cypher: `
					UNWIND $events AS event
					MERGE (y:Year {year: event.year})
					MERGE (y)<-[:IN]-(e:Event {id: event.id})
					RETURN e.id AS x
					ORDER BY x
					`,
			Bindings: map[string]reflect.Value{
				"x": reflect.ValueOf(&e.ID),
			},
			Parameters: map[string]any{
				"events": &events,
			},
		})
	})
}
