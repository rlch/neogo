package tests

import (
	"reflect"
	"testing"

	"github.com/rlch/neogo/db"
	"github.com/rlch/neogo/internal"
)

func TestCreate(t *testing.T) {
	t.Run("Create nodes", func(t *testing.T) {
		t.Run("Create single node", func(t *testing.T) {
			c := internal.NewCypherClient()
			cy, err := c.
				Create(db.Node("n")).
				Compile()

			check(t, cy, err, internal.CompiledCypher{
				Cypher: `
					CREATE (n)
					`,
			})
		})

		t.Run("Create multiple nodes", func(t *testing.T) {
			c := internal.NewCypherClient()
			cy, err := c.
				Create(
					db.Patterns(
						db.Node("n"),
						db.Node("m"),
					),
				).
				Compile()

			check(t, cy, err, internal.CompiledCypher{
				Cypher: `
					CREATE
					  (n),
					  (m)
					`,
			})
		})

		t.Run("Create a node with a label", func(t *testing.T) {
			c := internal.NewCypherClient()
			cy, err := c.
				Create(db.Node(db.Qual(Person{}, "n"))).
				Compile()

			check(t, cy, err, internal.CompiledCypher{
				Cypher: `
					CREATE (n:Person)
					`,
			})
		})

		t.Run("Create a node with multiple labels", func(t *testing.T) {
			c := internal.NewCypherClient()
			type SwedishPerson struct {
				Person `neo4j:"Swedish"`
			}
			cy, err := c.
				Create(db.Node(db.Qual(SwedishPerson{}, "n"))).
				Compile()

			check(t, cy, err, internal.CompiledCypher{
				Cypher: `
					CREATE (n:Person:Swedish)
					`,
			})
		})

		t.Run("Return created node", func(t *testing.T) {
			c := internal.NewCypherClient()
			var name string
			cy, err := c.
				Create(db.Node(db.Var("a", db.Props{
					"name": "'Andy'",
				}))).
				Return(db.Qual(&name, "a.name")).
				Compile()

			check(t, cy, err, internal.CompiledCypher{
				Cypher: `
					CREATE (a {name: 'Andy'})
					RETURN a.name
					`,
				Bindings: map[string]reflect.Value{
					"a.name": reflect.ValueOf(&name),
				},
			})
		})
	})

	t.Run("Create relationships", func(t *testing.T) {
		t.Run("Create a relationship between two nodes", func(t *testing.T) {
			c := internal.NewCypherClient()
			var (
				a     Person
				b     Person
				typeR string
			)
			type Reltype struct {
				internal.RelationshipEntity `neo4j:"RELTYPE"`
			}
			cy, err := c.
				Match(db.Patterns(
					db.Node(db.Qual(&a, "a")),
					db.Node(db.Qual(&b, "b")),
				)).
				Where(
					db.And(
						db.Cond(&a.Name, "=", "'A'"),
						db.Cond(&b.Name, "=", "'B'"),
					),
				).
				Create(
					db.Node(&a).To(db.Qual(Reltype{}, "r"), &b),
				).
				Return(db.Qual(&typeR, "type(r)")).
				Compile()

			check(t, cy, err, internal.CompiledCypher{
				Cypher: `
					MATCH
					  (a:Person),
					  (b:Person)
					WHERE a.name = 'A' AND b.name = 'B'
					CREATE (a)-[r:RELTYPE]->(b)
					RETURN type(r)
					`,
				Bindings: map[string]reflect.Value{
					"type(r)": reflect.ValueOf(&typeR),
				},
			})
		})

		t.Run("Create a relationship and set properties", func(t *testing.T) {
			c := internal.NewCypherClient()
			type Reltype struct {
				internal.RelationshipEntity `neo4j:"RELTYPE"`

				Name string `json:"name"`
			}
			var (
				a     Person
				b     Person
				r     Reltype
				typeR string
			)
			cy, err := c.
				Match(db.Patterns(
					db.Node(db.Qual(&a, "a")),
					db.Node(db.Qual(&b, "b")),
				)).
				Where(
					db.And(
						db.Cond(&a.Name, "=", "'A'"),
						db.Cond(&b.Name, "=", "'B'"),
					),
				).
				Create(
					db.Node(&a).To(
						db.Qual(&r, "r", db.Props{
							&r.Name: "a.name + '<->' + b.name",
						}),
						&b,
					),
				).
				Return(
					db.Qual(&typeR, "type(r)"),
					&r.Name,
				).
				Compile()

			check(t, cy, err, internal.CompiledCypher{
				Cypher: `
					MATCH
					  (a:Person),
					  (b:Person)
					WHERE a.name = 'A' AND b.name = 'B'
					CREATE (a)-[r:RELTYPE {name: a.name + '<->' + b.name}]->(b)
					RETURN type(r), r.name
					`,
				Bindings: map[string]reflect.Value{
					"type(r)": reflect.ValueOf(&typeR),
					"r.name":  reflect.ValueOf(&r.Name),
				},
			})
		})

		t.Run("Create a full path", func(t *testing.T) {
			c := internal.NewCypherClient()
			var p any
			cy, err := c.
				Create(db.Path(
					db.Node(db.Var(Person{}, db.Props{"name": "'Andy'"})).
						To(WorksAt{}, db.Var(Company{}, db.Props{"name": "'Neo4j'"})).
						From(WorksAt{}, db.Var(Person{}, db.Props{"name": "'Michael'"})),
					"p",
				)).
				Return(db.Qual(&p, "p")).
				Compile()

			check(t, cy, err, internal.CompiledCypher{
				Cypher: `
					CREATE p = (:Person {name: 'Andy'})-[:WORKS_AT]->(:Company {name: 'Neo4j'})<-[:WORKS_AT]-(:Person {name: 'Michael'})
					RETURN p
					`,
				Bindings: map[string]reflect.Value{
					"p": reflect.ValueOf(&p),
				},
			})
		})
	})

	t.Run("Use parameters with CREATE", func(t *testing.T) {
		t.Run("Create node with a parameter for the properties", func(t *testing.T) {
			c := internal.NewCypherClient()
			n := Person{
				Name:     "Andy",
				Position: "Developer",
			}
			cy, err := c.
				Create(db.Node(db.Qual(&n, "n"))).
				Return(&n).
				Compile()

			check(t, cy, err, internal.CompiledCypher{
				Cypher: `
					CREATE (n:Person $n)
					RETURN n
					`,
				Parameters: map[string]any{
					"n": &n,
				},
				Bindings: map[string]reflect.Value{
					"n": reflect.ValueOf(&n),
				},
			})
		})

		t.Run("Create multiple nodes with a parameter for their properties", func(t *testing.T) {
			c := internal.NewCypherClient()
			people := []Person{
				{
					Name:     "Andy",
					Position: "Developer",
				},
				{
					Name:     "Michael",
					Position: "Developer",
				},
			}
			cy, err := c.
				Unwind(db.Qual(&people, "props"), "map").
				Create(db.Node("n")).
				Set(db.SetPropValue("n", "map")).
				Compile()

			check(t, cy, err, internal.CompiledCypher{
				Cypher: `
					UNWIND $props AS map
					CREATE (n)
					SET n = map
					`,
				Parameters: map[string]any{
					"props": &people,
				},
			})
		})
	})
}
