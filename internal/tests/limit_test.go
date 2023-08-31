package tests

import (
	"reflect"
	"testing"

	"github.com/rlch/neogo/db"
	"github.com/rlch/neogo/internal"
)

func TestLimit(t *testing.T) {
	t.Run("Return a limited subset of the rows", func(t *testing.T) {
		var name string
		c := internal.NewCypherClient()
		cy, err := c.
			Match(db.Node("n")).
			Return(
				db.Return(db.Qual(&name, "n.name"), db.OrderBy("", true), db.Limit("3")),
			).
			Compile()

		Check(t, cy, err, internal.CompiledCypher{
			Cypher: `
					MATCH (n)
					RETURN n.name
					ORDER BY n.name
					LIMIT 3
					`,
			Bindings: map[string]reflect.Value{
				"n.name": reflect.ValueOf(&name),
			},
		})
	})

	t.Run("Using an expression with LIMIT to return a subset of the rows", func(t *testing.T) {
		var name string
		c := internal.NewCypherClient()
		cy, err := c.
			Match(db.Node("n")).
			Return(
				db.Return(
					db.Qual(&name, "n.name"),
					db.OrderBy("", true),
					db.Limit("1 + toInteger(3 * rand())"),
				),
			).
			Compile()

		Check(t, cy, err, internal.CompiledCypher{
			Cypher: `
					MATCH (n)
					RETURN n.name
					ORDER BY n.name
					LIMIT 1 + toInteger(3 * rand())
					`,
			Bindings: map[string]reflect.Value{
				"n.name": reflect.ValueOf(&name),
			},
		})
	})

	t.Run("Using an expression with SKIP to return a subset of the rows", func(t *testing.T) {
		var name string
		c := internal.NewCypherClient()
		cy, err := c.
			Match(db.Node("n")).
			Return(
				db.Return(db.Qual(&name, "n.name"), db.OrderBy("", true), db.Limit("3")),
			).
			Compile()

		Check(t, cy, err, internal.CompiledCypher{
			Cypher: `
					MATCH (n)
					RETURN n.name
					ORDER BY n.name
					LIMIT 3
					`,
			Bindings: map[string]reflect.Value{
				"n.name": reflect.ValueOf(&name),
			},
		})
	})
}
