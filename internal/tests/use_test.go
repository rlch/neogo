package tests

import (
	"reflect"
	"testing"

	"github.com/rlch/neogo/db"
	"github.com/rlch/neogo/internal"
)

func TestUse(t *testing.T) {
	t.Run("Query a graph", func(t *testing.T) {
		c := internal.NewCypherClient()
		var n any
		cy, err := c.
			Use("myDatabase").
			Match(db.Node(db.Qual(&n, "n"))).
			Return("n").
			Compile()

		Check(t, cy, err, internal.CompiledCypher{
			Cypher: `
					USE myDatabase
					MATCH (n)
					RETURN n
					`,
			Bindings: map[string]reflect.Value{
				"n": reflect.ValueOf(&n),
			},
		})
	})

	t.Run("Query a composite db constituent graph", func(t *testing.T) {
		c := internal.NewCypherClient()
		var n any
		cy, err := c.
			Use("myComposite.myConstituent").
			Match(db.Node(db.Qual(&n, "n"))).
			Return("n").
			Compile()

		Check(t, cy, err, internal.CompiledCypher{
			Cypher: `
					USE myComposite.myConstituent
					MATCH (n)
					RETURN n
					`,
			Bindings: map[string]reflect.Value{
				"n": reflect.ValueOf(&n),
			},
		})
	})

	t.Run("Query a composite db constituent graph dynamically", func(t *testing.T) {
		c := internal.NewCypherClient()
		var n any
		cy, err := c.
			Use("graph.byName($graphName)").
			Match(db.Node(db.Qual(&n, "n"))).
			Return("n").
			CompileWithParams(map[string]any{
				"graphName": "'idksomegraph'",
			})

		Check(t, cy, err, internal.CompiledCypher{
			Cypher: `
					USE graph.byName($graphName)
					MATCH (n)
					RETURN n
					`,
			Bindings: map[string]reflect.Value{
				"n": reflect.ValueOf(&n),
			},
			Parameters: map[string]any{
				"graphName": "'idksomegraph'",
			},
		})
	})
}
