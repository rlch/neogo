package tests

import (
	"reflect"
	"testing"

	"github.com/rlch/neogo/db"
	"github.com/rlch/neogo/internal"
)

func TestReturn(t *testing.T) {
	t.Run("Return nodes", func(t *testing.T) {
		var p Person
		c := internal.NewCypherClient()
		cy, err := c.
			Match(db.Node(db.Qual(
				&p, "p",
				db.Props{
					"name": "'Keanu Reeves'",
				},
			))).
			Return(&p).Compile()
		check(t, cy, err, internal.CompiledCypher{
			Cypher: `
					MATCH (p:Person {name: 'Keanu Reeves'})
					RETURN p
					`,
			Bindings: map[string]reflect.Value{
				"p": reflect.ValueOf(&p),
			},
		})
	})

	t.Run("Return relationships", func(t *testing.T) {
		var r string
		c := internal.NewCypherClient()
		cy, err := c.
			Match(
				db.Node(db.Qual(
					Person{},
					"p",
					db.Props{
						"name": "'Keanu Reeves'",
					},
				)).To(db.Qual(ActedIn{}, "r"), db.Var("m")),
			).
			Return(db.Qual(&r, "type(r)")).Compile()
		check(t, cy, err, internal.CompiledCypher{
			Cypher: `
					MATCH (p:Person {name: 'Keanu Reeves'})-[r:ACTED_IN]->(m)
					RETURN type(r)
					`,
			Bindings: map[string]reflect.Value{
				"type(r)": reflect.ValueOf(&r),
			},
		})
	})

	t.Run("Return property", func(t *testing.T) {
		var p Person
		c := internal.NewCypherClient()
		cy, err := c.
			Match(db.Node(db.Qual(
				&p, "p",
				db.Props{
					"name": "'Keanu Reeves'",
				},
			))).
			Return(&p.BornIn).Compile()
		check(t, cy, err, internal.CompiledCypher{
			Cypher: `
					MATCH (p:Person {name: 'Keanu Reeves'})
					RETURN p.bornIn
					`,
			Bindings: map[string]reflect.Value{
				"p.bornIn": reflect.ValueOf(&p.BornIn),
			},
		})
	})

	t.Run("Return all elements", func(t *testing.T) {
		// TODO(some kind soul): not sure if there's much of a use-case here.
		// Could maybe expose a FindAll() method that just gives the neo4j result directly?
	})

	t.Run("Variable with uncommon characters", func(t *testing.T) {
		// TODO: should be fine, just a pain to test
	})

	t.Run("Column alias", func(t *testing.T) {
		var p Person
		c := internal.NewCypherClient()
		cy, err := c.
			Match(db.Node(db.Qual(&p, "p", db.Props{"name": "'Keanu Reeves'"}))).
			Return(db.Qual(&p.Nationality, "citizenship")).Compile()
		check(t, cy, err, internal.CompiledCypher{
			Cypher: `
					MATCH (p:Person {name: 'Keanu Reeves'})
					RETURN p.nationality AS citizenship
					`,
			Bindings: map[string]reflect.Value{
				"citizenship": reflect.ValueOf(&p.Nationality),
			},
		})
	})

	t.Run("Optional properties", func(t *testing.T) {
		var bornIn any
		c := internal.NewCypherClient()
		cy, err := c.
			Match(db.Node(db.Var("n"))).
			Return(db.Qual(&bornIn, "n.bornIn")).Compile()
		check(t, cy, err, internal.CompiledCypher{
			Cypher: `
					MATCH (n)
					RETURN n.bornIn
					`,
			Bindings: map[string]reflect.Value{
				"n.bornIn": reflect.ValueOf(&bornIn),
			},
		})
	})

	t.Run("Other expressions", func(t *testing.T) {
		// TODO(some kind soul): not sure if pattern expressions are possible in the driver
	})

	t.Run("Unique results", func(t *testing.T) {
		var m []any
		c := internal.NewCypherClient()
		cy, err := c.
			Match(
				db.Node(db.Qual(Person{}, "p", db.Props{"name": "'Keanu Reeves'"})).
					To(nil, db.Qual(m, "m")),
			).
			Return(db.Return(m, db.Distinct)).Compile()
		check(t, cy, err, internal.CompiledCypher{
			Cypher: `
					MATCH (p:Person {name: 'Keanu Reeves'})-->(m)
					RETURN DISTINCT m
					`,
			Bindings: map[string]reflect.Value{
				"m": reflect.ValueOf(m),
			},
		})
	})
}
