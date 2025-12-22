package neogo

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestMockDriver(t *testing.T) {
	ctx := context.Background()

	t.Run("must provide bindings", func(t *testing.T) {
		require := require.New(t)
		m := NewMock()
		require.PanicsWithError("mock client used without bindings for all transactions", func() {
			_ = m.Exec().Cypher("RETURN n").Run(ctx, "n", new(int))
		})
	})

	t.Run("must provide bindings for all transactions", func(t *testing.T) {
		require := require.New(t)
		m := NewMock()
		m.Bind(map[string]any{"n": int64(1)})

		var out int64
		err := m.Exec().Cypher("RETURN n").Run(ctx, "n", &out)
		require.NoError(err)
		require.Equal(int64(1), out)

		require.PanicsWithError("mock client used without bindings for all transactions", func() {
			_ = m.Exec().Cypher("RETURN n").Run(ctx, "n", new(int))
		})
	})

	t.Run("binds to a single record", func(t *testing.T) {
		require := require.New(t)
		m := NewMock()
		m.Bind(map[string]any{"a": int64(1)})
		m.Bind(map[string]any{"b": int64(2)})

		var out int64
		err := m.Exec().Cypher("RETURN a").Run(ctx, "a", &out)
		require.NoError(err)
		require.Equal(int64(1), out)

		err = m.Exec().Cypher("RETURN b").Run(ctx, "b", &out)
		require.NoError(err)
		require.Equal(int64(2), out)
	})

	t.Run("binds to multiple records", func(t *testing.T) {
		require := require.New(t)
		m := NewMock()
		m.BindRecords([]map[string]any{
			{"a": int64(1), "b": int64(2)},
			{"a": int64(2), "b": int64(4)},
		})
		m.BindRecords([]map[string]any{
			{"a": int64(3), "b": int64(6)},
			{"a": int64(4), "b": int64(8)},
		})

		var outA, outB []int64
		err := m.Exec().Cypher("RETURN a, b").Run(ctx, "a", &outA, "b", &outB)
		require.NoError(err)
		require.Equal([]int64{1, 2}, outA)
		require.Equal([]int64{2, 4}, outB)

		err = m.Exec().Cypher("RETURN a, b").Run(ctx, "a", &outA, "b", &outB)
		require.NoError(err)
		require.Equal([]int64{3, 4}, outA)
		require.Equal([]int64{6, 8}, outB)
	})
}
