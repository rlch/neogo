package tests

import (
	"reflect"
	"testing"

	"github.com/rlch/neogo/db"
	"github.com/rlch/neogo/internal"
)

func TestMatch(t *testing.T) {
	t.Run("Basic node finding", func(t *testing.T) {
		t.Run("Get all nodes", func(t *testing.T) {
			var n []any
			c := internal.NewCypherClient()
			cy, err := c.Match(db.Node(db.Var(n, db.Name("n")))).Return(n).Compile()
			check(t, cy, err, internal.CompiledCypher{
				Cypher: `
					MATCH (n)
					RETURN n
					`,
				Bindings: map[string]reflect.Value{"n": reflect.ValueOf(n)},
			})
		})

		t.Run("Get all nodes with a label", func(t *testing.T) {
			var m Movie
			var mts []string
			c := internal.NewCypherClient()
			cy, err := c.Match(db.Node(&m)).Return(db.Bind(&m.Title, &mts)).Compile()
			check(t, cy, err, internal.CompiledCypher{
				Cypher: `
					MATCH (movie:Movie)
					RETURN movie.title
					`,
				Bindings: map[string]reflect.Value{
					"movie.title": reflect.ValueOf(&mts),
				},
			})
		})

		t.Run("Related nodes", func(t *testing.T) {
			var m Movie
			c := internal.NewCypherClient()
			cy, err := c.
				Match(
					db.Node(db.Var(
						"director",
						db.Props{
							"name": "'Oliver Stone'",
						},
					)).
						Related(nil, db.Var("movie"))).
				Return(db.Var(
					&m.Title,
					db.Name("movie.title"),
				)).Compile()
			check(t, cy, err, internal.CompiledCypher{
				Cypher: `
					MATCH (director {name: 'Oliver Stone'})--(movie)
					RETURN movie.title
					`,
				Bindings: map[string]reflect.Value{
					"movie.title": reflect.ValueOf(&m.Title),
				},
			})
		})

		t.Run("Match with labels", func(t *testing.T) {
			c := internal.NewCypherClient()
			var m Movie
			cy, err := c.Match(db.Node(db.Var(
				Person{},
				db.Props{
					"name": "'Oliver Stone'",
				},
			)).Related(nil, &m)).Return(&m.Title).Compile()
			check(t, cy, err, internal.CompiledCypher{
				Cypher: `
					MATCH (:Person {name: 'Oliver Stone'})--(movie:Movie)
					RETURN movie.title
					`,
				Bindings: map[string]reflect.Value{
					"movie.title": reflect.ValueOf(&m.Title),
				},
			})
		})

		t.Run("Match with a label expression for the node labels", func(t *testing.T) {
			c := internal.NewCypherClient()
			var name, title string
			cy, err := c.Match(
				db.Node(db.Var("n", db.Label("Movie|Person"))),
			).Return(
				db.Qual(&name, "n.name", db.Name("name")),
				db.Qual(&title, "n.title", db.Name("title")),
			).Compile()
			check(t, cy, err, internal.CompiledCypher{
				Cypher: `
					MATCH (n:Movie|Person)
					RETURN n.name AS name, n.title AS title
					`,
				Bindings: map[string]reflect.Value{
					"name":  reflect.ValueOf(&name),
					"title": reflect.ValueOf(&title),
				},
			})
		})
	})

	t.Run("Relationship basics", func(t *testing.T) {
		t.Run("Outgoing relationships", func(t *testing.T) {
			c := internal.NewCypherClient()
			var m Movie
			cy, err := c.
				Match(
					db.Node(db.Var(
						Person{},
						db.Props{
							"name": "'Oliver Stone'",
						},
					)).
						To(nil, db.Var("movie"))).
				Return(db.Qual(
					&m.Title,
					"movie.title",
				)).Compile()
			check(t, cy, err, internal.CompiledCypher{
				Cypher: `
					MATCH (:Person {name: 'Oliver Stone'})-->(movie)
					RETURN movie.title
					`,
				Bindings: map[string]reflect.Value{
					"movie.title": reflect.ValueOf(&m.Title),
				},
			})
		})

		t.Run("Relationship variables", func(t *testing.T) {
			c := internal.NewCypherClient()
			var out any
			cy, err := c.
				Match(db.Node(db.Var(
					Person{},
					db.Props{
						"name": "'Oliver Stone'",
					},
				)).To(db.Var("r"), db.Var("movie"))).
				Return(db.Qual(
					&out,
					"type(r)",
				)).Compile()
			check(t, cy, err, internal.CompiledCypher{
				Cypher: `
					MATCH (:Person {name: 'Oliver Stone'})-[r]->(movie)
					RETURN type(r)
					`,
				Bindings: map[string]reflect.Value{
					"type(r)": reflect.ValueOf(&out),
				},
			})
		})

		t.Run("Match on an undirected relationship", func(t *testing.T) {
			c := internal.NewCypherClient()
			var a, b any
			cy, err := c.
				Match(
					db.Node(db.Qual(&a, "a")).
						Related(
							db.Var(
								ActedIn{},
								db.Props{
									"role": "'Bud Fox'",
								},
							),
							db.Var(&b, db.Name("b")),
						)).
				Return(&a, &b).Compile()
			check(t, cy, err, internal.CompiledCypher{
				Cypher: `
					MATCH (a)-[:ACTED_IN {role: 'Bud Fox'}]-(b)
					RETURN a, b
					`,
				Bindings: map[string]reflect.Value{
					"a": reflect.ValueOf(&a),
					"b": reflect.ValueOf(&b),
				},
			})
		})

		t.Run("Match on relationship type", func(t *testing.T) {
			c := internal.NewCypherClient()
			var name string
			cy, err := c.
				Match(db.Node(db.Qual(
					Movie{},
					"wallstreet",
					db.Props{
						"title": "'Wall Street'",
					},
				)).From(ActedIn{}, db.Var("actor"))).
				Return(db.Qual(&name, "actor.name")).Compile()
			check(t, cy, err, internal.CompiledCypher{
				Cypher: `
					MATCH (wallstreet:Movie {title: 'Wall Street'})<-[:ACTED_IN]-(actor)
					RETURN actor.name
					`,
				Bindings: map[string]reflect.Value{
					"actor.name": reflect.ValueOf(&name),
				},
			})
		})

		t.Run("Match on multiple relationship types", func(t *testing.T) {
			c := internal.NewCypherClient()
			var name string
			cy, err := c.Match(db.Node(
				db.Var(
					"wallstreet",
					db.Props{"title": "'Wall Street'"},
				),
			).From(
				db.Var("", db.Label("ACTED_IN|DIRECTED")),
				db.Var("person"),
			)).Return(db.Qual(&name, "person.name")).Compile()
			check(t, cy, err, internal.CompiledCypher{
				Cypher: `
		MATCH (wallstreet {title: 'Wall Street'})<-[:ACTED_IN|DIRECTED]-(person)
		RETURN person.name
		`,
				Bindings: map[string]reflect.Value{
					"person.name": reflect.ValueOf(&name),
				},
			})
		})

		t.Run("Match on relationship type and use a variable", func(t *testing.T) {
			c := internal.NewCypherClient()
			var r ActedIn
			cy, err := c.
				Match(db.Node(db.Var(
					"wallstreet",
					db.Props{
						"title": "'Wall Street'",
					},
				)).From(db.Qual(&r, "r"), db.Var("actor"))).
				Return(&r.Role).Compile()
			check(t, cy, err, internal.CompiledCypher{
				Cypher: `
					MATCH (wallstreet {title: 'Wall Street'})<-[r:ACTED_IN]-(actor)
					RETURN r.role
					`,
				Bindings: map[string]reflect.Value{
					"r.role": reflect.ValueOf(&r.Role),
				},
			})
		})
	})

	t.Run("Relationships in depth", func(t *testing.T) {
		t.Run("Relationship types with uncommon characters", func(t *testing.T) {
			c := internal.NewCypherClient()
			var martin, rob Person
			cy, err := c.
				Match(
					db.Patterns(
						db.Node(db.Qual(
							&martin,
							"martin",
							db.Props{
								"name": "'Martin Sheen'",
							},
						)),
						db.Node(db.Qual(
							&rob,
							"rob",
							db.Props{
								"name": "'Rob Reiner'",
							},
						)),
					),
				).
				Create(
					db.Node(&rob).
						To(
							db.Var(nil, db.Label("`OLD FRIENDS`")),
							&martin,
						),
				).Compile()
			check(t, cy, err, internal.CompiledCypher{
				Cypher: `
		MATCH
		  (martin:Person {name: 'Martin Sheen'}),
		  (rob:Person {name: 'Rob Reiner'})
		CREATE (rob)-[:` + "`OLD FRIENDS`" + `]->(martin)
		`,
			})
		})

		t.Run("Multiple relationships", func(t *testing.T) {
			c := internal.NewCypherClient()
			var mTitle, dName string
			cy, err := c.
				Match(
					db.Node(db.Var(
						"charlie",
						db.Props{
							"name": "'Charlie Sheen'",
						},
					)).
						To(ActedIn{}, db.Var("movie")).
						From(Directed{}, db.Var("director")),
				).
				Return(
					db.Qual(&mTitle, "movie.title"),
					db.Qual(&dName, "director.name"),
				).Compile()
			check(t, cy, err, internal.CompiledCypher{
				Cypher: `
					MATCH (charlie {name: 'Charlie Sheen'})-[:ACTED_IN]->(movie)<-[:DIRECTED]-(director)
					RETURN movie.title, director.name
					`,
				Bindings: map[string]reflect.Value{
					"movie.title":   reflect.ValueOf(&mTitle),
					"director.name": reflect.ValueOf(&dName),
				},
			})
		})
	})

	t.Run("OPTIONAL MATCH", func(t *testing.T) {
		t.Run("In more detail", func(t *testing.T) {
			c := internal.NewCypherClient()
			a := Person{}
			r := Directed{}
			cy, err := c.
				Match(
					db.Node(db.Qual(
						&a,
						"a",
						db.Props{
							"name": "'Martin Sheen'",
						},
					)),
				).
				Match(
					db.Node(&a).To(db.Qual(&r, "r"), nil),
					db.Optional,
				).Return(&a.Name, &r).Compile()
			check(t, cy, err, internal.CompiledCypher{
				Cypher: `
		MATCH (a:Person {name: 'Martin Sheen'})
		OPTIONAL MATCH (a)-[r:DIRECTED]->()
		RETURN a.name, r
		`,
				Bindings: map[string]reflect.Value{
					"a.name": reflect.ValueOf(&a.Name),
					"r":      reflect.ValueOf(&r),
				},
			})
		})

		t.Run("Optional relationships", func(t *testing.T) {
			var (
				a Person
				x any
			)
			c := internal.NewCypherClient()
			cy, err := c.
				Match(db.Node(
					db.Qual(
						&a, "a",
						db.Props{"name": "'Charlie Sheen'"},
					),
				)).
				Match(db.Node(&a).To(nil, db.Qual(&x, "x")), db.Optional).
				Return(&x).Compile()

			check(t, cy, err, internal.CompiledCypher{
				Cypher: `
		MATCH (a:Person {name: 'Charlie Sheen'})
		OPTIONAL MATCH (a)-->(x)
		RETURN x
		`,
				Bindings: map[string]reflect.Value{
					"x": reflect.ValueOf(&x),
				},
			})
		})

		t.Run("Properties on optional elements", func(t *testing.T) {
			var (
				a    Person
				x    any
				name string
			)
			c := internal.NewCypherClient()
			cy, err := c.
				Match(db.Node(
					db.Qual(
						&a, "a",
						db.Props{"name": "'Martin Sheen'"},
					),
				)).
				Match(db.Node(&a).To(nil, db.Qual(&x, "x")), db.Optional).
				Return(&x, db.Qual(&name, "x.name")).Compile()

			check(t, cy, err, internal.CompiledCypher{
				Cypher: `
		MATCH (a:Person {name: 'Martin Sheen'})
		OPTIONAL MATCH (a)-->(x)
		RETURN x, x.name
		`,
				Bindings: map[string]reflect.Value{
					"x":      reflect.ValueOf(&x),
					"x.name": reflect.ValueOf(&name),
				},
			})
		})

		t.Run("Optional typed and named relationship", func(t *testing.T) {
			var (
				a     Movie
				name  string
				typeR string
			)
			c := internal.NewCypherClient()
			cy, err := c.
				Match(db.Node(
					db.Qual(
						&a, "a",
						db.Props{"title": "'Wall Street'"},
					),
				)).
				Match(db.Node(db.Var("x")).To(db.Qual(ActedIn{}, "r"), &a), db.Optional).
				Return(&a.Title, db.Qual(&name, "x.name"), db.Qual(&typeR, "type(r)")).Compile()

			check(t, cy, err, internal.CompiledCypher{
				Cypher: `
		MATCH (a:Movie {title: 'Wall Street'})
		OPTIONAL MATCH (x)-[r:ACTED_IN]->(a)
		RETURN a.title, x.name, type(r)
		`,
				Bindings: map[string]reflect.Value{
					"a.title": reflect.ValueOf(&a.Title),
					"type(r)": reflect.ValueOf(&typeR),
					"x.name":  reflect.ValueOf(&name),
				},
			})
		})
	})
}
