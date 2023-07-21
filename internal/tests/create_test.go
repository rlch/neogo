package tests

import (
	"reflect"
	"testing"

	neo4jgorm "github.com/rlch/neo4j-gorm"
	"github.com/rlch/neo4j-gorm/db"
	"github.com/rlch/neo4j-gorm/internal"
)

func TestCreate(t *testing.T) {
	t.Run("Create nodes", func(t *testing.T) {
		t.Run("Create single node", func(t *testing.T) {
			c := internal.NewCypherClient()
			cy, err := c.
				Create(c.Node("n")).
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
					c.Paths(
						c.Node("n"),
						c.Node("m"),
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
				Create(c.Node(db.Qual(Person{}, "n"))).
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
				Create(c.Node(db.Qual(SwedishPerson{}, "n"))).
				Compile()

			check(t, cy, err, internal.CompiledCypher{
				Cypher: `
					CREATE (n:Swedish:Person)
					`,
			})
		})

		t.Run("Return created node", func(t *testing.T) {
			c := internal.NewCypherClient()
			var name string
			cy, err := c.
				Create(c.Node(db.Var("a", db.Props{
					"name": "'Andy'",
				}))).
				Find(db.Qual(&name, "a.name")).
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
				neo4jgorm.Relationship `neo4j:"RELTYPE"`
			}
			cy, err := c.
				Match(c.Paths(
					c.Node(db.Qual(&a, "a")),
					c.Node(db.Qual(&b, "b")),
				)).
				Where(
					db.And(
						db.Cond(&a.Name, "=", "'A'"),
						db.Cond(&b.Name, "=", "'B'"),
					),
				).
				Create(
					c.Node(&a).To(db.Qual(Reltype{}, "r"), &b),
				).
				Find(db.Qual(&typeR, "type(r)")).
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
				neo4jgorm.Relationship `neo4j:"RELTYPE"`

				Name string `json:"name"`
			}
			var (
				a     Person
				b     Person
				r     Reltype
				typeR string
			)
			cy, err := c.
				Match(c.Paths(
					c.Node(db.Qual(&a, "a")),
					c.Node(db.Qual(&b, "b")),
				)).
				Where(
					db.And(
						db.Cond(&a.Name, "=", "'A'"),
						db.Cond(&b.Name, "=", "'B'"),
					),
				).
				Create(
					c.Node(&a).To(
						db.Qual(&r, "r", db.Props{
							&r.Name: "a.name + '<->' + b.name",
						}),
						&b,
					),
				).
				Find(
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
	})
}
