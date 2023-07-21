package tests

import (
	"reflect"
	"testing"

	"github.com/rlch/neogo/db"
	"github.com/rlch/neogo/internal"
)

func TestOrderBy(t *testing.T) {
	t.Run("Order nodes by property", func(t *testing.T) {
		var n Person
		c := internal.NewCypherClient()
		cy, err := c.
			Match(
				db.Node(db.Bind("n", &n)),
			).
			Return(
				db.Return(db.Qual(&n.Name, "n.name"), db.OrderBy("", true)),
				db.Qual(&n.Age, "n.age"),
			).
			Compile()

		check(t, cy, err, internal.CompiledCypher{
			Cypher: `
					MATCH (n)
					RETURN n.name, n.age
					ORDER BY n.name
					`,
			Bindings: map[string]reflect.Value{
				"n.name": reflect.ValueOf(&n.Name),
				"n.age":  reflect.ValueOf(&n.Age),
			},
		})
	})

	t.Run("Order nodes by multiple properties", func(t *testing.T) {
		var n Person
		c := internal.NewCypherClient()
		cy, err := c.
			Match(db.Node(db.Bind("n", &n))).
			Return(
				db.Return(db.Qual(&n.Name, "n.name"), db.OrderBy("", true)),
				db.Return(db.Qual(&n.Age, "n.age"), db.OrderBy("", true)),
			).
			Compile()

		check(t, cy, err, internal.CompiledCypher{
			Cypher: `
					MATCH (n)
					RETURN n.name, n.age
					ORDER BY n.age, n.name
					`,
			Bindings: map[string]reflect.Value{
				"n.name": reflect.ValueOf(&n.Name),
				"n.age":  reflect.ValueOf(&n.Age),
			},
		})
	})

	t.Run("Order nodes by ID", func(t *testing.T) {
		var n Person
		c := internal.NewCypherClient()
		cy, err := c.
			Match(db.Node(db.Qual(&n, "n"))).
			Return(
				&n.Name,
				&n.Age,
				db.Return(nil, db.OrderBy("elementId(n)", true)),
			).
			Compile()

		check(t, cy, err, internal.CompiledCypher{
			Cypher: `
					MATCH (n:Person)
					RETURN n.name, n.age
					ORDER BY elementId(n)
					`,
			Bindings: map[string]reflect.Value{
				"n.name": reflect.ValueOf(&n.Name),
				"n.age":  reflect.ValueOf(&n.Age),
			},
		})
	})

	t.Run("Order nodes by expression", func(t *testing.T) {
		var n Person
		c := internal.NewCypherClient()
		cy, err := c.
			Match(db.Node(db.Qual(&n, "n"))).
			Return(
				&n.Name,
				&n.Age,
				db.Return(nil, db.OrderBy("elementId(n)", true)),
			).
			Compile()

		check(t, cy, err, internal.CompiledCypher{
			Cypher: `
					MATCH (n:Person)
					RETURN n.name, n.age
					ORDER BY elementId(n)
					`,
			Bindings: map[string]reflect.Value{
				"n.name": reflect.ValueOf(&n.Name),
				"n.age":  reflect.ValueOf(&n.Age),
			},
		})
	})

	t.Run("Order nodes by descending order", func(t *testing.T) {
		var n Person
		c := internal.NewCypherClient()
		cy, err := c.
			Match(db.Node(db.Qual(&n, "n"))).
			Return(
				db.Return(&n.Name, db.OrderBy("", false)),
				&n.Age,
			).
			Compile()

		check(t, cy, err, internal.CompiledCypher{
			Cypher: `
					MATCH (n:Person)
					RETURN n.name, n.age
					ORDER BY n.name DESC
					`,
			Bindings: map[string]reflect.Value{
				"n.name": reflect.ValueOf(&n.Name),
				"n.age":  reflect.ValueOf(&n.Age),
			},
		})
	})

	t.Run("Ordering null", func(t *testing.T) {
		var n Person
		c := internal.NewCypherClient()
		cy, err := c.
			Match(db.Node(db.Qual(&n, "n"))).
			Return(
				db.Return("n.length", db.OrderBy("", true)),
				&n.Name,
				&n.Age,
			).
			Compile()

		check(t, cy, err, internal.CompiledCypher{
			Cypher: `
					MATCH (n:Person)
					RETURN n.length, n.name, n.age
					ORDER BY n.length
					`,
			Bindings: map[string]reflect.Value{
				"n.name": reflect.ValueOf(&n.Name),
				"n.age":  reflect.ValueOf(&n.Age),
			},
		})
	})

	t.Run("Ordering in a WITH clause", func(t *testing.T) {
		var names []string
		c := internal.NewCypherClient()
		cy, err := c.
			Match(db.Node("n")).
			With(db.With("n", db.OrderBy("age", true))).
			Return(
				db.Qual(&names, "collect(n.name)", db.Name("names")),
			).
			Compile()

		check(t, cy, err, internal.CompiledCypher{
			Cypher: `
					MATCH (n)
					WITH n
					ORDER BY n.age
					RETURN collect(n.name) AS names
					`,
			Bindings: map[string]reflect.Value{
				"names": reflect.ValueOf(&names),
			},
		})
	})
}
