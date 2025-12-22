package neogo

import (
	"context"
	"fmt"
	"testing"

	"github.com/neo4j/neo4j-go-driver/v5/neo4j"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/rlch/neogo/internal"
)

func TestDriver(t *testing.T) {
	ctx := context.Background()
	nc := startNeo4jContainer(ctx, t)
	defer nc.Close(ctx, t)

	d := nc.NewTestDriver(t)

	// First create test entities
	err := d.Exec().
		Cypher(`
		CREATE (n:TestNode {id: "test-123"})
		CREATE (c:TestChild {id: "child-123"})
		CREATE (n)-[:HAS_CHILD]->(c)
		`).
		Run(ctx)
	require.NoError(t, err, "failed to create test nodes")

	var count int64

	// Now try to delete and return count
	err = d.Exec().
		Cypher(`
		MATCH (n:TestNode {id: "test-123"})-[:HAS_CHILD]->(c:TestChild)
		WITH count(n) AS cnt, n, c
		DETACH DELETE n, c
		RETURN cnt
		`).
		Run(ctx, "cnt", &count)
	require.NoError(t, err, "failed to delete test nodes")
	assert.Equal(t, int64(1), count)
}

func ExampleDriver() {
	ctx := context.Background()
	// Always use mock for examples to avoid connection dependencies
	m := NewMock()
	m.Bind(map[string]any{
		"person": neo4j.Node{
			Props: map[string]any{
				"id":      "some-unique-id",
				"name":    "Spongebob",
				"surname": "Squarepants",
				"age":     int64(20),
			},
		},
	})
	d := m

	type Person struct {
		internal.Node `neo4j:"Person"`
		Name          string `neo4j:"name"`
		Surname       string `neo4j:"surname"`
		Age           int64  `neo4j:"age"`
	}

	var person Person
	err := d.Exec().
		Cypher(`
			CREATE (person:Person {id: $id, name: $name, surname: $surname})
			SET person.age = $age
			RETURN person
		`).
		RunWithParams(ctx, map[string]any{
			"id":      "some-unique-id",
			"name":    "Spongebob",
			"surname": "Squarepants",
			"age":     int64(20),
		}, "person", &person)
	fmt.Printf("err: %v\n", err)
	fmt.Printf("person: %s %s, age %d\n", person.Name, person.Surname, person.Age)
	// Output:
	// err: <nil>
	// person: Spongebob Squarepants, age 20
}

func ExampleDriver_readSession() {
	ctx := context.Background()
	// Always use mock for examples to avoid connection dependencies
	m := NewMock()
	records := make([]map[string]any, 11)
	for i := range records {
		records[i] = map[string]any{"i": int64(i)}
	}
	m.BindRecords(records)
	records2x := make([]map[string]any, 11)
	for i := range records2x {
		records2x[i] = map[string]any{"i2": int64(i * 2)}
	}
	m.BindRecords(records2x)
	d := m

	var ns, nsTimes2 []int64
	session := d.ReadSession(ctx)
	defer func() {
		if err := session.Close(ctx); err != nil {
			panic(err)
		}
	}()
	err := session.ReadTransaction(ctx, func(c Client) error {
		if err := c.
			Cypher("UNWIND range(0, 10) AS i RETURN i").
			Run(ctx, "i", &ns); err != nil {
			return err
		}
		if err := c.
			Cypher("UNWIND $ns AS i RETURN i * 2 AS i2").
			RunWithParams(ctx, map[string]any{"ns": ns}, "i2", &nsTimes2); err != nil {
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

func ExampleDriver_runWithParams() {
	ctx := context.Background()
	// Always use mock for examples to avoid connection dependencies
	m := NewMock()
	m.Bind(map[string]any{
		"ns": []any{int64(1), int64(2), int64(3)},
	})
	d := m

	var ns []int64
	err := d.Exec().
		Cypher("RETURN $ns AS ns").
		RunWithParams(ctx, map[string]any{
			"ns": []int64{1, 2, 3},
		}, "ns", &ns)

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
		records[i] = map[string]any{"i": int64(i)}
	}
	m.BindRecords(records)
	d := m

	ns := []int64{}
	session := d.ReadSession(ctx)
	defer func() {
		if err := session.Close(ctx); err != nil {
			panic(err)
		}
	}()
	err := session.ReadTransaction(ctx, func(c Client) error {
		var num int64
		params := map[string]any{
			"total": n,
		}
		return c.
			Cypher("UNWIND range(0, $total) AS i RETURN i").
			StreamWithParams(ctx, params, func() error {
				ns = append(ns, num)
				return nil
			}, "i", &num)
	})

	fmt.Printf("err: %v\n", err)
	fmt.Printf("ns: %v\n", ns)
	// Output: err: <nil>
	// ns: [0 1 2 3]
}

func TestWriteSession(t *testing.T) {
	ctx := context.Background()
	m := NewMock()
	m.Bind(map[string]any{"n": int64(1)})

	var n int64
	session := m.WriteSession(ctx)
	defer func() {
		require.NoError(t, session.Close(ctx))
	}()

	err := session.WriteTransaction(ctx, func(c Client) error {
		return c.Cypher("RETURN 1 AS n").Run(ctx, "n", &n)
	})
	require.NoError(t, err)
	assert.Equal(t, int64(1), n)
}

func TestBeginTransaction(t *testing.T) {
	ctx := context.Background()

	t.Run("commit transaction", func(t *testing.T) {
		m := NewMock()
		m.Bind(map[string]any{"n": int64(42)})

		session := m.WriteSession(ctx)
		defer func() { _ = session.Close(ctx) }()

		tx, err := session.BeginTransaction(ctx)
		require.NoError(t, err)

		var n int64
		err = tx.Run(func(c Client) error {
			return c.Cypher("RETURN 42 AS n").Run(ctx, "n", &n)
		})
		require.NoError(t, err)
		assert.Equal(t, int64(42), n)

		err = tx.Commit(ctx)
		require.NoError(t, err)

		err = tx.Close(ctx)
		require.NoError(t, err)
	})

	t.Run("rollback transaction", func(t *testing.T) {
		m := NewMock()
		m.Bind(map[string]any{"n": int64(1)})

		session := m.WriteSession(ctx)
		defer func() { _ = session.Close(ctx) }()

		tx, err := session.BeginTransaction(ctx)
		require.NoError(t, err)

		var n int64
		err = tx.Run(func(c Client) error {
			return c.Cypher("RETURN 1 AS n").Run(ctx, "n", &n)
		})
		require.NoError(t, err)

		err = tx.Rollback(ctx)
		require.NoError(t, err)

		err = tx.Close(ctx)
		require.NoError(t, err)
	})

	t.Run("close with joined errors", func(t *testing.T) {
		m := NewMock()
		m.Bind(map[string]any{"n": int64(1)})

		session := m.WriteSession(ctx)
		defer func() { _ = session.Close(ctx) }()

		tx, err := session.BeginTransaction(ctx)
		require.NoError(t, err)

		// Simulate an error that occurred during transaction work
		workErr := fmt.Errorf("work failed")
		err = tx.Close(ctx, workErr)
		require.Error(t, err)
		require.Contains(t, err.Error(), "work failed")
	})
}

func TestWithCausalConsistency(t *testing.T) {
	cfg := &Config{}
	WithCausalConsistency(func(ctx context.Context) string {
		return "user-123"
	})(cfg)

	require.NotNil(t, cfg.CausalConsistencyKey)
	key := cfg.CausalConsistencyKey(context.Background())
	assert.Equal(t, "user-123", key)
}

func TestWithTxConfig(t *testing.T) {
	ec := &execConfig{
		TransactionConfig: &neo4j.TransactionConfig{},
	}
	WithTxConfig(func(tc *neo4j.TransactionConfig) {
		tc.Timeout = 5000
	})(ec)

	assert.Equal(t, 5000*1, int(ec.Timeout))
}

func TestWithSessionConfig(t *testing.T) {
	ec := &execConfig{
		SessionConfig: &neo4j.SessionConfig{},
	}
	WithSessionConfig(func(sc *neo4j.SessionConfig) {
		sc.DatabaseName = "test-db"
	})(ec)

	assert.Equal(t, "test-db", ec.DatabaseName)
}

func TestDriverDB(t *testing.T) {
	m := NewMock()
	db := m.DB()
	require.NotNil(t, db)
}

func TestSessionClose(t *testing.T) {
	ctx := context.Background()
	m := NewMock()
	m.Bind(map[string]any{"n": int64(1)})

	session := m.ReadSession(ctx)

	// Close with no errors
	err := session.Close(ctx)
	require.NoError(t, err)
}

func TestSessionCloseWithError(t *testing.T) {
	ctx := context.Background()
	m := NewMock()
	m.Bind(map[string]any{"n": int64(1)})

	session := m.ReadSession(ctx)

	// Close with joined errors
	workErr := fmt.Errorf("some error")
	err := session.Close(ctx, workErr)
	require.Error(t, err)
	require.Contains(t, err.Error(), "some error")
}
