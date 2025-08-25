package neogo

import (
	"context"
	"testing"

	"github.com/neo4j/neo4j-go-driver/v5/neo4j"
	"github.com/rlch/neogo/db"
	"github.com/stretchr/testify/require"
)

func newHybridDriver(t *testing.T, ctx context.Context) (d Driver, m mockDriver) {
	m = NewMock()
	if testing.Short() {
		d = m
	} else {
		uri, cancel := startNeo4J(ctx)
		var err error
		d, err = New(uri, neo4j.BasicAuth("neo4j", "password", ""))
		if err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() {
			if err := cancel(ctx); err != nil {
				t.Fatal(err)
			}
		})
	}
	return
}

func TestMockDriver(t *testing.T) {
	ctx := context.Background()

	t.Run("must provide bindings for all transactions", func(t *testing.T) {
		d, m := newHybridDriver(t, ctx)
		m.Bind(nil)
		
		// Should work without error when bindings are provided
		err := d.Exec().
			Cypher("RETURN 1").
			Run(ctx)
		require.NoError(t, err)
	})

	t.Run("mock with data", func(t *testing.T) {
		d, m := newHybridDriver(t, ctx)
		m.Bind(map[string]any{
			"test": "value",
		})
		
		var result string
		err := d.Exec().
			Return(db.Qual(&result, "test")).
			Run(ctx)
		require.NoError(t, err)
		require.Equal(t, "value", result)
	})
}