package tests

import (
	"reflect"
	"testing"

	"github.com/rlch/neogo/db"
	"github.com/rlch/neogo/internal"
)

func TestCypher(t *testing.T) {
	c := internal.NewCypherClient()
	var n any
	cy, err := c.
		Match(db.Node(db.Qual(&n, "n"))).
		Cypher(`WHERE n.name = 'Bob'`).
		Return(&n).
		Compile()

	Check(t, cy, err, internal.CompiledCypher{
		Cypher: `
					MATCH (n)
					WHERE n.name = 'Bob'
					RETURN n
					`,
		Bindings: map[string]reflect.Value{
			"n": reflect.ValueOf(&n),
		},
	})
}
