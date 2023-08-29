package tests

import (
	"reflect"
	"testing"

	"github.com/rlch/neogo/db"
	"github.com/rlch/neogo/internal"
)

func TestRemove(t *testing.T) {
	t.Run("Set a property", func(t *testing.T) {
		var a Person
		c := internal.NewCypherClient()
		cy, err := c.
			Match(db.Node(db.Qual(&a, "a", db.Props{"name": "'Andy'"}))).
			Remove(db.RemoveProp(&a.Age)).
			Return(&a.Name, &a.Age).
			Compile()

		Check(t, cy, err, internal.CompiledCypher{
			Cypher: `
					MATCH (a:Person {name: 'Andy'})
					REMOVE a.age
					RETURN a.name, a.age
					`,
			Bindings: map[string]reflect.Value{
				"a.name": reflect.ValueOf(&a.Name),
				"a.age":  reflect.ValueOf(&a.Age),
			},
		})
	})

	t.Run("Remove all properties", func(t *testing.T) {
		var n Person
		var labels []string
		c := internal.NewCypherClient()
		cy, err := c.
			Match(db.Node(db.Qual(&n, "n", db.Props{"name": "'Peter'"}))).
			Remove(db.RemoveLabels(&n, "German")).
			Return(&n.Name, db.Qual(&labels, "labels(n)")).
			Compile()

		Check(t, cy, err, internal.CompiledCypher{
			Cypher: `
					MATCH (n:Person {name: 'Peter'})
					REMOVE n:German
					RETURN n.name, labels(n)
					`,
			Bindings: map[string]reflect.Value{
				"n.name":    reflect.ValueOf(&n.Name),
				"labels(n)": reflect.ValueOf(&labels),
			},
		})
	})

	t.Run("Remove a label from a node", func(t *testing.T) {
		var n Person
		var labels []string
		c := internal.NewCypherClient()
		cy, err := c.
			Match(db.Node(db.Qual(&n, "n", db.Props{"name": "'Peter'"}))).
			Remove(db.RemoveLabels(&n, "German")).
			Return(&n.Name, db.Qual(&labels, "labels(n)")).
			Compile()

		Check(t, cy, err, internal.CompiledCypher{
			Cypher: `
					MATCH (n:Person {name: 'Peter'})
					REMOVE n:German
					RETURN n.name, labels(n)
					`,
			Bindings: map[string]reflect.Value{
				"n.name":    reflect.ValueOf(&n.Name),
				"labels(n)": reflect.ValueOf(&labels),
			},
		})
	})

	t.Run("Remove multiple labels from a node", func(t *testing.T) {
		var n Person
		var labels []string
		c := internal.NewCypherClient()
		cy, err := c.
			Match(db.Node(db.Qual(&n, "n", db.Props{"name": "'Peter'"}))).
			Remove(db.RemoveLabels(&n, "German", "Swedish")).
			Return(&n.Name, db.Qual(&labels, "labels(n)")).
			Compile()

		Check(t, cy, err, internal.CompiledCypher{
			Cypher: `
					MATCH (n:Person {name: 'Peter'})
					REMOVE n:German:Swedish
					RETURN n.name, labels(n)
					`,
			Bindings: map[string]reflect.Value{
				"n.name":    reflect.ValueOf(&n.Name),
				"labels(n)": reflect.ValueOf(&labels),
			},
		})
	})
}
