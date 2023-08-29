package tests

import (
	"reflect"
	"testing"

	"github.com/rlch/neogo/db"
	"github.com/rlch/neogo/internal"
)

func TestCallSubquery(t *testing.T) {
	t.Run("Semantics", func(t *testing.T) {
		c := internal.NewCypherClient()
		var x, innerReturn any
		cy, err := c.
			Unwind(db.Qual(&x, "[0, 1, 2]"), "x").
			Subquery(func(c *internal.CypherClient) *internal.CypherRunner {
				return c.
					Return(db.Qual(&innerReturn, "'hello'", db.Name("innerReturn")))
			}).
			Return(&innerReturn).Compile()

		Check(t, cy, err, internal.CompiledCypher{
			Cypher: `
					UNWIND [0, 1, 2] AS x
					CALL {
					  RETURN 'hello' AS innerReturn
					}
					RETURN innerReturn
					`,
			Bindings: map[string]reflect.Value{
				"innerReturn": reflect.ValueOf(&innerReturn),
			},
		})

		c = internal.NewCypherClient()
		type Counter struct {
			internal.Node `neo4j:"Counter"`

			Count int `json:"count"`
		}
		var (
			n          Counter
			innerCount []int
			totalCount []int
		)
		cy, err = c.
			Unwind(db.Qual(&x, "[0, 1, 2]"), "x").
			Subquery(func(c *internal.CypherClient) *internal.CypherRunner {
				return c.
					Match(db.Node(db.Qual(&n, "n"))).
					Set(db.SetPropValue(&n.Count, "n.count + 1")).
					Return(db.Qual(db.Bind(&n.Count, &innerCount), "innerCount"))
			}).
			With(&innerCount).
			Match(db.Node(db.Qual(&n, "n"))).
			Return(
				&innerCount,
				db.Qual(db.Bind(&n.Count, &totalCount), "totalCount"),
			).Compile()

		Check(t, cy, err, internal.CompiledCypher{
			Cypher: `
					UNWIND [0, 1, 2] AS x
					CALL {
					  MATCH (n:Counter)
					  SET n.count = n.count + 1
					  RETURN n.count AS innerCount
					}
					WITH innerCount
					MATCH (n:Counter)
					RETURN innerCount, n.count AS totalCount
					`,
			Bindings: map[string]reflect.Value{
				"innerCount": reflect.ValueOf(&innerCount),
				"totalCount": reflect.ValueOf(&totalCount),
			},
		})
	})

	t.Run("Importing variables into subqueries", func(t *testing.T) {
		c := internal.NewCypherClient()
		var x, y any
		cy, err := c.
			Unwind(db.Qual(&x, "[0, 1, 2]"), "x").
			Subquery(func(c *internal.CypherClient) *internal.CypherRunner {
				return c.
					With(&x).
					Return(db.Qual(&y, "x * 10", db.Name("y")))
			}).
			Return(&x, &y).Compile()

		Check(t, cy, err, internal.CompiledCypher{
			Cypher: `
					UNWIND [0, 1, 2] AS x
					CALL {
					  WITH x
					  RETURN x * 10 AS y
					}
					RETURN x, y
					`,
			Bindings: map[string]reflect.Value{
				"x": reflect.ValueOf(&x),
				"y": reflect.ValueOf(&y),
			},
		})

		c = internal.NewCypherClient()
		var (
			person Person
			next   Person
			from   Person
		)
		cy, err = c.
			Match(db.Node(db.Qual(&person, "person"))).
			With(db.With(&person, db.OrderBy(&person.Age, true), db.Limit("1"))).
			Set(db.SetLabels(&person, "ListHead")).
			With("*").
			Match(db.Node(db.Qual(&next, "next"))).
			Where(db.Not(db.Expr("next:ListHead"))).
			With(db.With(&next, db.OrderBy(&next.Age, true))).
			Subquery(func(c *internal.CypherClient) *internal.CypherRunner {
				return c.
					With(&next).
					Match(db.Node(db.Var("current", db.Label("ListHead")))).
					Remove(db.RemoveLabels("current", "ListHead")).
					Set(db.SetLabels(&next, "ListHead")).
					Create(
						db.Node("current").To(db.Var("r", db.Label("IS_YOUNGER_THAN")), &next),
					).
					Return(
						db.Qual("current", "from"),
						db.Qual(&next, "to"),
					)
			}).
			Return(
				db.Qual(&from.Name, "from.name", db.Name("name")),
				db.Qual(&from.Age, "from.age", db.Name("age")),
				db.Qual(&next.Name, "closestOlderName"),
				db.Qual(&next.Age, "closestOlderAge"),
			).
			Compile()

			// TODO: Maybe expose db.Multiline as an option that can force multiline
			// queries
		Check(t, cy, err, internal.CompiledCypher{
			Cypher: `
					MATCH (person:Person)
					WITH person
					ORDER BY person.age
					LIMIT 1
					SET person:ListHead
					WITH *
					MATCH (next:Person)
					WHERE NOT next:ListHead
					WITH next
					ORDER BY next.age
					CALL {
					  WITH next
					  MATCH (current:ListHead)
					  REMOVE current:ListHead
					  SET next:ListHead
					  CREATE (current)-[r:IS_YOUNGER_THAN]->(next)
					  RETURN current AS from, next AS to
					}
					RETURN from.name AS name, from.age AS age, to.name AS closestOlderName, to.age AS closestOlderAge
					`,
			Bindings: map[string]reflect.Value{
				"name":             reflect.ValueOf(&from.Name),
				"age":              reflect.ValueOf(&from.Age),
				"closestOlderName": reflect.ValueOf(&next.Name),
				"closestOlderAge":  reflect.ValueOf(&next.Age),
			},
		})
	})

	t.Run("Post-union processing", func(t *testing.T) {
		c := internal.NewCypherClient()
		var p Person
		cy, err := c.
			Subquery(func(c *internal.CypherClient) *internal.CypherRunner {
				return c.Union(
					func(c *internal.CypherClient) *internal.CypherRunner {
						return c.
							Match(db.Node(db.Qual(&p, "p"))).
							Return(
								db.Return(&p, db.OrderBy(&p.Age, true), db.Limit("1")),
							)
					},
					func(c *internal.CypherClient) *internal.CypherRunner {
						return c.
							Match(db.Node(db.Qual(&p, "p"))).
							Return(
								db.Return(&p, db.OrderBy(&p.Age, false), db.Limit("1")),
							)
					},
				).CypherRunner
			}).
			Return(db.Return(&p.Name, db.OrderBy("", true)), &p.Age).
			Compile()

			// TODO: Maybe expose db.Multiline as an option that can force multiline
			// queries
		Check(t, cy, err, internal.CompiledCypher{
			Cypher: `
					CALL {
					  MATCH (p:Person)
					  RETURN p
					  ORDER BY p.age
					  LIMIT 1
					UNION
					  MATCH (p:Person)
					  RETURN p
					  ORDER BY p.age DESC
					  LIMIT 1
					}
					RETURN p.name, p.age
					ORDER BY p.name
					`,
			Bindings: map[string]reflect.Value{
				"p.name": reflect.ValueOf(&p.Name),
				"p.age":  reflect.ValueOf(&p.Age),
			},
		})

		var (
			other  any
			cOther any
		)
		c = internal.NewCypherClient()
		cy, err = c.
			Match(db.Node(db.Qual(&p, "p"))).
			Subquery(func(c *internal.CypherClient) *internal.CypherRunner {
				return c.Union(
					func(c *internal.CypherClient) *internal.CypherRunner {
						return c.
							With(&p).
							OptionalMatch(
								db.Node(&p).
									To(
										db.Var(nil, db.Label("FRIEND_OF")),
										db.Qual(&other, "other", db.Label("Person")),
									),
							).
							Return(&other)
					},
					func(c *internal.CypherClient) *internal.CypherRunner {
						return c.
							With(&p).
							OptionalMatch(
								db.Node(&p).
									To(
										db.Var(nil, db.Label("CHILD_OF")),
										db.Qual(&other, "other", db.Label("Parent")),
									),
							).
							Return(&other)
					},
				).CypherRunner
			}).
			Return(db.Return(&p.Name, db.Distinct), db.Qual(&cOther, "count(other)")).
			Compile()

			// TODO: Maybe expose db.Multiline as an option that can force multiline
			// queries
		Check(t, cy, err, internal.CompiledCypher{
			Cypher: `
					MATCH (p:Person)
					CALL {
					  WITH p
					  OPTIONAL MATCH (p)-[:FRIEND_OF]->(other:Person)
					  RETURN other
					UNION
					  WITH p
					  OPTIONAL MATCH (p)-[:CHILD_OF]->(other:Parent)
					  RETURN other
					}
					RETURN DISTINCT p.name, count(other)
					`,
			Bindings: map[string]reflect.Value{
				"p.name":       reflect.ValueOf(&p.Name),
				"count(other)": reflect.ValueOf(&cOther),
			},
		})
	})

	t.Run("Aggregations", func(t *testing.T) {
		c := internal.NewCypherClient()
		var (
			p       Person
			numConn int
		)
		cy, err := c.
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
			Compile()

			// TODO: Maybe expose db.Multiline as an option that can force multiline
			// queries
		Check(t, cy, err, internal.CompiledCypher{
			Cypher: `
					MATCH (p:Person)
					CALL {
					  WITH p
					  MATCH (p)--(c)
					  RETURN count(c) AS numberOfConnections
					}
					RETURN p.name, numberOfConnections
					`,
			Bindings: map[string]reflect.Value{
				"p.name":              reflect.ValueOf(&p.Name),
				"numberOfConnections": reflect.ValueOf(&numConn),
			},
		})
	})

	t.Run("Unit subqueries and side-effects", func(t *testing.T) {
		c := internal.NewCypherClient()
		var (
			p     Person
			count int
		)
		cy, err := c.
			Match(db.Node(db.Qual(&p, "p"))).
			Subquery(func(c *internal.CypherClient) *internal.CypherRunner {
				return c.
					With(&p).
					Unwind("range (1, 5)", "i").
					Create(
						db.Node(db.Var(Person{}, db.Props{"name": &p.Name})),
					).CypherRunner
			}).
			Return(db.Qual(&count, "count(*)")).
			Compile()

			// TODO: Maybe expose db.Multiline as an option that can force multiline
			// queries
		Check(t, cy, err, internal.CompiledCypher{
			Cypher: `
					MATCH (p:Person)
					CALL {
					  WITH p
					  UNWIND range (1, 5) AS i
					  CREATE (:Person {name: p.name})
					}
					RETURN count(*)
					`,
			Bindings: map[string]reflect.Value{
				"count(*)": reflect.ValueOf(&count),
			},
		})
	})

	t.Run("Aggregation on imported variables", func(t *testing.T) {
		c := internal.NewCypherClient()
		var (
			p     Person
			other Person
			count int
		)
		cy, err := c.
			Match(db.Node(db.Qual(&p, "p"))).
			Subquery(func(c *internal.CypherClient) *internal.CypherRunner {
				return c.
					With(&p).
					Match(db.Node(db.Qual(&other, "other"))).
					Where(db.Cond(&other.Age, "<", &p.Age)).
					Return(db.Qual(&count, "count(other)", db.Name("youngerPersonsCount")))
			}).
			Return(&p.Name, &count).
			Compile()

			// TODO: Maybe expose db.Multiline as an option that can force multiline
			// queries
		Check(t, cy, err, internal.CompiledCypher{
			Cypher: `
					MATCH (p:Person)
					CALL {
					  WITH p
					  MATCH (other:Person)
					  WHERE other.age < p.age
					  RETURN count(other) AS youngerPersonsCount
					}
					RETURN p.name, youngerPersonsCount
					`,
			Bindings: map[string]reflect.Value{
				"youngerPersonsCount": reflect.ValueOf(&count),
				"p.name":              reflect.ValueOf(&p.Name),
			},
		})
	})

	t.Run("Subqueries in transactions", func(t *testing.T) {
		// TODO: is this possible with the driver? Skipping for now
	})

	t.Run("Variable collisions are avoided", func(t *testing.T) {
		c := internal.NewCypherClient()
		v1 := db.Var([]int{1, 2, 3})
		v2 := db.Var([]int{4, 5, 6})
		v3 := db.Var([]int{7, 8, 9})
		cy, err := c.
			With(&v1).
			Subquery(func(c *internal.CypherClient) *internal.CypherRunner {
				return c.With(&v1).Return(&v2)
			}).
			With(&v1, &v2).
			Subquery(func(c *internal.CypherClient) *internal.CypherRunner {
				return c.With(&v1, &v2).Return(&v3)
			}).
			Return(&v1, &v2, &v3).Compile()

		Check(t, cy, err, internal.CompiledCypher{
			Cypher: `
			WITH $ptr AS ptr
			CALL {
			  WITH ptr
			  RETURN $ptr1 AS ptr1
			}
			WITH ptr, ptr1
			CALL {
			  WITH ptr, ptr1
			  RETURN $ptr2 AS ptr2
			}
			RETURN ptr, ptr1, ptr2
					`,
			Bindings: map[string]reflect.Value{
				"ptr":  reflect.ValueOf(&v1),
				"ptr1": reflect.ValueOf(&v2),
				"ptr2": reflect.ValueOf(&v3),
			},
			Parameters: map[string]any{
				"ptr":  &v1,
				"ptr1": &v2,
				"ptr2": &v3,
			},
		})
	})

	t.Run("Parameter collisions are avoided", func(t *testing.T) {
		c := internal.NewCypherClient()
		cy, err := c.
			With(db.Param(nil)).
			Subquery(func(c *internal.CypherClient) *internal.CypherRunner {
				// NOTE: In the public API runner will be returned as an interface.
				return c.With(db.Param(nil)).CypherRunner
			}).
			Subquery(func(c *internal.CypherClient) *internal.CypherRunner {
				return c.With(db.Param(nil)).CypherRunner
			}).
			With(db.Param(nil)).
			Return(db.Param(nil)).Compile()

		Check(t, cy, err, internal.CompiledCypher{
			Cypher: `
			WITH $v1
			CALL {
			  WITH $v2
			}
			CALL {
			  WITH $v3
			}
			WITH $v4
			RETURN $v5
					`,
			Parameters: map[string]any{
				"v1": nil,
				"v2": nil,
				"v3": nil,
				"v4": nil,
				"v5": nil,
			},
		})
	})
}

func TestCallProcedure(t *testing.T) {
	t.Run("Call a procedure using CALL", func(t *testing.T) {
		var labels []string
		c := internal.NewCypherClient()
		cy, err := c.
			Call("db.labels()").
			Yield(db.Qual(&labels, "label")).
			Return(&labels).
			Compile()

		Check(t, cy, err, internal.CompiledCypher{
			Cypher: `
			CALL db.labels()
			YIELD label
			RETURN label
					`,
			Bindings: map[string]reflect.Value{
				"label": reflect.ValueOf(&labels),
			},
		})
	})

	t.Run("View the signature for a procedure", func(t *testing.T) {
		var (
			name any
			sig  string
		)
		c := internal.NewCypherClient()
		cy, err := c.
			Show("PROCEDURES").
			Yield(
				db.Qual(&name, "name"),
				db.Qual(&sig, "signature"),
			).
			Where(db.Cond(&name, "=", "'dbms.listConfig'")).
			Return(&sig).
			Compile()

		Check(t, cy, err, internal.CompiledCypher{
			Cypher: `
					SHOW PROCEDURES
					YIELD name, signature
					WHERE name = 'dbms.listConfig'
					RETURN signature
					`,
			Bindings: map[string]reflect.Value{
				"signature": reflect.ValueOf(&sig),
			},
		})
	})

	t.Run("Call a procedure within a complex query and rename its outputs", func(t *testing.T) {
		var (
			prop     any
			numNodes int
		)
		c := internal.NewCypherClient()
		cy, err := c.
			Call("db.propertyKeys()").
			Yield(
				db.Qual(&prop, "propertyKey", db.Name("prop")),
			).
			Match(db.Node("n")).
			Where(db.Cond("n[prop]", "IS NOT", "NULL")).
			Return(&prop, db.Qual(&numNodes, "count(n)", db.Name("numNodes"))).
			Compile()

			// ignore, just a bad tree-sitter query triggering a false-positive for
			// sql lol
		isNot := "IS NOT"
		Check(t, cy, err, internal.CompiledCypher{
			Cypher: `
					CALL db.propertyKeys()
					YIELD propertyKey AS prop
					MATCH (n)
					WHERE n[prop] ` + isNot + ` NULL
					RETURN prop, count(n) AS numNodes
					`,
			Bindings: map[string]reflect.Value{
				"prop":     reflect.ValueOf(&prop),
				"numNodes": reflect.ValueOf(&numNodes),
			},
		})
	})
}
