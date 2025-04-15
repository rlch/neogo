package tests

import (
	"testing"

	"github.com/rlch/neogo/db"
	"github.com/rlch/neogo/internal"
)

func TestDelete(t *testing.T) {
	t.Run("Delete single node", func(t *testing.T) {
		var n Person
		c := internal.NewCypherClient(r)
		cy, err := c.
			Match(db.Node(db.Qual(&n, "n", db.Props{"name": "'Tom Hanks'"}))).
			Delete(&n).
			Compile()

		Check(t, cy, err, internal.CompiledCypher{
			Cypher: `
					MATCH (n:Person {name: 'Tom Hanks'})
					DELETE n
					`,
		})
	})

	t.Run("Delete relationships only", func(t *testing.T) {
		var (
			n       Person
			actedIn ActedIn
		)
		c := internal.NewCypherClient(r)
		cy, err := c.
			Match(
				db.Node(db.Qual(&n, "n", db.Props{"name": "'Laurence Fishburne'"})).
					To(db.Qual(&actedIn, "r"), nil),
			).
			Delete(&actedIn).
			Compile()

		Check(t, cy, err, internal.CompiledCypher{
			Cypher: `
					MATCH (n:Person {name: 'Laurence Fishburne'})-[r:ACTED_IN]->()
					DELETE r
					`,
		})
	})

	t.Run("Delete a node with all its relationships", func(t *testing.T) {
		var n Person
		c := internal.NewCypherClient(r)
		cy, err := c.
			Match(
				db.Node(
					db.Qual(&n, "n",
						db.Props{"name": "'Carrie-Anne Moss'"},
					),
				),
			).
			DetachDelete(&n).
			Compile()

		Check(t, cy, err, internal.CompiledCypher{
			Cypher: `
					MATCH (n:Person {name: 'Carrie-Anne Moss'})
					DETACH DELETE n
					`,
		})
	})

	t.Run("Delete all nodes and relationships", func(t *testing.T) {
		var n any
		c := internal.NewCypherClient(r)
		cy, err := c.
			Match(db.Node(db.Qual(&n, "n"))).
			DetachDelete(&n).
			Compile()

		Check(t, cy, err, internal.CompiledCypher{
			Cypher: `
					MATCH (n)
					DETACH DELETE n
					`,
		})
	})
}
