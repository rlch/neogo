package tests

import (
	"reflect"
	"testing"

	"github.com/rlch/neogo/db"
	"github.com/rlch/neogo/internal"
)

func TestSet(t *testing.T) {
	t.Run("Set a property", func(t *testing.T) {
		var n Person
		c := internal.NewCypherClient()
		cy, err := c.
			Match(db.Node(db.Qual(&n, "n", db.Props{"name": "'Andy'"}))).
			Set(db.SetPropValue(&n.Surname, "'Taylor'")).
			Return(&n.Name, &n.Surname).
			Compile()

		check(t, cy, err, internal.CompiledCypher{
			Cypher: `
					MATCH (n:Person {name: 'Andy'})
					SET n.surname = 'Taylor'
					RETURN n.name, n.surname
					`,
			Bindings: map[string]reflect.Value{
				"n.name":    reflect.ValueOf(&n.Name),
				"n.surname": reflect.ValueOf(&n.Surname),
			},
		})
	})

	t.Run("Update a property", func(t *testing.T) {
		var n Person
		c := internal.NewCypherClient()
		cy, err := c.
			Match(db.Node(db.Qual(&n, "n", db.Props{"name": "'Andy'"}))).
			Set(db.SetPropValue(&n.Age, "toString(n.age)")).
			Return(&n.Name, &n.Age).
			Compile()

		check(t, cy, err, internal.CompiledCypher{
			Cypher: `
					MATCH (n:Person {name: 'Andy'})
					SET n.age = toString(n.age)
					RETURN n.name, n.age
					`,
			Bindings: map[string]reflect.Value{
				"n.name": reflect.ValueOf(&n.Name),
				"n.age":  reflect.ValueOf(&n.Age),
			},
		})
	})

	t.Run("Remove a property", func(t *testing.T) {
		var n Person
		c := internal.NewCypherClient()
		cy, err := c.
			Match(db.Node(db.Qual(&n, "n", db.Props{"name": "'Andy'"}))).
			Set(db.SetPropValue(&n.Name, "null")).
			Return(&n.Name, &n.Age).
			Compile()

		check(t, cy, err, internal.CompiledCypher{
			Cypher: `
					MATCH (n:Person {name: 'Andy'})
					SET n.name = null
					RETURN n.name, n.age
					`,
			Bindings: map[string]reflect.Value{
				"n.name": reflect.ValueOf(&n.Name),
				"n.age":  reflect.ValueOf(&n.Age),
			},
		})
	})

	t.Run("Copy properties between nodes and relationships", func(t *testing.T) {
		var (
			at     Person
			pn     Person
			hungry bool
		)
		c := internal.NewCypherClient()
		cy, err := c.
			Match(
				db.Patterns(
					db.Node(db.Qual(&at, "at", db.Props{"name": "'Andy'"})),
					db.Node(db.Qual(&pn, "pn", db.Props{"name": "'Peter'"})),
				),
			).
			Set(db.SetPropValue(&at, "properties(pn)")).
			Return(&at.Name, &at.Age, db.Qual(&hungry, "at.hungry"), &pn.Name, &pn.Age).
			Compile()

		check(t, cy, err, internal.CompiledCypher{
			Cypher: `
					MATCH
					  (at:Person {name: 'Andy'}),
					  (pn:Person {name: 'Peter'})
					SET at = properties(pn)
					RETURN at.name, at.age, at.hungry, pn.name, pn.age
					`,
			Bindings: map[string]reflect.Value{
				"at.name":   reflect.ValueOf(&at.Name),
				"at.age":    reflect.ValueOf(&at.Age),
				"pn.name":   reflect.ValueOf(&pn.Name),
				"pn.age":    reflect.ValueOf(&pn.Age),
				"at.hungry": reflect.ValueOf(&hungry),
			},
		})
	})

	t.Run("Replace all properties using a map and =", func(t *testing.T) {
		var p Person
		c := internal.NewCypherClient()
		cy, err := c.
			Match(
				db.Node(db.Qual(&p, "p", db.Props{"name": "'Peter'"})),
			).
			Set(db.SetPropValue(&p, "{name: 'Peter Smith', position: 'Entrepreneur'}")).
			Return(&p.Name, &p.Age, &p.Position).
			Compile()

		check(t, cy, err, internal.CompiledCypher{
			Cypher: `
					MATCH (p:Person {name: 'Peter'})
					SET p = {name: 'Peter Smith', position: 'Entrepreneur'}
					RETURN p.name, p.age, p.position
					`,
			Bindings: map[string]reflect.Value{
				"p.name":     reflect.ValueOf(&p.Name),
				"p.age":      reflect.ValueOf(&p.Age),
				"p.position": reflect.ValueOf(&p.Position),
			},
		})
	})

	t.Run("Remove all properties using an empty map and =", func(t *testing.T) {
		var p Person
		c := internal.NewCypherClient()
		cy, err := c.
			Match(
				db.Node(db.Qual(&p, "p", db.Props{"name": "'Peter'"})),
			).
			Set(db.SetPropValue(&p, "{}")).
			Return(&p.Name, &p.Age).
			Compile()

		check(t, cy, err, internal.CompiledCypher{
			Cypher: `
					MATCH (p:Person {name: 'Peter'})
					SET p = {}
					RETURN p.name, p.age
					`,
			Bindings: map[string]reflect.Value{
				"p.name": reflect.ValueOf(&p.Name),
				"p.age":  reflect.ValueOf(&p.Age),
			},
		})
	})

	t.Run("Mutate specific properties using a map and +=", func(t *testing.T) {
		var p Person
		var hungry bool
		c := internal.NewCypherClient()
		cy, err := c.
			Match(
				db.Node(db.Qual(&p, "p", db.Props{"name": "'Peter'"})),
			).
			Set(db.SetMerge(&p, "{age: 38, hungry: true, position: 'Entrepreneur'}")).
			Return(&p.Name, &p.Age, db.Qual(&hungry, "p.hungry"), &p.Position).
			Compile()

		check(t, cy, err, internal.CompiledCypher{
			Cypher: `
					MATCH (p:Person {name: 'Peter'})
					SET p += {age: 38, hungry: true, position: 'Entrepreneur'}
					RETURN p.name, p.age, p.hungry, p.position
					`,
			Bindings: map[string]reflect.Value{
				"p.name":     reflect.ValueOf(&p.Name),
				"p.age":      reflect.ValueOf(&p.Age),
				"p.position": reflect.ValueOf(&p.Position),
				"p.hungry":   reflect.ValueOf(&hungry),
			},
		})
	})

	t.Run("Set multiple properties using one SET clause", func(t *testing.T) {
		var n Person
		c := internal.NewCypherClient()
		cy, err := c.
			Match(
				db.Node(db.Qual(&n, "n", db.Props{"name": "'Andy'"})),
			).
			Set(
				db.SetPropValue(&n.Position, "'Developer'"),
				db.SetPropValue(&n.Surname, "'Taylor'"),
			).
			Compile()

		check(t, cy, err, internal.CompiledCypher{
			Cypher: `
					MATCH (n:Person {name: 'Andy'})
					SET
					  n.position = 'Developer',
					  n.surname = 'Taylor'
					`,
		})
	})

	t.Run("Set a property using a parameter", func(t *testing.T) {
		var n Person
		c := internal.NewCypherClient()
		cy, err := c.
			Match(
				db.Node(db.Qual(&n, "n", db.Props{"name": "'Andy'"})),
			).
			Set(
				db.SetPropValue(&n.Surname, db.NamedParam("surname", "Taylor")),
			).
			Return(&n.Name, &n.Surname).
			Compile()

		check(t, cy, err, internal.CompiledCypher{
			Cypher: `
					MATCH (n:Person {name: 'Andy'})
					SET n.surname = $surname
					RETURN n.name, n.surname
					`,
			Bindings: map[string]reflect.Value{
				"n.name":    reflect.ValueOf(&n.Name),
				"n.surname": reflect.ValueOf(&n.Surname),
			},
			Parameters: map[string]any{
				"surname": "Taylor",
			},
		})
	})

	t.Run("Set all properties using a parameter", func(t *testing.T) {
		var n Person
		props := map[string]any{
			"name":     "Andy",
			"position": "Developer",
		}
		c := internal.NewCypherClient()
		cy, err := c.
			Match(
				db.Node(db.Qual(&n, "n", db.Props{"name": "'Andy'"})),
			).
			Set(
				db.SetPropValue(&n, db.NamedParam("props", props)),
			).
			Return(&n.Name).
			Compile()

		check(t, cy, err, internal.CompiledCypher{
			Cypher: `
					MATCH (n:Person {name: 'Andy'})
					SET n = $props
					RETURN n.name
					`,
			Bindings: map[string]reflect.Value{
				"n.name": reflect.ValueOf(&n.Name),
			},
			Parameters: map[string]any{
				"props": props,
			},
		})
	})

	t.Run("Set a label on a node", func(t *testing.T) {
		var n Person
		var labels []string
		c := internal.NewCypherClient()
		cy, err := c.
			Match(
				db.Node(db.Qual(&n, "n", db.Props{"name": "'Stefan'"})),
			).
			Set(db.SetLabels(&n, "German")).
			Return(&n.Name, db.Qual(&labels, "labels(n)", db.Name("labels"))).
			Compile()

		check(t, cy, err, internal.CompiledCypher{
			Cypher: `
					MATCH (n:Person {name: 'Stefan'})
					SET n:German
					RETURN n.name, labels(n) AS labels
					`,
			Bindings: map[string]reflect.Value{
				"n.name": reflect.ValueOf(&n.Name),
				"labels": reflect.ValueOf(&labels),
			},
		})
	})

	t.Run("Set multiple labels on a node", func(t *testing.T) {
		var n Person
		var labels []string
		c := internal.NewCypherClient()
		cy, err := c.
			Match(
				db.Node(db.Qual(&n, "n", db.Props{"name": "'George'"})),
			).
			Set(db.SetLabels(&n, "Swedish", "Bossman")).
			Return(&n.Name, db.Qual(&labels, "labels(n)", db.Name("labels"))).
			Compile()

		check(t, cy, err, internal.CompiledCypher{
			Cypher: `
					MATCH (n:Person {name: 'George'})
					SET n:Swedish:Bossman
					RETURN n.name, labels(n) AS labels
					`,
			Bindings: map[string]reflect.Value{
				"n.name": reflect.ValueOf(&n.Name),
				"labels": reflect.ValueOf(&labels),
			},
		})
	})
}
