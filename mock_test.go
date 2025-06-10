package neogo

import (
	"context"
	"testing"

	"github.com/rlch/neogo/db"
	"github.com/stretchr/testify/require"
)

func newHybridDriver(t *testing.T, ctx context.Context) (d Driver, m mockDriver) {
	m = NewMock()
	if testing.Short() {
		d = m
	} else {
		neo4j, cancel := startNeo4J(ctx)
		d = New(neo4j)
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

	t.Run("must provide bindings", func(t *testing.T) {
		require := require.New(t)
		m := NewMock()
		require.PanicsWithError("mock client used without bindings for all transactions", func() {
			_ = m.Exec().Return("n").Run(ctx)
		})
	})

	t.Run("must provide bindings for all transactions", func(t *testing.T) {
		require := require.New(t)
		m := NewMock()
		m.Bind(map[string]any{"n": 1})

		var out int
		err := m.Exec().Return(db.Qual(&out, "n")).Run(ctx)
		require.NoError(err)
		require.Equal(1, out)

		require.PanicsWithError("mock client used without bindings for all transactions", func() {
			_ = m.Exec().Return("n").Run(ctx)
		})
	})

	t.Run("binds to a single record", func(t *testing.T) {
		require := require.New(t)
		m := NewMock()
		m.Bind(map[string]any{"a": 1})
		m.Bind(map[string]any{"b": 2})

		var out int
		err := m.Exec().Return(db.Qual(&out, "a")).Run(ctx)
		require.NoError(err)
		require.Equal(1, out)

		err = m.Exec().Return(db.Qual(&out, "b")).Run(ctx)
		require.NoError(err)
		require.Equal(2, out)
	})

	t.Run("binds to multiple records", func(t *testing.T) {
		require := require.New(t)
		m := NewMock()
		m.BindRecords([]map[string]any{
			{"a": 1, "b": 2},
			{"a": 2, "b": 4},
		})
		m.BindRecords([]map[string]any{
			{"a": 3, "b": 6},
			{"a": 4, "b": 8},
		})

		var outA, outB []int
		err := m.Exec().Return(
			db.Qual(&outA, "a"),
			db.Qual(&outB, "b"),
		).Run(ctx)
		require.NoError(err)
		require.Equal([]int{1, 2}, outA)
		require.Equal([]int{2, 4}, outB)

		err = m.Exec().Return(
			db.Qual(&outA, "a"),
			db.Qual(&outB, "b"),
		).Run(ctx)
		require.NoError(err)
		require.Equal([]int{3, 4}, outA)
		require.Equal([]int{6, 8}, outB)
	})
}
