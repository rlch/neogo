package tests

import (
	"fmt"
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
		Cypher(func(scope *internal.Scope) string {
			return fmt.Sprintf("WHERE %s.name = 'Bob'", scope.Name(&n))
		}).
		Return(&n).
		Compile()

	check(t, cy, err, internal.CompiledCypher{
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
