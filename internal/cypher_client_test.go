package internal

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCypherClient(t *testing.T) {
	r := NewRegistry()
	t.Run("isWrite inference", func(t *testing.T) {
		t.Run("false when using non-write clauses", func(t *testing.T) {
			cy := newCypher(r)
			newCypherClient(cy).
				Match(&CypherPatterns{
					resolver: func(r *Registry) []*nodePatternPart {
						return []*nodePatternPart{{data: "n"}}
					},
				}).
				Where(&Condition{
					Key:   "n.age",
					Op:    "=",
					Value: "10",
				}).
				Unwind("[1,2,3]", "m").
				Return("n", "m")
			assert.Equal(t, false, cy.isWrite)
		})

		t.Run("true when using write clauses", func(t *testing.T) {
			cy := newCypher(r)
			newCypherClient(cy).
				Create(&CypherPatterns{
					resolver: func(r *Registry) []*nodePatternPart {
						return []*nodePatternPart{{data: "n"}}
					},
				}).
				Return("n", "y")
			assert.Equal(t, true, cy.isWrite)
		})

		t.Run("true when using Cypher if a write clause is used", func(t *testing.T) {
			cy := newCypher(r)
			newCypherClient(cy).
				Cypher("").
				Return("y")
			assert.Equal(t, false, cy.isWrite)

			newCypherClient(cy).
				Cypher("Merge (n)").
				Return("y")
			assert.Equal(t, true, cy.isWrite)
		})

		t.Run("true when using write clauses in subquery", func(t *testing.T) {
			cy := newCypher(r)
			newCypherClient(cy).
				Subquery(func(c *CypherClient) *CypherRunner {
					return c.Create(&CypherPatterns{
						resolver: func(r *Registry) []*nodePatternPart {
							return []*nodePatternPart{{data: "n"}}
						},
					}).CypherRunner
				}).
				Return("n")
			assert.Equal(t, true, cy.isWrite)
		})

		t.Run("true when using Cypher in subquery if a write clause is used", func(t *testing.T) {
			cy := newCypher(r)
			newCypherClient(cy).
				Subquery(func(c *CypherClient) *CypherRunner {
					return c.Cypher("").CypherRunner
				}).
				Return("n")
			assert.Equal(t, false, cy.isWrite)

			newCypherClient(cy).
				Subquery(func(c *CypherClient) *CypherRunner {
					return c.Cypher("Create (n)").CypherRunner
				}).
				Return("n")
			assert.Equal(t, true, cy.isWrite)
		})
	})
}
