package tests

import (
	"testing"

	"github.com/rlch/neogo/db"
	"github.com/rlch/neogo/internal"
)

// TestZeroSizeEmbedCollision tests the scenario where a relationship struct
// with a zero-size embedded type has both the struct and its first field
// referenced in the same query.
//
// The issue: Relationship{} is zero-size, so ActedIn's memory layout is:
//   &r == &r.Relationship == &r.Role (all same address)
//
// Without proper disambiguation, lookups would confuse the struct pointer
// with the field pointer.
func TestZeroSizeEmbedCollision(t *testing.T) {
	t.Run("Return both struct and field of relationship with zero-size embed", func(t *testing.T) {
		c := internal.NewCypherClient(r)
		var rel ActedIn // ActedIn embeds Relationship{} (zero-size)
		cy, err := c.
			Match(
				db.Node(db.Var(Person{}, db.Props{"name": "'Charlie'"})).
					To(db.Qual(&rel, "r"), db.Var(Movie{})),
			).
			Return(&rel, &rel.Role). // Both struct AND field!
			Compile()

		// Expected: RETURN r, r.role
		// If disambiguation fails: RETURN r, r (field lookup returns struct name)
		Check(t, cy, err, internal.CompiledCypher{
			Cypher: `
MATCH (:Person {name: 'Charlie'})-[r:ACTED_IN]->(:Movie)
RETURN r, r.role
`,
			Bindings: map[string]any{
				"r":      &rel,
				"r.role": &rel.Role,
			},
		})
	})

	t.Run("Use struct and field in WHERE condition", func(t *testing.T) {
		c := internal.NewCypherClient(r)
		var rel ActedIn
		cy, err := c.
			Match(
				db.Node(db.Var(Person{})).
					To(db.Qual(&rel, "r"), db.Var(Movie{})),
			).
			Where(db.Cond(&rel.Role, "=", "'Bud Fox'")). // Field in WHERE
			Return(&rel).                                // Struct in RETURN
			Compile()

		Check(t, cy, err, internal.CompiledCypher{
			Cypher: `
MATCH (:Person)-[r:ACTED_IN]->(:Movie)
WHERE r.role = 'Bud Fox'
RETURN r
`,
			Bindings: map[string]any{
				"r": &rel,
			},
		})
	})

	t.Run("ORDER BY field while returning struct", func(t *testing.T) {
		c := internal.NewCypherClient(r)
		var rel Knows // Knows has Since int field
		cy, err := c.
			Match(
				db.Node(db.Qual(Person{}, "a")).
					To(db.Qual(&rel, "r"), db.Qual(Person{}, "b")),
			).
			Return(
				db.Return(&rel, db.OrderBy(&rel.Since, false)), // Order by field, return struct
			).
			Compile()

		Check(t, cy, err, internal.CompiledCypher{
			Cypher: `
MATCH (a:Person)-[r:KNOWS]->(b:Person)
RETURN r
ORDER BY r.since DESC
`,
			Bindings: map[string]any{
				"r": &rel,
			},
		})
	})
}
