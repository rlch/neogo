package tests

import (
	"reflect"
	"testing"

	"github.com/rlch/neogo/db"
	"github.com/rlch/neogo/internal"
	"github.com/stretchr/testify/assert"
)

func TestParameter(t *testing.T) {
	t.Run("Doesn't allow non nil parameters, expression and aliasing at same time", func(t *testing.T) {
		t.Skip("TODO: This should be possible. Need to refactor register logic a bit.")
		// propsList := []map[string]any{
		// 	{"id": "n0", "name": "Alice"},
		// 	{"id": "n1", "name": "Bob"},
		// }
		// props := map[string]any{
		// 	"id": "n2", "name": "Charlie",
		// }
		//
		// c := internal.NewCypherClient()
		// cy, err := c.
		// 	With(db.NamedParam(&propsList, "propsList")).
		// 	Unwind("range(0, size($propsList)-1)", "i").
		// 	With("i", db.Qual(&props, "$propsList[i]", db.Name("props"))).
		// 	Return(&props).DebugPrint().Compile()
		// assert.Nil(t, cy)
		// assert.Error(t, err)
		// assert.ErrorIs(t, err, internal.ErrAliasAlreadyBound)
	})

	t.Run("Allow maps to be bound to an expression", func(t *testing.T) {
		var props map[string]any
		propsList := []map[string]any{
			{"id": "n0", "name": "Alice"},
			{"id": "n1", "name": "Bob"},
		}

		c := internal.NewCypherClient(r)
		cy, err := c.
			With(db.NamedParam(&propsList, "propsList")).
			Unwind("range(0, size($propsList)-1)", "i").
			With("i", db.Qual(&props, "$propsList[i]", db.Name("props"))).
			Return(&props).Compile()
		assert.NoError(t, err)

		Check(t, cy, err, internal.CompiledCypher{
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
		})
	})

	t.Run("Doesn't allow bounding a parameter to a value which is already bound to another value", func(t *testing.T) {
		var (
			num1 int
			num2 float64
		)
		numParam := []int{1, 2, 3}

		c := internal.NewCypherClient(r)
		cy, err := c.
			With(db.NamedParam(&numParam, "numbers")).
			With(db.Qual(&num1, "numbers[0]")).
			With(db.Qual(&num2, "numbers[0]")).
			Return(&num1, &num2).Compile()
		assert.Error(t, err)
		assert.ErrorIs(t, err, internal.ErrExpressionAlreadyBound)
		assert.Nil(t, cy)
	})
}
