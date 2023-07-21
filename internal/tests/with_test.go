package tests

import (
	"reflect"
	"testing"

	"github.com/rlch/neo4j-gorm/db"
	"github.com/rlch/neo4j-gorm/internal"
)

func TestWith(t *testing.T) {
	t.Run("Introducing variables for expressions", func(t *testing.T) {
		var otherPersonName string
		c := internal.NewCypherClient()
		cy, err := c.
			Match(
				c.Node(db.Var("george", db.Props{"name": "'George'"})).
					From(nil, "otherPerson"),
			).
			With("otherPerson", db.Qual(db.Expr("toUpper(otherPerson.name)"), "upperCaseName")).
			Where(db.Cond("upperCaseName", "STARTS WITH", "'C'")).
			Find(db.Qual(&otherPersonName, "otherPerson.name")).Compile()

		check(t, cy, err, internal.CompiledCypher{
			Cypher: `
					MATCH (george {name: 'George'})<--(otherPerson)
					WITH otherPerson, toUpper(otherPerson.name) AS upperCaseName
					WHERE upperCaseName STARTS WITH 'C'
					RETURN otherPerson.name
					`,
			Bindings: map[string]reflect.Value{
				"otherPerson.name": reflect.ValueOf(&otherPersonName),
			},
		})
	})

	t.Run("Using the wildcard to carry over variables", func(t *testing.T) {
		var (
			personName      string
			otherPersonName string
			connectionType  string
		)
		c := internal.NewCypherClient()
		cy, err := c.
			Match(c.Node("person").To("r", "otherPerson")).
			With("*", db.Qual("type(r)", "connectionType")).
			Find(
				db.Qual(&personName, "person.name"),
				db.Qual(&otherPersonName, "otherPerson.name"),
				db.Qual(&connectionType, "connectionType"),
			).Compile()

		check(t, cy, err, internal.CompiledCypher{
			Cypher: `
					MATCH (person)-[r]->(otherPerson)
					WITH *, type(r) AS connectionType
					RETURN person.name, otherPerson.name, connectionType
					`,
			Bindings: map[string]reflect.Value{
				"person.name":      reflect.ValueOf(&personName),
				"otherPerson.name": reflect.ValueOf(&otherPersonName),
				"connectionType":   reflect.ValueOf(&connectionType),
			},
		})
	})

	t.Run("Filter on aggregate function results", func(t *testing.T) {
		var otherPersonName string

		c := internal.NewCypherClient()
		cy, err := c.
			Match(
				c.Node(db.Var("david", db.Props{"name": "'David'"})).
					Related(nil, "otherPerson").To(nil, nil),
			).
			With("otherPerson", db.Qual("count(*)", "foaf")).
			Where(db.Cond("foaf", ">", "1")).
			Find(
				db.Qual(&otherPersonName, "otherPerson.name"),
			).Compile()

		check(t, cy, err, internal.CompiledCypher{
			Cypher: `
					MATCH (david {name: 'David'})--(otherPerson)-->()
					WITH otherPerson, count(*) AS foaf
					WHERE foaf > 1
					RETURN otherPerson.name
					`,
			Bindings: map[string]reflect.Value{
				"otherPerson.name": reflect.ValueOf(&otherPersonName),
			},
		})
	})

	t.Run("Sort results before using collect on them", func(t *testing.T) {
		var names []string

		c := internal.NewCypherClient()
		cy, err := c.
			Match(c.Node("n")).
			With(
				db.With("n", db.OrderBy("name", false), db.Limit("3")),
			).
			Find(
				db.Qual(names, "collect(n.name)"),
			).Compile()

		check(t, cy, err, internal.CompiledCypher{
			Cypher: `
					MATCH (n)
					WITH n
					ORDER BY n.name DESC
					LIMIT 3
					RETURN collect(n.name)
					`,
			Bindings: map[string]reflect.Value{
				"collect(n.name)": reflect.ValueOf(names),
			},
		})
	})

	t.Run("Limit branching of a path search", func(t *testing.T) {
		var names []string

		c := internal.NewCypherClient()
		cy, err := c.
			Match(
				c.Node(db.Var("n", db.Props{"name": "'Anders'"})).
					Related(nil, "m"),
			).
			With(
				db.With("m", db.OrderBy("name", false), db.Limit("1")),
			).
			Match(c.Node("m").Related(nil, "o")).
			Find(
				db.Qual(names, "o.name"),
			).Compile()

		check(t, cy, err, internal.CompiledCypher{
			Cypher: `
					MATCH (n {name: 'Anders'})--(m)
					WITH m
					ORDER BY m.name DESC
					LIMIT 1
					MATCH (m)--(o)
					RETURN o.name
					`,
			Bindings: map[string]reflect.Value{
				"o.name": reflect.ValueOf(names),
			},
		})
	})

	t.Run("Limit and Filtering", func(t *testing.T) {
		var x []float64
		c := internal.NewCypherClient()
		cy, err := c.
			Unwind("[1, 2, 3, 4, 5, 6]", "x").
			With(db.With("x", db.Limit("5"), db.Where(db.Cond(db.Expr("x"), ">", "2")))).
			Find(db.Bind("x", &x)).Compile()

		check(t, cy, err, internal.CompiledCypher{
			Cypher: `
					UNWIND [1, 2, 3, 4, 5, 6] AS x
					WITH x
					LIMIT 5
					WHERE x > 2
					RETURN x
					`,
			Bindings: map[string]reflect.Value{
				"x": reflect.ValueOf(&x),
			},
		})
	})
}
