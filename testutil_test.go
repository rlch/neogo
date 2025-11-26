package neogo

import (
	"context"
	"testing"
	"time"

	"github.com/neo4j/neo4j-go-driver/v5/neo4j"
	"github.com/stretchr/testify/require"
	tcneo4j "github.com/testcontainers/testcontainers-go/modules/neo4j"
	"golang.org/x/sync/semaphore"

	"github.com/rlch/neogo/internal"
)

// neo4jTestContainer holds the test container and driver for integration tests
type neo4jTestContainer struct {
	Container *tcneo4j.Neo4jContainer
	Driver    neo4j.DriverWithContext
	BoltURL   string
}

// startNeo4jContainer starts a Neo4j testcontainer and returns connection details.
// Uses Neo4j Enterprise for full feature support.
// Skips the test if running with -short flag.
func startNeo4jContainer(ctx context.Context, t *testing.T) *neo4jTestContainer {
	t.Helper()

	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	container, err := tcneo4j.Run(ctx,
		"neo4j:5.26-enterprise",
		tcneo4j.WithoutAuthentication(),
		tcneo4j.WithAcceptCommercialLicenseAgreement(),
	)
	require.NoError(t, err, "failed to start Neo4j container")

	boltURL, err := container.BoltUrl(ctx)
	require.NoError(t, err, "failed to get Bolt URL")

	driver, err := neo4j.NewDriverWithContext(boltURL, neo4j.NoAuth())
	require.NoError(t, err, "failed to create Neo4j driver")

	// Wait for connectivity
	ctx2, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	err = driver.VerifyConnectivity(ctx2)
	require.NoError(t, err, "failed to verify Neo4j connectivity")

	return &neo4jTestContainer{
		Container: container,
		Driver:    driver,
		BoltURL:   boltURL,
	}
}

// Close cleans up the container and driver
func (c *neo4jTestContainer) Close(ctx context.Context, t *testing.T) {
	t.Helper()
	if c.Driver != nil {
		_ = c.Driver.Close(ctx)
	}
	if c.Container != nil {
		_ = c.Container.Terminate(ctx)
	}
}

// NewTestDriver creates a neogo Driver from the test container
func (c *neo4jTestContainer) NewTestDriver(t *testing.T, types ...any) Driver {
	t.Helper()
	d := &driver{
		reg:              internal.NewRegistry(),
		db:               c.Driver,
		sessionSemaphore: semaphore.NewWeighted(100),
	}
	if len(types) > 0 {
		d.reg.RegisterTypes(types...)
	}
	return d
}

// CleanupData removes all nodes and relationships from the database
func (c *neo4jTestContainer) CleanupData(ctx context.Context, t *testing.T) {
	t.Helper()
	session := c.Driver.NewSession(ctx, neo4j.SessionConfig{AccessMode: neo4j.AccessModeWrite})
	defer session.Close(ctx)
	_, err := session.Run(ctx, "MATCH (n) DETACH DELETE n", nil)
	require.NoError(t, err, "failed to cleanup data")
}
