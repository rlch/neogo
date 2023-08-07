package neogo_test

import (
	"fmt"

	"github.com/rlch/neogo/db"
	"github.com/rlch/neogo/internal"
	"github.com/rlch/neogo/internal/tests"
)

func c() *internal.CypherClient { return internal.NewCypherClient() }

func ExampleClient_match() {
	var m tests.Movie
	c().
		Match(
			db.Node(db.Var(
				tests.Person{},
				db.Props{
					"name": "'Oliver Stone'",
				},
			)).To(nil, db.Var("movie")),
		).
		Return(db.Qual(
			&m.Title,
			"movie.title",
		)).Print()
	// Output:
	// MATCH (:Person {name: 'Oliver Stone'})-->(movie)
	// RETURN movie.title
}

func ExampleClient_optionalMatch() {
	a := tests.Person{}
	r := tests.Directed{}
	c().
		Match(
			db.Node(db.Qual(
				&a, "a",
				db.Props{
					"name": "'Martin Sheen'",
				},
			)),
		).
		OptionalMatch(
			db.Node(&a).To(db.Qual(&r, "r"), nil),
		).Return(&a.Name, &r).Print()
	// Output:
	// MATCH (a:Person {name: 'Martin Sheen'})
	// OPTIONAL MATCH (a)-[r:DIRECTED]->()
	// RETURN a.name, r
}

func ExampleClient_return() {
	var p tests.Person
	c().
		Match(db.Node(db.Qual(&p, "p", db.Props{"name": "'Keanu Reeves'"}))).
		Return(db.Qual(&p.Nationality, "citizenship")).Print()
	// Output:
	// MATCH (p:Person {name: 'Keanu Reeves'})
	// RETURN p.nationality AS citizenship
}

func ExampleClient_with() {
	var names []string
	c().
		Match(
			db.Node(db.Var("n", db.Props{"name": "'Anders'"})).
				Related(nil, "m"),
		).
		With(
			db.With("m", db.OrderBy("name", false), db.Limit("1")),
		).
		Match(db.Node("m").Related(nil, "o")).
		Return(db.Qual(names, "o.name")).Print()

	// Output:
	// MATCH (n {name: 'Anders'})--(m)
	// WITH m
	// ORDER BY m.name DESC
	// LIMIT 1
	// MATCH (m)--(o)
	// RETURN o.name
}

func ExampleClient_subquery() {
	var (
		p       Person
		numConn int
	)
	c().
		Match(db.Node(db.Qual(&p, "p"))).
		Subquery(func(c *internal.CypherClient) *internal.CypherRunner {
			return c.
				With(&p).
				Match(db.Node(&p).Related(nil, db.Var("c"))).
				Return(
					db.Qual(&numConn, "count(c)", db.Name("numberOfConnections")),
				)
		}).
		Return(&p.Name, &numConn).
		Print()

	// Output:
	// MATCH (p:Person)
	// CALL {
	//   WITH p
	//   MATCH (p)--(c)
	//   RETURN count(c) AS numberOfConnections
	// }
	// RETURN p.name, numberOfConnections
}

func ExampleClient_call() {
	var labels []string
	c().
		Call("db.labels()").
		Yield(db.Qual(&labels, "label")).
		Return(&labels).
		Print()

	// Output:
	// CALL db.labels()
	// YIELD label
	// RETURN label
}

func ExampleClient_show() {
	var (
		name any
		sig  string
	)
	c().
		Show("PROCEDURES").
		Yield(
			db.Qual(&name, "name"),
			db.Qual(&sig, "signature"),
		).
		Where(db.Cond(&name, "=", "'dbms.listConfig'")).
		Return(&sig).
		Print()

	// Output:
	// SHOW PROCEDURES
	// YIELD name, signature
	// WHERE name = 'dbms.listConfig'
	// RETURN signature
}

func ExampleClient_unwind() {
	events := map[string]any{
		"events": []map[string]any{
			{
				"id":   1,
				"year": 2014,
			},
			{
				"id":   2,
				"year": 2015,
			},
		},
	}
	type Year struct {
		internal.Node `neo4j:"Year"`

		Year int `json:"year"`
	}
	type Event struct {
		internal.Node `neo4j:"Event"`

		ID   int `json:"id"`
		Year int `json:"year"`
	}
	type In struct {
		internal.Relationship `neo4j:"IN"`
	}
	var (
		y Year
		e Event
	)
	c().
		Unwind(db.Qual(&events, "events"), "event").
		Merge(
			db.Node(db.Qual(&y, "y", db.Props{"year": "event.year"})),
		).
		Merge(
			db.Node(&y).
				From(In{}, db.Qual(&e, "e", db.Props{"id": "event.id"})),
		).
		Return(db.Return(db.Qual(&e.ID, "x"), db.OrderBy("", true))).
		Print()

	// Output:
	// UNWIND $events AS event
	// MERGE (y:Year {year: event.year})
	// MERGE (y)<-[:IN]-(e:Event {id: event.id})
	// RETURN e.id AS x
	// ORDER BY x
}

func ExampleClient_cypher() {
	var n any
	c().
		Match(db.Node(db.Qual(&n, "n"))).
		Cypher(func(scope *internal.Scope) string {
			return fmt.Sprintf("WHERE %s.name = 'Bob'", scope.Name(&n))
		}).
		Return(&n).
		Print()

	// Output:
	// MATCH (n)
	// WHERE n.name = 'Bob'
	// RETURN n
}

func ExampleClient_use() {
	var n any
	c().
		Use("myDatabase").
		Match(db.Node(db.Qual(&n, "n"))).
		Return("n").
		Print()
	// Output:
	// USE myDatabase
	// MATCH (n)
	// RETURN n
}

func ExampleClient_union() {
	var name string
	c().Union(
		func(c *internal.CypherClient) *internal.CypherRunner {
			return c.
				Match(db.Node(db.Var("n", db.Label("Person")))).
				Return(db.Qual(&name, "n.name", db.Name("name")))
		},
		func(c *internal.CypherClient) *internal.CypherRunner {
			return c.
				Match(db.Node(db.Var("n", db.Label("Movie")))).
				Return(db.Qual(&name, "n.title", db.Name("name")))
		},
	).Print()

	// Output:
	// MATCH (n:Person)
	// RETURN n.name AS name
	// UNION
	// MATCH (n:Movie)
	// RETURN n.title AS name
}

func ExampleClient_unionAll() {
	var name string
	c().UnionAll(
		func(c *internal.CypherClient) *internal.CypherRunner {
			return c.
				Match(db.Node(db.Var("n", db.Label("Person")))).
				Return(db.Qual(&name, "n.name", db.Name("name")))
		},
		func(c *internal.CypherClient) *internal.CypherRunner {
			return c.
				Match(db.Node(db.Var("n", db.Label("Movie")))).
				Return(db.Qual(&name, "n.title", db.Name("name")))
		},
	).Print()

	// Output:
	// MATCH (n:Person)
	// RETURN n.name AS name
	// UNION ALL
	// MATCH (n:Movie)
	// RETURN n.title AS name
}

func ExampleClient_yield() {
	var labels []string
	c().
		Call("db.labels()").
		Yield(db.Qual(&labels, "label")).
		Return(&labels).
		Print()

	// Output:
	// CALL db.labels()
	// YIELD label
	// RETURN label
}

func ExampleClient_create() {
	var p any
	c().
		Create(db.Path(
			db.Node(db.Var(tests.Person{}, db.Props{"name": "'Andy'"})).
				To(tests.WorksAt{}, db.Var(tests.Company{}, db.Props{"name": "'Neo4j'"})).
				From(tests.WorksAt{}, db.Var(tests.Person{}, db.Props{"name": "'Michael'"})),
			"p",
		)).
		Return(db.Qual(&p, "p")).
		Print()

	// Output:
	// CREATE p = (:Person {name: 'Andy'})-[:WORKS_AT]->(:Company {name: 'Neo4j'})<-[:WORKS_AT]-(:Person {name: 'Michael'})
	// RETURN p
}

func ExampleClient_merge() {
	var person tests.Person
	c().
		Merge(
			db.Node(db.Qual(&person, "person")),
			db.OnMatch(
				db.SetPropValue(&person.Found, true),
				db.SetPropValue(&person.LastSeen, "timestamp()"),
			),
		).
		Return(&person.Name, &person.Found, &person.LastSeen).
		Print()

	// Output:
	// MERGE (person:Person)
	// ON MATCH
	//   SET
	//     person.found = true,
	//     person.lastSeen = timestamp()
	// RETURN person.name, person.found, person.lastSeen
}

func ExampleClient_delete() {
	var (
		n tests.Person
		r tests.ActedIn
	)
	c().
		Match(
			db.Node(db.Qual(&n, "n", db.Props{"name": "'Laurence Fishburne'"})).
				To(db.Qual(&r, "r"), nil),
		).
		Delete(&r).
		Print()

	// Output:
	// MATCH (n:Person {name: 'Laurence Fishburne'})-[r:ACTED_IN]->()
	// DELETE r
}

func ExampleClient_detachDelete() {
	var n tests.Person
	c().
		Match(
			db.Node(
				db.Qual(&n, "n",
					db.Props{"name": "'Carrie-Anne Moss'"},
				),
			),
		).
		DetachDelete(&n).
		Print()

	// Output:
	// MATCH (n:Person {name: 'Carrie-Anne Moss'})
	// DETACH DELETE n
}

func ExampleClient_set() {
	var n tests.Person
	c().
		Match(
			db.Node(db.Qual(&n, "n", db.Props{"name": "'Andy'"})),
		).
		Set(
			db.SetPropValue(&n.Position, "'Developer'"),
			db.SetPropValue(&n.Surname, "'Taylor'"),
		).
		Print()

	// Output:
	// MATCH (n:Person {name: 'Andy'})
	// SET
	//   n.position = 'Developer',
	//   n.surname = 'Taylor'
}

func ExampleClient_remove() {
	var n tests.Person
	var labels []string
	c().
		Match(db.Node(db.Qual(&n, "n", db.Props{"name": "'Peter'"}))).
		Remove(db.RemoveLabels(&n, "German", "Swedish")).
		Return(&n.Name, db.Qual(&labels, "labels(n)")).
		Print()

	// Output:
	// MATCH (n:Person {name: 'Peter'})
	// REMOVE n:German:Swedish
	// RETURN n.name, labels(n)
}

func ExampleClient_forEach() {
	c().
		Match(
			db.Path(db.Node("start").To(db.Var(nil, db.Quantifier("*")), "finish"), "p"),
		).
		Where(db.And(
			db.Cond("start.name", "=", "'A'"),
			db.Cond("finish.name", "=", "'D'"),
		)).
		ForEach("n", "nodes(p)", func(c *internal.CypherUpdater[any]) {
			c.Set(db.SetPropValue("n.marked", true))
		}).
		Print()

	// Output:
	// MATCH p = (start)-[*]->(finish)
	// WHERE start.name = 'A' AND finish.name = 'D'
	// FOREACH (n IN nodes(p) | SET n.marked = true)
}

func ExampleClient_where() {
	var n tests.Person
	c().
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
		).Print()

	// Output:
	// MATCH (n:Person)
	// WHERE (n.name = 'Peter' XOR (n.age < 30 AND n.name = 'Timothy')) OR NOT (n.name = 'Timothy' OR n.name = 'Peter')
	// RETURN n.name AS name, n.age AS age
	// ORDER BY name
}

func ExampleScope() {
	var n any
	c().
		Match(db.Node(db.Qual(&n, "n"))).
		Cypher(func(scope *internal.Scope) string {
			return fmt.Sprintf("WHERE %s.name = 'Bob'", scope.Name(&n))
		}).
		Return(&n).
		Print()

	// Output:
	// MATCH (n)
	// WHERE n.name = 'Bob'
	// RETURN n
}
