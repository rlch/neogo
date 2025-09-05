package neogo

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/neo4j/neo4j-go-driver/v5/neo4j"
	"github.com/rlch/neogo/db"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
)

func getNeo4JConnection(ctx context.Context) (string, neo4j.AuthToken) {
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
	return uri, neo4j.BasicAuth("neo4j", "password", "")
}

func newHybridDriver(t *testing.T, ctx context.Context) (d Driver, m mockDriver) {
	m = NewMock()
	if testing.Short() {
		d = m
	} else {
		_, cancel := startNeo4J(ctx)
		// Extract URI and auth from startNeo4J pattern
		uri, auth := getNeo4JConnection(ctx)
		var err error
		d, err = New(uri, auth)
		require.NoError(t, err)
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
