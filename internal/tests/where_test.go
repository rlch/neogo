package tests

import (
	"reflect"
	"testing"

	"github.com/rlch/neogo/db"
	"github.com/rlch/neogo/internal"
	"github.com/stretchr/testify/require"
)

func TestWhere(t *testing.T) {
	t.Run("Basic usage", func(t *testing.T) {
		t.Run("Node pattern predicates", func(t *testing.T) {
			var b Person
			c := internal.NewCypherClient(r)
			cy, err := c.
				With(db.Qual("30", "minAge")).
				Match(
					db.Node(db.Qual(Person{}, "a", db.Where(db.Cond("name", "=", "'Andy'")))).
						To(Knows{}, db.Qual(&b, "b", db.Where(db.Cond("age", ">", "minAge")))),
				).Return(&b.Name).Compile()

			Check(t, cy, err, internal.CompiledCypher{
				Cypher: `
					WITH 30 AS minAge
					MATCH (a:Person WHERE a.name = 'Andy')-[:KNOWS]->(b:Person WHERE b.age > minAge)
					RETURN b.name
					`,
				Bindings: map[string]reflect.Value{
					"b.name": reflect.ValueOf(&b.Name),
				},
			})

			var names []string
			c = internal.NewCypherClient(r)
			cy, err = c.
				Match(
					db.Node(db.Qual(Person{}, "a", db.Props{"name": "'Andy'"})),
				).
				Return(db.Qual(&names, "[(a)-->(b WHERE b:Person) | b.name]", db.Name("friends"))).Compile()

			Check(t, cy, err, internal.CompiledCypher{
				Cypher: `
					MATCH (a:Person {name: 'Andy'})
					RETURN [(a)-->(b WHERE b:Person) | b.name] AS friends
					`,
				Bindings: map[string]reflect.Value{
					"friends": reflect.ValueOf(&names),
				},
			})
		})

		t.Run("Boolean operations", func(t *testing.T) {
			var n Person
			c := internal.NewCypherClient(r)
			cy, err := c.
				Match(db.Node(db.Qual(&n, "n"))).
				Where(
					db.Or(
						db.Xor(
							db.Cond(&n.Name, "=", "'Peter'"),
							db.And(
								db.Cond(&n.Age, "<", "30"),
								db.Cond(&n.Name, "=", "'Timothy'"),
							),
						),
						db.Not(db.Or(
							db.Cond(&n.Name, "=", "'Timothy'"),
							db.Cond(&n.Name, "=", "'Peter'"),
						)),
					),
				).
				Return(
					db.Return(db.Qual(&n.Name, "name"), db.OrderBy("", true)),
					db.Qual(&n.Age, "age"),
				).Compile()

			Check(t, cy, err, internal.CompiledCypher{
				Cypher: `
					MATCH (n:Person)
					WHERE (n.name = 'Peter' XOR (n.age < 30 AND n.name = 'Timothy')) OR NOT (n.name = 'Timothy' OR n.name = 'Peter')
					RETURN n.name AS name, n.age AS age
					ORDER BY name
					`,
				Bindings: map[string]reflect.Value{
					"name": reflect.ValueOf(&n.Name),
					"age":  reflect.ValueOf(&n.Age),
				},
			})
		})

		t.Run("Filter on node label", func(t *testing.T) {
			var (
				name string
				age  int
			)
			c := internal.NewCypherClient(r)
			cy, err := c.
				Match(db.Node("n")).
				Where(db.Expr("n:Swedish")).
				Return(
					db.Qual(&name, "n.name"),
					db.Qual(&age, "n.age"),
				).Compile()

			Check(t, cy, err, internal.CompiledCypher{
				Cypher: `
					MATCH (n)
					WHERE n:Swedish
					RETURN n.name, n.age
					`,
				Bindings: map[string]reflect.Value{
					"n.name": reflect.ValueOf(&name),
					"n.age":  reflect.ValueOf(&age),
				},
			})
		})

		t.Run("Filter on node label", func(t *testing.T) {
			var (
				name string
				age  int
			)
			c := internal.NewCypherClient(r)
			cy, err := c.
				Match(db.Node("n")).
				Where(db.Expr("n:Swedish")).
				Return(
					db.Qual(&name, "n.name"),
					db.Qual(&age, "n.age"),
				).Compile()

			Check(t, cy, err, internal.CompiledCypher{
				Cypher: `
					MATCH (n)
					WHERE n:Swedish
					RETURN n.name, n.age
					`,
				Bindings: map[string]reflect.Value{
					"n.name": reflect.ValueOf(&name),
					"n.age":  reflect.ValueOf(&age),
				},
			})
		})

		t.Run("Filter on node property", func(t *testing.T) {
			var n Person
			c := internal.NewCypherClient(r)
			cy, err := c.
				Match(db.Node(db.Qual(&n, "n"))).
				Where(db.Cond(&n.Age, "<", "30")).
				Return(
					&n.Name,
					&n.Age,
				).Compile()

			Check(t, cy, err, internal.CompiledCypher{
				Cypher: `
					MATCH (n:Person)
					WHERE n.age < 30
					RETURN n.name, n.age
					`,
				Bindings: map[string]reflect.Value{
					"n.name": reflect.ValueOf(&n.Name),
					"n.age":  reflect.ValueOf(&n.Age),
				},
			})
		})

		t.Run("Filter on relationship property", func(t *testing.T) {
			var (
				name  string
				age   int
				email string
			)
			c := internal.NewCypherClient(r)
			cy, err := c.
				Match(
					db.Node(db.Qual(Person{}, "n")).
						To(db.Qual(Knows{}, "k"), "f"),
				).
				Where(db.Cond("k.since", "<", "2000")).
				Return(
					db.Qual(&name, "f.name"),
					db.Qual(&age, "f.age"),
					db.Qual(&email, "f.email"),
				).Compile()

			Check(t, cy, err, internal.CompiledCypher{
				Cypher: `
					MATCH (n:Person)-[k:KNOWS]->(f)
					WHERE k.since < 2000
					RETURN f.name, f.age, f.email
					`,
				Bindings: map[string]reflect.Value{
					"f.name":  reflect.ValueOf(&name),
					"f.age":   reflect.ValueOf(&age),
					"f.email": reflect.ValueOf(&email),
				},
			})
		})

		t.Run("Filter on dynamically-computed node property", func(t *testing.T) {
			var n Person
			c := internal.NewCypherClient(r)
			cy, err := c.
				With(db.Qual("'AGE'", "propname")).
				Match(
					db.Node(db.Qual(&n, "n")),
				).
				Where(db.Cond("n[toLower(propname)]", "<", "30")).
				Return(
					&n.Name,
					&n.Age,
				).Compile()

			Check(t, cy, err, internal.CompiledCypher{
				Cypher: `
					WITH 'AGE' AS propname
					MATCH (n:Person)
					WHERE n[toLower(propname)] < 30
					RETURN n.name, n.age
					`,
				Bindings: map[string]reflect.Value{
					"n.name": reflect.ValueOf(&n.Name),
					"n.age":  reflect.ValueOf(&n.Age),
				},
			})
		})

		t.Run("Property existence checking", func(t *testing.T) {
			var n Person
			c := internal.NewCypherClient(r)
			cy, err := c.
				Match(
					db.Node(db.Qual(&n, "n")),
				).
				Where(db.Cond(&n.Belt, "IS NOT", "NULL")).
				Return(
					&n.Name,
					&n.Belt,
				).Compile()

			Check(t, cy, err, internal.CompiledCypher{
				Cypher: `
					MATCH (n:Person)
					WHERE n.belt IS NOT NULL
					RETURN n.name, n.belt
					`,
				Bindings: map[string]reflect.Value{
					"n.name": reflect.ValueOf(&n.Name),
					"n.belt": reflect.ValueOf(&n.Belt),
				},
			})
		})

		t.Run("Using WITH", func(t *testing.T) {
			var n Person
			c := internal.NewCypherClient(r)
			cy, err := c.
				Match(
					db.Node(db.Qual(&n, "n")),
				).
				With(db.With(
					db.Qual(&n.Name, "name"),
					db.Where(db.Cond(&n.Age, "=", "25")),
				)).
				Return(&n.Name).Compile()

			Check(t, cy, err, internal.CompiledCypher{
				Cypher: `
					MATCH (n:Person)
					WITH n.name AS name
					WHERE n.age = 25
					RETURN name
					`,
				Bindings: map[string]reflect.Value{
					"name": reflect.ValueOf(&n.Name),
				},
			})
		})
	})

	t.Run("String matching", func(t *testing.T) {
		t.Run("Prefix string search using STARTS WITH", func(t *testing.T) {
			var n Person
			c := internal.NewCypherClient(r)
			cy, err := c.
				Match(db.Node(db.Qual(&n, "n"))).
				Where(db.Cond(&n.Name, "STARTS WITH", "'Pet'")).
				Return(&n.Name, &n.Age).Compile()

			Check(t, cy, err, internal.CompiledCypher{
				Cypher: `
					MATCH (n:Person)
					WHERE n.name STARTS WITH 'Pet'
					RETURN n.name, n.age
					`,
				Bindings: map[string]reflect.Value{
					"n.name": reflect.ValueOf(&n.Name),
					"n.age":  reflect.ValueOf(&n.Age),
				},
			})
		})

		t.Run("Suffix string search using ENDS WITH", func(t *testing.T) {
			var n Person
			c := internal.NewCypherClient(r)
			cy, err := c.
				Match(db.Node(db.Qual(&n, "n"))).
				Where(db.Cond(&n.Name, "ENDS WITH", "'ter'")).
				Return(&n.Name, &n.Age).Compile()

			Check(t, cy, err, internal.CompiledCypher{
				Cypher: `
					MATCH (n:Person)
					WHERE n.name ENDS WITH 'ter'
					RETURN n.name, n.age
					`,
				Bindings: map[string]reflect.Value{
					"n.name": reflect.ValueOf(&n.Name),
					"n.age":  reflect.ValueOf(&n.Age),
				},
			})
		})

		t.Run("Substring search using CONTAINS", func(t *testing.T) {
			var n Person
			c := internal.NewCypherClient(r)
			cy, err := c.
				Match(db.Node(db.Qual(&n, "n"))).
				Where(db.Cond(&n.Name, "CONTAINS", "'ete'")).
				Return(&n.Name, &n.Age).Compile()

			Check(t, cy, err, internal.CompiledCypher{
				Cypher: `
					MATCH (n:Person)
					WHERE n.name CONTAINS 'ete'
					RETURN n.name, n.age
					`,
				Bindings: map[string]reflect.Value{
					"n.name": reflect.ValueOf(&n.Name),
					"n.age":  reflect.ValueOf(&n.Age),
				},
			})
		})

		t.Run("String matching negation", func(t *testing.T) {
			var n Person
			c := internal.NewCypherClient(r)
			cy, err := c.
				Match(db.Node(db.Qual(&n, "n"))).
				Where(db.Not(db.Cond(&n.Name, "ENDS WITH", "'y'"))).
				Return(&n.Name, &n.Age).Compile()

			Check(t, cy, err, internal.CompiledCypher{
				Cypher: `
					MATCH (n:Person)
					WHERE NOT n.name ENDS WITH 'y'
					RETURN n.name, n.age
					`,
				Bindings: map[string]reflect.Value{
					"n.name": reflect.ValueOf(&n.Name),
					"n.age":  reflect.ValueOf(&n.Age),
				},
			})
		})
	})

	t.Run("Regular expressions", func(t *testing.T) {
		t.Run("Matching using regular expressions", func(t *testing.T) {
			var n Person
			c := internal.NewCypherClient(r)
			cy, err := c.
				Match(db.Node(db.Qual(&n, "n"))).
				Where(db.Cond(&n.Name, "=~", "'Tim.*'")).
				Return(&n.Name, &n.Age).Compile()

			Check(t, cy, err, internal.CompiledCypher{
				Cypher: `
					MATCH (n:Person)
					WHERE n.name =~ 'Tim.*'
					RETURN n.name, n.age
					`,
				Bindings: map[string]reflect.Value{
					"n.name": reflect.ValueOf(&n.Name),
					"n.age":  reflect.ValueOf(&n.Age),
				},
			})
		})

		t.Run("Escaping in regular expressions", func(t *testing.T) {
			var n Person
			c := internal.NewCypherClient(r)
			cy, err := c.
				Match(db.Node(db.Qual(&n, "n"))).
				Where(db.Cond(&n.Email, "=~", "'.*\\\\.com'")).
				Return(&n.Name, &n.Age, &n.Email).Compile()

			Check(t, cy, err, internal.CompiledCypher{
				Cypher: `
					MATCH (n:Person)
					WHERE n.email =~ '.*\\.com'
					RETURN n.name, n.age, n.email
					`,
				Bindings: map[string]reflect.Value{
					"n.name":  reflect.ValueOf(&n.Name),
					"n.age":   reflect.ValueOf(&n.Age),
					"n.email": reflect.ValueOf(&n.Email),
				},
			})
		})

		t.Run("Case-insensitive regular expressions", func(t *testing.T) {
			var n Person
			c := internal.NewCypherClient(r)
			cy, err := c.
				Match(db.Node(db.Qual(&n, "n"))).
				Where(db.Cond(&n.Name, "=~", "'(?i)AND.*'")).
				Return(&n.Name, &n.Age).Compile()

			Check(t, cy, err, internal.CompiledCypher{
				Cypher: `
				MATCH (n:Person)
				WHERE n.name =~ '(?i)AND.*'
				RETURN n.name, n.age
				`,
				Bindings: map[string]reflect.Value{
					"n.name": reflect.ValueOf(&n.Name),
					"n.age":  reflect.ValueOf(&n.Age),
				},
			})
		})
	})

	t.Run("Using path patterns in WHERE", func(t *testing.T) {
		t.Run("Filter on patterns", func(t *testing.T) {
			var (
				timothy Person
				other   Person
			)
			c := internal.NewCypherClient(r)
			cy, err := c.
				Match(
					db.Patterns(
						db.Node(db.Qual(&timothy, "timothy", db.Props{"name": "'Timothy'"})),
						db.Node(db.Qual(&other, "other")),
					),
				).
				Where(db.And(
					db.Cond(&other.Name, "IN", "['Andy', 'Peter']"),
					db.Node(&other).To(nil, &timothy),
				)).
				Return(&other.Name, &other.Age).Compile()

			Check(t, cy, err, internal.CompiledCypher{
				Cypher: `
				MATCH
				  (timothy:Person {name: 'Timothy'}),
				  (other:Person)
				WHERE other.name IN ['Andy', 'Peter'] AND (other)-->(timothy)
				RETURN other.name, other.age
				`,
				Bindings: map[string]reflect.Value{
					"other.name": reflect.ValueOf(&other.Name),
					"other.age":  reflect.ValueOf(&other.Age),
				},
			})
		})

		t.Run("Filter on patterns using NOT", func(t *testing.T) {
			var (
				peter  Person
				person Person
			)
			c := internal.NewCypherClient(r)
			cy, err := c.
				Match(
					db.Patterns(
						db.Node(db.Qual(&person, "person")),
						db.Node(db.Qual(&peter, "peter", db.Props{"name": "'Peter'"})),
					),
				).
				Where(db.Not(
					db.Node(&person).To(nil, &peter),
				)).
				Return(&person.Name, &person.Age).Compile()

			Check(t, cy, err, internal.CompiledCypher{
				Cypher: `
				MATCH
				  (person:Person),
				  (peter:Person {name: 'Peter'})
				WHERE NOT (person)-->(peter)
				RETURN person.name, person.age
				`,
				Bindings: map[string]reflect.Value{
					"person.name": reflect.ValueOf(&person.Name),
					"person.age":  reflect.ValueOf(&person.Age),
				},
			})
		})

		t.Run("Filter on patterns with properties", func(t *testing.T) {
			var n Person
			c := internal.NewCypherClient(r)
			cy, err := c.
				Match(db.Node(db.Qual(&n, "n"))).
				Where(
					db.Node(&n).Related(
						Knows{},
						db.Var(nil, db.Props{"name": "'Timothy'"}),
					),
				).
				Return(&n.Name, &n.Age).Compile()

			Check(t, cy, err, internal.CompiledCypher{
				Cypher: `
				MATCH (n:Person)
				WHERE (n)-[:KNOWS]-({name: 'Timothy'})
				RETURN n.name, n.age
				`,
				Bindings: map[string]reflect.Value{
					"n.name": reflect.ValueOf(&n.Name),
					"n.age":  reflect.ValueOf(&n.Age),
				},
			})
		})

		t.Run("Filter on relationship type", func(t *testing.T) {
			var (
				n     Person
				typeR string
				since int
			)
			c := internal.NewCypherClient(r)
			cy, err := c.
				Match(
					db.Node(db.Qual(&n, "n")).
						To("r", nil),
				).
				Where(
					db.And(
						db.Cond(&n.Name, "=", "'Andy'"),
						db.Cond("type(r)", "=~", "'K.*'"),
					),
				).
				Return(
					db.Qual(&typeR, "type(r)"),
					db.Qual(&since, "r.since"),
				).Compile()

			Check(t, cy, err, internal.CompiledCypher{
				Cypher: `
				MATCH (n:Person)-[r]->()
				WHERE n.name = 'Andy' AND type(r) =~ 'K.*'
				RETURN type(r), r.since
				`,
				Bindings: map[string]reflect.Value{
					"type(r)": reflect.ValueOf(&typeR),
					"r.since": reflect.ValueOf(&since),
				},
			})
		})
	})

	t.Run("Lists", func(t *testing.T) {
		t.Run("IN operator", func(t *testing.T) {
			var a Person
			c := internal.NewCypherClient(r)
			cy, err := c.
				Match(db.Node(db.Qual(&a, "a"))).
				Where(
					db.Cond(&a.Name, "IN", "['Peter', 'Timothy']"),
				).
				Return(&a.Name, &a.Age).Compile()

			Check(t, cy, err, internal.CompiledCypher{
				Cypher: `
				MATCH (a:Person)
				WHERE a.name IN ['Peter', 'Timothy']
				RETURN a.name, a.age
				`,
				Bindings: map[string]reflect.Value{
					"a.age":  reflect.ValueOf(&a.Age),
					"a.name": reflect.ValueOf(&a.Name),
				},
			})
		})
	})

	t.Run("Missing properties and values", func(t *testing.T) {
		t.Run("Default to false if property is missing", func(t *testing.T) {
			var n Person
			c := internal.NewCypherClient(r)
			cy, err := c.
				Match(db.Node(db.Qual(&n, "n"))).
				Where(
					db.Cond(&n.Belt, "=", "'white'"),
				).
				Return(&n.Name, &n.Age, &n.Belt).Compile()

			Check(t, cy, err, internal.CompiledCypher{
				Cypher: `
				MATCH (n:Person)
				WHERE n.belt = 'white'
				RETURN n.name, n.age, n.belt
				`,
				Bindings: map[string]reflect.Value{
					"n.age":  reflect.ValueOf(&n.Age),
					"n.name": reflect.ValueOf(&n.Name),
					"n.belt": reflect.ValueOf(&n.Belt),
				},
			})
		})

		t.Run("Default to true if property is missing", func(t *testing.T) {
			var n Person
			c := internal.NewCypherClient(r)
			cy, err := c.
				Match(db.Node(db.Qual(&n, "n"))).
				Where(
					db.Or(
						db.Cond(&n.Belt, "=", "'white'"),
						db.Cond(&n.Belt, "IS", "NULL"),
					),
				).
				Return(
					db.Return(&n.Name, db.OrderBy("", true)),
					&n.Age, &n.Belt,
				).Compile()

			Check(t, cy, err, internal.CompiledCypher{
				Cypher: `
				MATCH (n:Person)
				WHERE n.belt = 'white' OR n.belt IS NULL
				RETURN n.name, n.age, n.belt
				ORDER BY n.name
				`,
				Bindings: map[string]reflect.Value{
					"n.age":  reflect.ValueOf(&n.Age),
					"n.name": reflect.ValueOf(&n.Name),
					"n.belt": reflect.ValueOf(&n.Belt),
				},
			})
		})

		t.Run("Filter on null", func(t *testing.T) {
			var person Person
			c := internal.NewCypherClient(r)
			cy, err := c.
				Match(db.Node(db.Qual(&person, "person"))).
				Where(
					db.And(
						db.Cond(&person.Name, "=", "'Peter'"),
						db.Cond(&person.Belt, "IS", "NULL"),
					),
				).
				Return(&person.Name, &person.Age, &person.Belt).Compile()

			Check(t, cy, err, internal.CompiledCypher{
				Cypher: `
				MATCH (person:Person)
				WHERE person.name = 'Peter' AND person.belt IS NULL
				RETURN person.name, person.age, person.belt
				`,
				Bindings: map[string]reflect.Value{
					"person.age":  reflect.ValueOf(&person.Age),
					"person.name": reflect.ValueOf(&person.Name),
					"person.belt": reflect.ValueOf(&person.Belt),
				},
			})
		})
	})

	t.Run("Using ranges", func(t *testing.T) {
		t.Run("Simple range", func(t *testing.T) {
			var a Person
			c := internal.NewCypherClient(r)
			cy, err := c.
				Match(db.Node(db.Qual(&a, "a"))).
				Where(
					db.Cond(&a.Name, ">=", "'Peter'"),
				).
				Return(&a.Name, &a.Age).Compile()

			Check(t, cy, err, internal.CompiledCypher{
				Cypher: `
				MATCH (a:Person)
				WHERE a.name >= 'Peter'
				RETURN a.name, a.age
				`,
				Bindings: map[string]reflect.Value{
					"a.age":  reflect.ValueOf(&a.Age),
					"a.name": reflect.ValueOf(&a.Name),
				},
			})
		})

		t.Run("Composite range", func(t *testing.T) {
			var a Person
			c := internal.NewCypherClient(r)
			cy, err := c.
				Match(db.Node(db.Qual(&a, "a"))).
				Where(
					db.And(
						db.Cond(&a.Name, ">", "'Andy'"),
						db.Cond(&a.Name, "<", "'Timothy'"),
					),
				).
				Return(&a.Name, &a.Age).Compile()

			Check(t, cy, err, internal.CompiledCypher{
				Cypher: `
				MATCH (a:Person)
				WHERE a.name > 'Andy' AND a.name < 'Timothy'
				RETURN a.name, a.age
				`,
				Bindings: map[string]reflect.Value{
					"a.age":  reflect.ValueOf(&a.Age),
					"a.name": reflect.ValueOf(&a.Name),
				},
			})
		})
	})

	t.Run("Pattern element predicates", func(t *testing.T) {
		t.Run("Relationship pattern predicates", func(t *testing.T) {
			var (
				a     Person
				knows Knows
			)
			c := internal.NewCypherClient(r)
			cy, err := c.
				With(db.Qual("2000", "minYear")).
				Match(
					db.Node(db.Qual(&a, "a")).
						To(
							db.Qual(&knows, "r", db.Where(db.Cond("since", "<", "minYear"))),
							db.Qual(Person{}, "b"),
						),
				).
				Return(&knows.Since).Compile()

			Check(t, cy, err, internal.CompiledCypher{
				Cypher: `
				WITH 2000 AS minYear
				MATCH (a:Person)-[r:KNOWS WHERE r.since < minYear]->(b:Person)
				RETURN r.since
				`,
				Bindings: map[string]reflect.Value{
					"r.since": reflect.ValueOf(&knows.Since),
				},
			})
		})
	})

	t.Run("Shorthand syntax", func(t *testing.T) {
		t.Run("Substitutes _ for the current identifier", func(t *testing.T) {
			c := internal.NewCypherClient(r)
			statuses := []string{"active", "processing"}
			cy, err := c.
				Match(
					db.Node(
						"n",
						db.Where(
							"_.archived = ? AND _.status IN ?",
							false,
							db.NamedParam(statuses, "statuses"),
						),
					),
				).
				Compile()
			Check(t, cy, err, internal.CompiledCypher{
				Cypher: `
        MATCH (n WHERE n.archived = false AND n.status IN $statuses)
        `,
				Parameters: map[string]any{
					"statuses": statuses,
				},
			})
		})
	})

	t.Run("Implicit syntax", func(t *testing.T) {
		// t.Run("Tries to convert 3 arguments to a Cond()", func(t *testing.T) {
		// 	c := internal.NewCypherClient(r)
		// 	cy, err := c.
		// 		Match(db.Node("n")).
		// 		Where("n.id", "=", db.String("123")).
		// 		Return("n").
		// 		Compile()
		// 	Check(t, cy, err, internal.CompiledCypher{
		// 		Cypher: `
		// 		MATCH (n)
		// 		WHERE n.id = "123"
		// 		RETURN n
		// 		`,
		// 	})
		// })

		t.Run("Forwards all WhereOptions", func(t *testing.T) {
			c := internal.NewCypherClient(r)
			cy, err := c.
				Match(db.Node("n")).
				Where(
					db.Cond("n.id", "=", db.String("123")),
					db.Cond("n.name", "=", db.String("Test")),
				).
				Return("n").
				Compile()
			Check(t, cy, err, internal.CompiledCypher{
				Cypher: `
				MATCH (n)
				WHERE n.id = "123" AND n.name = "Test"
				RETURN n
				`,
			})
		})

		t.Run("Creates an expression for multiple arguemnts", func(t *testing.T) {
			c := internal.NewCypherClient(r)
			cy, err := c.
				Match(db.Node("n")).
				Where(
					"n.id = ?", "123",
				).
				Return("n").
				Compile()
			Check(t, cy, err, internal.CompiledCypher{
				Cypher: `
				MATCH (n)
				WHERE n.id = $v1
				RETURN n
				`,
				Parameters: map[string]any{
					"v1": "123",
				},
			})
		})

		t.Run("Fails when expecting all WhereOptions", func(t *testing.T) {
			c := internal.NewCypherClient(r)
			require.PanicsWithError(t, "expected all args to be ICondition, but arg 1 is string", func() {
				_, _ = c.
					Match(db.Node("n")).
					Where(db.Cond("n.id", "=", "1"), "oopsie").
					Return("n").
					Compile()
			})
		})

		t.Run("Fails when the first arguemnt is not a string or ICondition", func(t *testing.T) {
			c := internal.NewCypherClient(r)
			require.PanicsWithError(t, "expected condition to be ICondition, <key> <op> <value> or <expr> <args>", func() {
				_, _ = c.
					Match(db.Node("n")).
					Where(123).
					Return("n").
					Compile()
			})
		})
	})
}
