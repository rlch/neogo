package internal_test

import (
	"reflect"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/rlch/neogo/db"
	"github.com/rlch/neogo/internal"
)

func TestParameter(t *testing.T) {
	t.Run("Doesn't allow non nil parameters, expression and aliasing at same time", func(t *testing.T) {
		propsList := []map[string]any{
			{"id": "n0", "name": "Alice"},
			{"id": "n1", "name": "Bob"},
		}
		props := map[string]any{
			"id": "n2", "name": "Charlie",
		}

		c := internal.NewCypherClient()
		cy, err := c.
			With(db.NamedParam(&propsList, "propsList")).
			Unwind("range(0, size($propsList)-1)", "i").
			With("i", db.Qual(&props, "$propsList[i]", db.Name("props"))).
			Return(&props).Compile()
		assert.Error(t, err)
		assert.Nil(t, cy)
	})

	t.Run("Allow maps to be bound to an expression", func(t *testing.T) {
		var props map[string]any
		propsList := []map[string]any{
			{"id": "n0", "name": "Alice"},
			{"id": "n1", "name": "Bob"},
		}

		c := internal.NewCypherClient()
		cy, err := c.
			With(db.NamedParam(&propsList, "propsList")).
			Unwind("range(0, size($propsList)-1)", "i").
			With("i", db.Qual(&props, "$propsList[i]", db.Name("props"))).
			Return(&props).Compile()
		assert.NoError(t, err)

		wantCypher := internal.CompiledCypher{
			Cypher: `
          WITH $propsList
					UNWIND range(0, size($propsList)-1) AS i
					WITH i, $propsList[i] AS props
					RETURN props
					`,
			Parameters: map[string]any{
				"propsList": &propsList,
			},
			Bindings: map[string]reflect.Value{
				"props": reflect.ValueOf(&props),
			},
		}
		cypher := strings.TrimSpace(wantCypher.Cypher)
		cypher = strings.ReplaceAll(cypher, "\t", "")
		assert.Equal(t, cypher, cy.Cypher)
		assert.Equal(t, wantCypher.Bindings, cy.Bindings)
		assert.Equal(t, wantCypher.Parameters, cy.Parameters)
	})
}
