package tests

import (
	"reflect"
	"testing"

	"github.com/rlch/neogo/db"
	"github.com/rlch/neogo/internal"
)

func TestSkip(t *testing.T) {
	t.Run("Skip first three rows", func(t *testing.T) {
		var n Person
		c := internal.NewCypherClient()
		cy, err := c.
			Match(db.Node(db.Qual(&n, "n"))).
			Return(
				db.Return(&n.Name, db.OrderBy("", true), db.Skip("3")),
			).
			Compile()

		Check(t, cy, err, internal.CompiledCypher{
			Cypher: `
					MATCH (n:Person)
					RETURN n.name
					ORDER BY n.name
					SKIP 3
					`,
			Bindings: map[string]reflect.Value{
				"n.name": reflect.ValueOf(&n.Name),
			},
		})
	})

	t.Run("Return middle two rows", func(t *testing.T) {
		var n Person
		c := internal.NewCypherClient()
		cy, err := c.
			Match(db.Node(db.Qual(&n, "n"))).
			Return(
				db.Return(&n.Name, db.OrderBy("", true), db.Skip("1"), db.Limit("2")),
			).
			Compile()

		Check(t, cy, err, internal.CompiledCypher{
			Cypher: `
					MATCH (n:Person)
					RETURN n.name
					ORDER BY n.name
					SKIP 1
					LIMIT 2
					`,
			Bindings: map[string]reflect.Value{
				"n.name": reflect.ValueOf(&n.Name),
			},
		})
	})

	t.Run("Using an expression with SKIP to return a subset of the rows", func(t *testing.T) {
		var n Person
		c := internal.NewCypherClient()
		cy, err := c.
			Match(db.Node(db.Qual(&n, "n"))).
			Return(
				db.Return(&n.Name, db.OrderBy("", true), db.Skip("1 + toInteger(3*rand())")),
			).
			Compile()

		Check(t, cy, err, internal.CompiledCypher{
			Cypher: `
					MATCH (n:Person)
					RETURN n.name
					ORDER BY n.name
					SKIP 1 + toInteger(3*rand())
					`,
			Bindings: map[string]reflect.Value{
				"n.name": reflect.ValueOf(&n.Name),
			},
		})
	})
}
