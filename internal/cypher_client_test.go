package internal

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewCypherClient(t *testing.T) {
	r := NewRegistry()
	c := NewCypherClient(r)

	require.NotNil(t, c)
	require.NotNil(t, c.Registry)
	require.Empty(t, c.parameters)
	require.Empty(t, c.bindings)
}

func TestCypher(t *testing.T) {
	r := NewRegistry()
	c := NewCypherClient(r)

	t.Run("read query", func(t *testing.T) {
		runner := c.Cypher("MATCH (n) RETURN n")
		require.NotNil(t, runner)
		require.False(t, runner.client.isWrite)
	})

	t.Run("CREATE is write", func(t *testing.T) {
		c2 := NewCypherClient(r)
		runner := c2.Cypher("CREATE (n:Person) RETURN n")
		require.True(t, runner.client.isWrite)
	})

	t.Run("MERGE is write", func(t *testing.T) {
		c2 := NewCypherClient(r)
		runner := c2.Cypher("MERGE (n:Person {id: 1}) RETURN n")
		require.True(t, runner.client.isWrite)
	})

	t.Run("DELETE is write", func(t *testing.T) {
		c2 := NewCypherClient(r)
		runner := c2.Cypher("MATCH (n) DELETE n")
		require.True(t, runner.client.isWrite)
	})

	t.Run("SET is write", func(t *testing.T) {
		c2 := NewCypherClient(r)
		runner := c2.Cypher("MATCH (n) SET n.name = 'test'")
		require.True(t, runner.client.isWrite)
	})

	t.Run("REMOVE is write", func(t *testing.T) {
		c2 := NewCypherClient(r)
		runner := c2.Cypher("MATCH (n) REMOVE n.name")
		require.True(t, runner.client.isWrite)
	})

	t.Run("CALL is write", func(t *testing.T) {
		c2 := NewCypherClient(r)
		runner := c2.Cypher("CALL db.labels()")
		require.True(t, runner.client.isWrite)
	})
}

func TestCompile(t *testing.T) {
	r := NewRegistry()
	c := NewCypherClient(r)

	t.Run("compiles without bindings", func(t *testing.T) {
		runner := c.Cypher("RETURN 1")
		cy, err := runner.Compile()
		require.NoError(t, err)
		assert.Equal(t, "RETURN 1", cy.Cypher)
		assert.Empty(t, cy.Bindings)
	})

	t.Run("compiles with bindings", func(t *testing.T) {
		c2 := NewCypherClient(r)
		runner := c2.Cypher("MATCH (n) RETURN n")
		var n int64
		cy, err := runner.Compile("n", &n)
		require.NoError(t, err)
		assert.Equal(t, "MATCH (n) RETURN n", cy.Cypher)
		assert.Len(t, cy.Bindings, 1)
		assert.Contains(t, cy.Bindings, "n")
	})

	t.Run("fails with odd bindings count", func(t *testing.T) {
		c2 := NewCypherClient(r)
		runner := c2.Cypher("RETURN n")
		_, err := runner.Compile("n")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "bindings must be pairs")
	})

	t.Run("fails with non-string binding name", func(t *testing.T) {
		c2 := NewCypherClient(r)
		runner := c2.Cypher("RETURN n")
		var n int64
		_, err := runner.Compile(123, &n)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "must be a string")
	})
}

func TestCompileWithParams(t *testing.T) {
	r := NewRegistry()
	c := NewCypherClient(r)

	t.Run("compiles with parameters", func(t *testing.T) {
		runner := c.Cypher("MATCH (n {name: $name}) RETURN n")
		var n int64
		cy, err := runner.CompileWithParams(
			map[string]any{"name": "Alice"},
			"n", &n,
		)
		require.NoError(t, err)
		assert.Equal(t, "Alice", cy.Parameters["name"])
	})

	t.Run("trims trailing newlines", func(t *testing.T) {
		c2 := NewCypherClient(r)
		runner := c2.Cypher("RETURN 1\n\n")
		cy, err := runner.Compile()
		require.NoError(t, err)
		assert.Equal(t, "RETURN 1", cy.Cypher)
	})

	t.Run("builds binding plans for bindings", func(t *testing.T) {
		c2 := NewCypherClient(r)
		runner := c2.Cypher("RETURN n")
		var n string
		cy, err := runner.Compile("n", &n)
		require.NoError(t, err)
		require.NotNil(t, cy.Plans)
		require.Contains(t, cy.Plans, "n")
	})

	t.Run("sets IsWrite for write queries", func(t *testing.T) {
		c2 := NewCypherClient(r)
		runner := c2.Cypher("CREATE (n) RETURN n")
		cy, err := runner.Compile()
		require.NoError(t, err)
		assert.True(t, cy.IsWrite)
	})
}

func TestCypherRunnerPrint(t *testing.T) {
	r := NewRegistry()
	c := NewCypherClient(r)
	runner := c.Cypher("MATCH (n) RETURN n")

	// Print returns the same runner for chaining
	result := runner.Print()
	assert.Equal(t, runner, result)
}
