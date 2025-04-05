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

func TestExprInPropertyIdentifier(t *testing.T) {
	c := internal.NewCypherClient()
	cy, err := c.Unwind(
		db.Expr("[?, ?, ?, ?]", "a", 2, []int{1}, nil),
		"x",
	).Compile()

	Check(t, cy, err, internal.CompiledCypher{
		Cypher: `
					UNWIND [$v1, $v2, $v3, null] AS x
					`,
		Parameters: map[string]any{
			"v1": "a",
			"v2": 2,
			"v3": []int{1},
		},
	})
}
