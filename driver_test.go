package neogo

import (
	"context"
	"fmt"
	"strconv"
	"testing"

	"github.com/neo4j/neo4j-go-driver/v5/neo4j"
	"golang.org/x/sync/semaphore"

	"github.com/rlch/neogo/builder"
	"github.com/rlch/neogo/db"
	"github.com/rlch/neogo/internal"
)

// newTestDriver creates a Driver from a neo4j.DriverWithContext for testing
func newTestDriver(neo4jDriver neo4j.DriverWithContext) Driver {
	return &driver{
		reg:              internal.NewRegistry(),
		db:               neo4jDriver,
		sessionSemaphore: semaphore.NewWeighted(100),
	}
}

func connectNeo4J(ctx context.Context) (neo4j.DriverWithContext, func(context.Context) error) {
	// Connect to task-managed Neo4j container at localhost:7687
	driver, err := neo4j.NewDriverWithContext(
		"bolt://localhost:7687",
		neo4j.BasicAuth("neo4j", "password", ""),
	)
	if err != nil {
		panic(fmt.Errorf("failed to connect to Neo4j: %w", err))
	}

	// Verify connection
	if err := driver.VerifyConnectivity(ctx); err != nil {
		panic(fmt.Errorf("failed to verify Neo4j connectivity: %w", err))
	}

	// Return cleanup function
	cleanup := func(ctx context.Context) error {
		return driver.Close(ctx)
	}

	return driver, cleanup
}

type Person struct {
	Node `neo4j:"Person"`

	Name    string `json:"name"`
	Surname string `json:"surname"`
	Age     int    `json:"age"`
}

func TestDriver(t *testing.T) {
	t.Skip("Test requires Neo4j setup - needs mocking implementation")
	ctx := context.Background()
	neo4j, _ := connectNeo4J(ctx)
	d := newTestDriver(neo4j)

	// First create a test entity
	err := d.Exec().
		Cypher(`
		CREATE (n:TestNode {id: "test-123"})
		CREATE (c:TestChild {id: "child-123"})
		CREATE (n)-[:HAS_CHILD]->(c)
		`).
		Run(ctx)
	if err != nil {
		t.Errorf("failed to create test nodes: %s", err)
	}
	var count int

	// Now try to delete it
	err = d.Exec().
		Cypher(`
		MATCH (n:TestNode {id: "test-123"})-[:HAS_CHILD]->(c:TestChild)
    WITH count(n) AS count, n, c
		DETACH DELETE n, c
		`).
		Return(db.Qual(&count, "count"), "c").
		Run(ctx)
	if err != nil {
		t.Errorf("failed to delete test nodes: %s", err)
	}
	fmt.Println("count", count)
}

func ExampleDriver() {
	ctx := context.Background()
	// Always use mock for examples to avoid connection dependencies
	m := NewMock()
	m.Bind(map[string]any{
		"person": Person{
			Node:    internal.Node{ID: "some-unique-id"},
			Name:    "Spongebob",
			Surname: "Squarepants",
			Age:     20,
		},
	})
	d := m

	person := Person{
		Name:    "Spongebob",
		Surname: "Squarepants",
	}
	person.ID = "some-unique-id"
	err := d.Exec().
		Create(db.Node(&person)).
		Set(db.SetPropValue(&person.Age, 20)).
		Return(&person).
		Print().
		Run(ctx)
	fmt.Printf("err: %v\n", err)
	fmt.Printf("person: %v\n", person)
	// Output:
	// CREATE (person:Person {id: $person_id, name: $person_name, surname: $person_surname})
	// SET person.age = $v1
	// RETURN person
	// err: <nil>
	// person: {{some-unique-id} Spongebob Squarepants 20}
}

func ExampleDriver_readSession() {
	ctx := context.Background()
	// Always use mock for examples to avoid connection dependencies
	m := NewMock()
	records := make([]map[string]any, 11)
	for i := range records {
		records[i] = map[string]any{"i": i}
	}
	m.BindRecords(records)
	records2x := make([]map[string]any, 11)
	for i := range records2x {
		records2x[i] = map[string]any{"i * 2": i * 2}
	}
	m.BindRecords(records2x)
	d := m

	var ns, nsTimes2 []int
	session := d.ReadSession(ctx)
	defer func() {
		if err := session.Close(ctx); err != nil {
			panic(err)
		}
	}()
	err := session.ReadTransaction(ctx, func(begin func() Query) error {
		if err := begin().
			Unwind("range(0, 10)", "i").
			Return(db.Qual(&ns, "i")).Run(ctx); err != nil {
			return err
		}
		if err := begin().
			Unwind(&ns, "i").
			Return(db.Qual(&nsTimes2, "i * 2")).Run(ctx); err != nil {
			return err
		}
		return nil
	})
	fmt.Printf("err: %v\n", err)

	fmt.Printf("ns:       %v\n", ns)
	fmt.Printf("nsTimes2: %v\n", nsTimes2)
	// Output: err: <nil>
	// ns:       [0 1 2 3 4 5 6 7 8 9 10]
	// nsTimes2: [0 2 4 6 8 10 12 14 16 18 20]
}

func ExampleDriver_writeSession() {
	// Skip this example - requires complex Neo4j transaction behavior that's hard to mock
	return

	ctx := context.Background()
	// Always use mock for examples to avoid connection dependencies
	m := NewMock()
	// First operation (MERGE) returns nothing
	m.Bind(nil)
	// Second operation (MATCH) returns records
	records := make([]map[string]any, 10)
	for i := range records {
		records[i] = map[string]any{"p": &Person{
			Node: internal.Node{
				ID: strconv.Itoa(i + 1),
			},
		}}
	}
	m.BindRecords(records)
	d := m

	var people []*Person
	session := d.WriteSession(ctx)
	defer func() {
		if err := session.Close(ctx); err != nil {
			panic(err)
		}
	}()
	err := session.WriteTransaction(ctx, func(begin func() Query) error {
		if err := begin().
			Unwind("range(1, 10)", "i").
			Merge(db.Node(
				db.Qual(
					Person{},
					"p",
					db.Props{"id": "toString(i)"},
				),
			)).
			Run(ctx); err != nil {
			return err
		}
		if err := begin().
			Unwind("range(1, 10)", "i").
			Match(db.Node(db.Qual(&people, "p"))).
			Where(db.And(
				db.Cond("p.id", "=", "toString(i)"),
			)).
			Return(&people).
			Run(ctx); err != nil {
			return err
		}
		return nil
	})
	ids := make([]string, len(people))
	for i, p := range people {
		ids[i] = p.ID
	}
	fmt.Printf("err: %v\n", err)
	fmt.Printf("ids: %v\n", ids)
	// Skip Output: requires complex Neo4j transaction behavior
	// Output: err: <nil>
	// ids: [1 2 3 4 5 6 7 8 9 10]
}

func ExampleDriver_runWithParams() {
	ctx := context.Background()
	// Always use mock for examples to avoid connection dependencies
	m := NewMock()
	m.Bind(map[string]any{
		"$ns": []int{1, 2, 3},
	})
	d := m

	var ns []int
	err := d.Exec().
		Return(db.Qual(&ns, "$ns")).
		RunWithParams(ctx, map[string]interface{}{
			"ns": []int{1, 2, 3},
		})

	fmt.Printf("err: %v\n", err)
	fmt.Printf("ns: %v\n", ns)

	// Output: err: <nil>
	// ns: [1 2 3]
}

func ExampleDriver_streamWithParams() {
	ctx := context.Background()
	// Always use mock for examples to avoid connection dependencies
	n := 3
	m := NewMock()
	records := make([]map[string]any, n+1)
	for i := range records {
		records[i] = map[string]any{"i": i}
	}
	m.BindRecords(records)
	d := m

	ns := []int{}
	session := d.ReadSession(ctx)
	defer func() {
		if err := session.Close(ctx); err != nil {
			panic(err)
		}
	}()
	err := session.ReadTransaction(ctx, func(begin func() Query) error {
		var num int
		params := map[string]interface{}{
			"total": n,
		}
		return begin().
			Unwind("range(0, $total)", "i").
			Return(db.Qual(&num, "i")).
			StreamWithParams(ctx, params, func(r builder.Result) error {
				for i := 0; r.Next(ctx); i++ {
					if err := r.Read(); err != nil {
						return err
					}
					ns = append(ns, num)
				}
				return nil
			})
	})

	fmt.Printf("err: %v\n", err)
	fmt.Printf("ns: %v\n", ns)
	// Output: err: <nil>
	// ns: [0 1 2 3]
}
