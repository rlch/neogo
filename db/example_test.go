package db

import (
	"github.com/rlch/neogo/internal"
	"github.com/rlch/neogo/internal/tests"
)

func c() *internal.CypherClient { return internal.NewCypherClient() }

func Example() {
	c().
		Return("1 + 2").
		Print()
	// Output:
	// RETURN 1 + 2
}

func ExampleString() {
	c().
		Return(String("hello")).
		Print()
	// Output:
	// RETURN "hello"
}

func ExampleParam() {
	c().
		Return(Param(123)).
		Print()
	// Output:
	// RETURN $v1
}

func ExampleNamedParam() {
	c().
		Return(NamedParam(123, "n")).
		Print()
	// Output:
	// RETURN $n
}

func ExamplePattern() {
	c().
		Match(Node("p").To("r", "c")).
		Print()
	// Output:
	// MATCH (p)-[r]->(c)
}

func ExampleNode() {
	c().
		Match(Node("n")).
		Print()
	// Output:
	// MATCH (n)
}

func ExamplePath() {
	c().
		Match(Path(Node("n").Related("r", "m"), "p")).
		Print()
	// Output:
	// MATCH p = (n)-[r]-(m)
}

func ExamplePatterns() {
	c().
		Match(Patterns(
			Node("n").To("e", "m"),
			Node(nil).From("e", "n"),
		)).
		Print()
	// Output:
	// MATCH
	//   (n)-[e]->(m),
	//   ()<-[e]-(n)
}

func ExampleWith() {
	c().
		With(With("n", OrderBy("name", false))).
		Print()
	// Output:
	// WITH n
	// ORDER BY n.name DESC
}

func ExampleReturn() {
	c().
		Return(Return("n", OrderBy("name", false))).
		Print()
	// Output:
	// RETURN n
	// ORDER BY n.name DESC
}

func ExampleOrderBy() {
	c().
		Return(Return("n", OrderBy("name", false))).
		Print()
	// Output:
	// RETURN n
	// ORDER BY n.name DESC
}

func ExampleSkip() {
	c().
		With(With("n", Skip("2"))).
		Print()
	// Output:
	// WITH n
	// SKIP 2
}

func ExampleLimit() {
	c().
		With(With("n", Limit("2"))).
		Print()
	// Output:
	// WITH n
	// LIMIT 2
}

func ExampleDistinct() {
	c().
		Return(Return("n", Distinct)).
		Print()
	// Output:
	// RETURN DISTINCT n
}

func ExampleSetPropValue() {
	c().
		Set(SetPropValue("n.name", String("John"))).
		Print()
	// Output:
	// SET n.name = "John"
}

func ExampleSetMerge() {
	c().
		Set(SetMerge("n", "{x: 2}")).
		Print()
	// Output:
	// SET n += {x: 2}
}

func ExampleSetLabels() {
	c().
		Set(SetLabels("n", "Person", "Employee")).
		Print()
	// Output:
	// SET n:Person:Employee
}

func ExampleRemoveProp() {
	c().
		Remove(RemoveProp("n.name")).
		Print()
	// Output:
	// REMOVE n.name
}

func ExampleRemoveLabels() {
	c().
		Remove(RemoveLabels("n", "Person", "Employee")).
		Print()
	// Output:
	// REMOVE n:Person:Employee
}

func ExampleOnCreate() {
	c().
		Merge(
			Node("n"),
			OnCreate(SetPropValue("n.created", "true")),
		).Print()
	// Output:
	// MERGE (n)
	// ON CREATE
	//   SET n.created = true
}

func ExampleOnMatch() {
	c().
		Merge(
			Node("n"),
			OnMatch(SetPropValue("n.found", "true")),
		).Print()
	// Output:
	// MERGE (n)
	// ON MATCH
	//   SET n.found = true
}

func ExampleVar() {
	c().
		With(Var("n")).
		Print()
	// Output:
	// WITH n
}

func ExampleQual() {
	c().
		With(Qual("timestamp()", "n")).
		Print()
	// Output:
	// WITH timestamp() AS n
}

func ExampleBind() {
	var from, to any
	c().
		Match(Node(Qual(&from, "from"))).
		With(Qual(Bind(&from, &to), "to")).
		Return(&to).
		Print()
	// Output:
	// MATCH (from)
	// WITH from AS to
	// RETURN to
}

func ExampleName() {
	c().
		With(Var("n", Name("m"))).
		Print()
	// Output:
	// WITH n AS m
}

func ExampleLabel() {
	c().
		Match(Node(Var("n", Label("Person:Child")))).
		Print()
	// Output:
	// MATCH (n:Person:Child)
}

func ExampleVarLength() {
	c().
		Match(Node(nil).Related(Var("r", VarLength("*..")), "n")).
		Print()
	// Output:
	// MATCH ()-[r*..]-(n)
}

func ExampleProps() {
	var p tests.Person
	c().
		Match(
			Patterns(
				Node(Qual(&p, "p")),
				Node(Var(nil, Props{
					"name": &p.Name,
				})),
			),
		).
		Print()
	// Output:
	// MATCH
	//   (p:Person),
	//   ({name: p.name})
}

func ExamplePropsExpr() {
	var p tests.Person
	c().
		Create(
			Node(Qual(&p, "p", PropsExpr("$someVar"))).
				To(Var("e", PropsExpr("$anotherVar")), nil),
		).
		Print()
	// Output:
	// CREATE (p:Person $someVar)-[e $anotherVar]->()
}

func ExampleWhere() {
	c().
		Match(Node("n")).
		Where(Cond("n.name", "=", String("Alice"))).
		Print()
	// Output:
	// MATCH (n)
	// WHERE n.name = "Alice"
}

func ExampleOr() {
	c().
		Match(Node("n")).
		Where(Or(
			Cond("n.name", "=", String("Alice")),
			Cond("n.name", "=", String("Bob")),
		)).
		Print()
	// Output:
	// MATCH (n)
	// WHERE n.name = "Alice" OR n.name = "Bob"
}

func ExampleAnd() {
	c().
		Match(Node("n")).
		Where(And(
			Cond("n.age", ">", "25"),
			Cond("n.city", "=", String("New York")),
		)).
		Print()
	// Output:
	// MATCH (n)
	// WHERE n.age > 25 AND n.city = "New York"
}

func ExampleXor() {
	c().
		Match(Node("n")).
		Where(Xor(
			Cond("n.status", "=", String("active")),
			Cond("n.status", "=", String("inactive")),
		)).
		Print()
	// Output:
	// MATCH (n)
	// WHERE n.status = "active" XOR n.status = "inactive"
}

func ExampleNot() {
	c().
		Match(Node("n")).
		Where(Not(Cond("n.isBlocked", "=", true))).
		Print()
	// Output:
	// MATCH (n)
	// WHERE NOT n.isBlocked = true
}
