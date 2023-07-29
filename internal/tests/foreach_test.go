package tests

import (
	"testing"

	"github.com/rlch/neogo/db"
	"github.com/rlch/neogo/internal"
)

// TODO: probs needs more tests lol
func TestForEach(t *testing.T) {
	t.Run("Return a limited subset of the rows", func(t *testing.T) {
		c := internal.NewCypherClient()
		cy, err := c.
			Match(
				db.Path(db.Node("start").To(db.Var(nil, db.Quantifier("*")), "finish"), "p"),
			).
			Where(db.And(
				db.Cond("start.name", "=", "'A'"),
				db.Cond("finish.name", "=", "'D'"),
			)).
			ForEach("n", "nodes(p)", func(c *internal.CypherUpdater[any]) {
				c.Set(db.SetPropValue("n.marked", true))
			}).
			Compile()

		check(t, cy, err, internal.CompiledCypher{
			Cypher: `
					MATCH p = (start)-[*]->(finish)
					WHERE start.name = 'A' AND finish.name = 'D'
					FOREACH (n IN nodes(p) | SET n.marked = true)
					`,
		})
	})
}
