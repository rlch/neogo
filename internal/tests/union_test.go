package tests

import (
	"reflect"
	"testing"

	"github.com/rlch/neogo/db"
	"github.com/rlch/neogo/internal"
)

func TestUnion(t *testing.T) {
	t.Run("Combine two queries and retain duplicates", func(t *testing.T) {
		c := internal.NewCypherClient(r)
		var name string
		cy, err := c.UnionAll(
			func(c *internal.CypherClient) *internal.CypherRunner {
				return c.
					Match(db.Node(db.Var("n", db.Label("Person")))).
					Return(db.Qual(&name, "n.name", db.Name("name")))
			},
			func(c *internal.CypherClient) *internal.CypherRunner {
				return c.
					Match(db.Node(db.Var("n", db.Label("Movie")))).
					Return(db.Qual(&name, "n.title", db.Name("name")))
			},
		).Compile()

		Check(t, cy, err, internal.CompiledCypher{
			Cypher: `
					MATCH (n:Person)
					RETURN n.name AS name
					UNION ALL
					MATCH (n:Movie)
					RETURN n.title AS name
					`,
			Bindings: map[string]reflect.Value{
				"name": reflect.ValueOf(&name),
			},
		})
	})

	t.Run("Combine two queries and remove duplicates", func(t *testing.T) {
		c := internal.NewCypherClient(r)
		var name string
		cy, err := c.Union(
			func(c *internal.CypherClient) *internal.CypherRunner {
				return c.
					Match(db.Node(db.Var("n", db.Label("Person")))).
					Return(db.Qual(&name, "n.name", db.Name("name")))
			},
			func(c *internal.CypherClient) *internal.CypherRunner {
				return c.
					Match(db.Node(db.Var("n", db.Label("Movie")))).
					Return(db.Qual(&name, "n.title", db.Name("name")))
			},
		).Compile()

		Check(t, cy, err, internal.CompiledCypher{
			Cypher: `
					MATCH (n:Person)
					RETURN n.name AS name
					UNION
					MATCH (n:Movie)
					RETURN n.title AS name
					`,
			Bindings: map[string]reflect.Value{
				"name": reflect.ValueOf(&name),
			},
		})
	})
}
