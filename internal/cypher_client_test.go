package internal

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCypherClient(t *testing.T) {
	t.Run("isWrite inference", func(t *testing.T) {
		t.Run("false when using non-write clauses", func(t *testing.T) {
			cy := newCypher()
			newCypherClient(cy).
				Match(&CypherPattern{
					ns: []*nodePattern{{data: "n"}},
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
			cy := newCypher()
			newCypherClient(cy).
				Create(&CypherPattern{
					ns: []*nodePattern{{data: "n"}},
				}).
				Return("n", "y")
			assert.Equal(t, true, cy.isWrite)
		})

		t.Run("true when using write clauses in subquery", func(t *testing.T) {
			cy := newCypher()
			newCypherClient(cy).
				Subquery(func(c *CypherClient) *CypherRunner {
					return c.Create(&CypherPattern{
						ns: []*nodePattern{{data: "n"}},
					}).CypherRunner
				}).
				Return("n")
			assert.Equal(t, true, cy.isWrite)
		})
	})
}
