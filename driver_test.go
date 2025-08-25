package neogo

import (
	"context"
	"fmt"
	"strconv"
	"testing"
	"time"

	"github.com/neo4j/neo4j-go-driver/v5/neo4j"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/rlch/neogo/db"
	"github.com/rlch/neogo/internal"
	"github.com/rlch/neogo/query"
)

func startNeo4J(ctx context.Context) (string, func(context.Context) error) {
	request := testcontainers.ContainerRequest{
		Name:         "neo4j",
		Image:        "neo4j:5.7-enterprise",
		ExposedPorts: []string{"7687/tcp"},
		WaitingFor:   wait.ForLog("Bolt enabled").WithStartupTimeout(time.Minute * 2),
		Env: map[string]string{
			"NEO4J_AUTH":                     fmt.Sprintf("%s/%s", "neo4j", "password"),
			"NEO4J_PLUGINS":                  `["apoc"]`,
			"NEO4J_ACCEPT_LICENSE_AGREEMENT": "yes",
		},
	}
	container, err := testcontainers.GenericContainer(
		ctx, testcontainers.GenericContainerRequest{
			ContainerRequest: request,
			Started:          true,
			Reuse:            true,
		})
	if err != nil {
		panic(fmt.Errorf("container should start: %w", err))
	}

	port, err := container.MappedPort(ctx, "7687")
	if err != nil {
		panic(err)
	}
	uri := fmt.Sprintf("bolt://localhost:%d", port.Int())
	return uri, container.Terminate
}

type Person struct {
	Node `neo4j:"Person"`

	Name    string `json:"name"`
	Surname string `json:"surname"`
	Age     int    `json:"age"`
}

func TestDriver(t *testing.T) {
	ctx := context.Background()
	uri, cancel := startNeo4J(ctx)
	t.Cleanup(func() {
		if err := cancel(ctx); err != nil {
			t.Logf("error canceling container: %v", err)
		}
	})
	d, err := New(uri, neo4j.BasicAuth("neo4j", "password", ""))
	if err != nil {
		t.Fatalf("failed to create driver: %v", err)
	}

	// First create a test entity
	err = d.Exec().
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
		Return(db.Qual(&count, "count")).
		Run(ctx)
	if err != nil {
		t.Errorf("failed to delete test nodes: %s", err)
	}
	fmt.Println("count", count)
}

func ExampleDriver() {
	ctx := context.Background()
	var d Driver
	if testing.Short() {
		m := NewMock()
		m.Bind(map[string]any{
			"person": Person{
				Node:    internal.Node{ID: "some-unique-id"},
				Name:    "Spongebob",
				Surname: "Squarepants",
				Age:     20,
			},
		})
		d = m
	} else {
		uri, cancel := startNeo4J(ctx)
		var err error
		d, err = New(uri, neo4j.BasicAuth("neo4j", "password", ""))
		if err != nil {
			panic(err)
		}
		defer func() {
			if err := cancel(ctx); err != nil {
				panic(err)
			}
		}()
	}

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
	var d Driver

	if testing.Short() {
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
		d = m
	} else {
		uri, cancel := startNeo4J(ctx)
		var err error
		d, err = New(uri, neo4j.BasicAuth("neo4j", "password", ""))
		if err != nil {
			panic(err)
		}
		defer func() {
			if err := cancel(ctx); err != nil {
				panic(err)
			}
		}()
	}

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
	ctx := context.Background()
	var d Driver
	if testing.Short() {
		m := NewMock()
		m.Bind(nil)
		records := make([]map[string]any, 10)
		for i := range records {
			records[i] = map[string]any{"p": &Person{
				Node: internal.Node{
					ID: strconv.Itoa(i + 1),
				},
			}}
		}
		m.BindRecords(records)
		d = m
	} else {
		uri, cancel := startNeo4J(ctx)
		var err error
		d, err = New(uri, neo4j.BasicAuth("neo4j", "password", ""))
		if err != nil {
			panic(err)
		}
		defer func() {
			if err := cancel(ctx); err != nil {
				panic(err)
			}
		}()
	}

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
	// Output: err: <nil>
	// ids: [1 2 3 4 5 6 7 8 9 10]
}

func ExampleDriver_runWithParams() {
	ctx := context.Background()
	var d Driver
	if testing.Short() {
		m := NewMock()
		m.Bind(map[string]any{
			"$ns": []int{1, 2, 3},
		})
		d = m
	} else {
		uri, cancel := startNeo4J(ctx)
		var err error
		d, err = New(uri, neo4j.BasicAuth("neo4j", "password", ""))
		if err != nil {
			panic(err)
		}
		defer func() {
			if err := cancel(ctx); err != nil {
				panic(err)
			}
		}()
	}

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
	var d Driver
	n := 3

	if testing.Short() {
		m := NewMock()
		records := make([]map[string]any, n+1)
		for i := range records {
			records[i] = map[string]any{"i": i}
		}
		m.BindRecords(records)
		d = m
	} else {
		uri, cancel := startNeo4J(ctx)
		var err error
		d, err = New(uri, neo4j.BasicAuth("neo4j", "password", ""))
		if err != nil {
			panic(err)
		}
		defer func() {
			if err := cancel(ctx); err != nil {
				panic(err)
			}
		}()
	}

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
			StreamWithParams(ctx, params, func(r query.Result) error {
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

func TestConfigOverride(t *testing.T) {
	ctx := context.Background()
	uri, cancel := startNeo4J(ctx)
	t.Cleanup(func() {
		if err := cancel(ctx); err != nil {
			t.Logf("error canceling container: %v", err)
		}
	})

	t.Run("default config values", func(t *testing.T) {
		d, err := New(uri, neo4j.BasicAuth("neo4j", "password", ""))
		require.NoError(t, err)
		
		// Access the underlying neo4j driver to verify default config
		neo4jDriver := d.DB()
		require.NotNil(t, neo4jDriver)
		
		// We can't directly access the config from the driver, but we can test 
		// that the driver was created successfully with defaults
		assert.NotNil(t, neo4jDriver)
	})

	t.Run("custom config values", func(t *testing.T) {
		customTimeout := 10 * time.Second
		customPoolSize := 50
		
		d, err := New(uri, neo4j.BasicAuth("neo4j", "password", ""), func(cfg *Config) {
			cfg.MaxTransactionRetryTime = customTimeout
			cfg.MaxConnectionPoolSize = customPoolSize
		})
		require.NoError(t, err)
		
		// Access the underlying neo4j driver
		neo4jDriver := d.DB()
		require.NotNil(t, neo4jDriver)
		
		// Test that the driver works with custom config
		err = d.Exec().
			Cypher("RETURN 1 as test").
			Run(ctx)
		assert.NoError(t, err)
	})

	t.Run("causal consistency config", func(t *testing.T) {
		keyFunc := func(ctx context.Context) string {
			return "test-key"
		}
		
		d, err := New(uri, neo4j.BasicAuth("neo4j", "password", ""), WithCausalConsistency(keyFunc))
		require.NoError(t, err)
		
		// Test that the driver works with causal consistency
		err = d.Exec().
			Cypher("RETURN 1 as test").
			Run(ctx)
		assert.NoError(t, err)
	})

	t.Run("multiple configurers", func(t *testing.T) {
		d, err := New(uri, neo4j.BasicAuth("neo4j", "password", ""), 
			func(cfg *Config) {
				cfg.MaxConnectionPoolSize = 25
			},
			func(cfg *Config) {
				cfg.MaxTransactionRetryTime = 15 * time.Second
			},
			WithCausalConsistency(func(ctx context.Context) string {
				return "multi-config-key"
			}),
		)
		require.NoError(t, err)
		
		// Test that the driver works with multiple configs
		err = d.Exec().
			Cypher("RETURN 1 as test").
			Run(ctx)
		assert.NoError(t, err)
	})
}