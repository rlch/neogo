package neogo_test

import (
	"context"
	"testing"
	"time"

	"github.com/neo4j/neo4j-go-driver/v5/neo4j"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	tcneo4j "github.com/testcontainers/testcontainers-go/modules/neo4j"

	"github.com/rlch/neogo"
	"github.com/rlch/neogo/internal"
)

// =============================================================================
// Test Container Setup (for _test package - separate from neogo package)
// =============================================================================

// neo4jContainer holds the test container and driver for integration tests
type neo4jContainer struct {
	container *tcneo4j.Neo4jContainer
	driver    neo4j.DriverWithContext
	boltURL   string
}

// startNeo4j starts a Neo4j testcontainer and returns connection details.
// Uses Neo4j Enterprise for full feature support.
// Skips the test if running with -short flag.
func startNeo4j(ctx context.Context, t *testing.T) *neo4jContainer {
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

	return &neo4jContainer{
		container: container,
		driver:    driver,
		boltURL:   boltURL,
	}
}

// close cleans up the container and driver
func (c *neo4jContainer) close(ctx context.Context, t *testing.T) {
	t.Helper()
	if c.driver != nil {
		_ = c.driver.Close(ctx)
	}
	if c.container != nil {
		_ = c.container.Terminate(ctx)
	}
}

// cleanupSchema drops all indexes and constraints created during tests
func cleanupSchema(ctx context.Context, t *testing.T, driver neo4j.DriverWithContext) {
	t.Helper()
	session := driver.NewSession(ctx, neo4j.SessionConfig{AccessMode: neo4j.AccessModeWrite})
	defer func() { _ = session.Close(ctx) }()

	// Drop all constraints first (some indexes are auto-created by constraints)
	result, err := session.Run(ctx, "SHOW CONSTRAINTS YIELD name RETURN name", nil)
	if err == nil {
		for result.Next(ctx) {
			name, _ := result.Record().Get("name")
			if nameStr, ok := name.(string); ok {
				_, _ = session.Run(ctx, "DROP CONSTRAINT "+nameStr+" IF EXISTS", nil)
			}
		}
	}

	// Drop all non-lookup indexes
	result, err = session.Run(ctx, "SHOW INDEXES YIELD name, type WHERE type <> 'LOOKUP' RETURN name", nil)
	if err == nil {
		for result.Next(ctx) {
			name, _ := result.Record().Get("name")
			if nameStr, ok := name.(string); ok {
				_, _ = session.Run(ctx, "DROP INDEX "+nameStr+" IF EXISTS", nil)
			}
		}
	}
}

// =============================================================================
// Test Types for Integration Tests
// =============================================================================

// schemaTestNode is a basic node with various schema elements
type schemaTestNode struct {
	internal.Node `neo4j:"SchemaTestNode"`
	Email         string `neo4j:"email,unique"`
	Name          string `neo4j:"name,index"`
	Bio           string `neo4j:"bio,notNull"`
}

func (schemaTestNode) IsNode()         {}
func (n schemaTestNode) GetID() string { return n.ID }

// schemaTestNodeFulltext tests fulltext indexes
type schemaTestNodeFulltext struct {
	internal.Node `neo4j:"SchemaTestFulltext"`
	Title         string `neo4j:"title,fulltext"`
	Description   string `neo4j:"description,fulltext:ft_test_search"`
}

func (schemaTestNodeFulltext) IsNode()         {}
func (n schemaTestNodeFulltext) GetID() string { return n.ID }

// schemaTestNodeComposite tests composite indexes
type schemaTestNodeComposite struct {
	internal.Node `neo4j:"SchemaTestComposite"`
	FirstName     string `neo4j:"first_name,index:idx_test_name,priority:1"`
	LastName      string `neo4j:"last_name,index:idx_test_name,priority:2"`
}

func (schemaTestNodeComposite) IsNode()         {}
func (n schemaTestNodeComposite) GetID() string { return n.ID }

// schemaTestNodeNodeKey tests node key constraints
type schemaTestNodeNodeKey struct {
	internal.Node `neo4j:"SchemaTestNodeKey"`
	TenantID      string `neo4j:"tenant_id,nodeKey:key_test_tenant,priority:1"`
	UserID        string `neo4j:"user_id,nodeKey:key_test_tenant,priority:2"`
}

func (schemaTestNodeNodeKey) IsNode()         {}
func (n schemaTestNodeNodeKey) GetID() string { return n.ID }

// schemaTestRel tests relationship indexes/constraints
type schemaTestRel struct {
	internal.Relationship `neo4j:"SCHEMA_TEST_REL"`
	Role                  string `neo4j:"role,index"`
	StartDate             string `neo4j:"start_date,notNull"`
}

func (schemaTestRel) IsRelationship() {}

// =============================================================================
// Integration Tests
// =============================================================================

func TestSchemaIntegration_GetIndexes(t *testing.T) {
	ctx := context.Background()
	nc := startNeo4j(ctx, t)
	defer nc.close(ctx, t)
	defer cleanupSchema(ctx, t, nc.driver)

	d, err := neogo.New(nc.boltURL, neo4j.NoAuth())
	require.NoError(t, err)

	t.Run("empty database returns empty slice", func(t *testing.T) {
		cleanupSchema(ctx, t, nc.driver)

		indexes, err := d.Schema().GetIndexes(ctx)
		require.NoError(t, err)
		assert.Empty(t, indexes, "expected no indexes in fresh database")
	})

	t.Run("after creating index, returns index info", func(t *testing.T) {
		cleanupSchema(ctx, t, nc.driver)

		// Create an index directly
		session := nc.driver.NewSession(ctx, neo4j.SessionConfig{AccessMode: neo4j.AccessModeWrite})
		_, err := session.Run(ctx, "CREATE INDEX test_manual_idx FOR (n:TestLabel) ON (n.prop)", nil)
		require.NoError(t, err)
		_ = session.Close(ctx)

		// Wait for index to be online
		time.Sleep(500 * time.Millisecond)

		indexes, err := d.Schema().GetIndexes(ctx)
		require.NoError(t, err)
		require.Len(t, indexes, 1)

		assert.Equal(t, "test_manual_idx", indexes[0].Name)
		assert.Equal(t, "RANGE", indexes[0].Type)
		assert.Equal(t, "NODE", indexes[0].EntityType)
		assert.Contains(t, indexes[0].LabelsOrTypes, "TestLabel")
		assert.Contains(t, indexes[0].Properties, "prop")
	})
}

func TestSchemaIntegration_GetConstraints(t *testing.T) {
	ctx := context.Background()
	nc := startNeo4j(ctx, t)
	defer nc.close(ctx, t)
	defer cleanupSchema(ctx, t, nc.driver)

	d, err := neogo.New(nc.boltURL, neo4j.NoAuth())
	require.NoError(t, err)

	t.Run("empty database returns empty slice", func(t *testing.T) {
		cleanupSchema(ctx, t, nc.driver)

		constraints, err := d.Schema().GetConstraints(ctx)
		require.NoError(t, err)
		assert.Empty(t, constraints, "expected no constraints in fresh database")
	})

	t.Run("after creating constraint, returns constraint info", func(t *testing.T) {
		cleanupSchema(ctx, t, nc.driver)

		// Create a constraint directly
		session := nc.driver.NewSession(ctx, neo4j.SessionConfig{AccessMode: neo4j.AccessModeWrite})
		_, err := session.Run(ctx, "CREATE CONSTRAINT test_manual_con FOR (n:TestLabel) REQUIRE n.prop IS UNIQUE", nil)
		require.NoError(t, err)
		_ = session.Close(ctx)

		// Wait for constraint to be created
		time.Sleep(500 * time.Millisecond)

		constraints, err := d.Schema().GetConstraints(ctx)
		require.NoError(t, err)
		require.Len(t, constraints, 1)

		assert.Equal(t, "test_manual_con", constraints[0].Name)
		assert.Equal(t, "UNIQUENESS", constraints[0].Type)
		assert.Equal(t, "NODE", constraints[0].EntityType)
		assert.Contains(t, constraints[0].LabelsOrTypes, "TestLabel")
		assert.Contains(t, constraints[0].Properties, "prop")
	})
}

func TestSchemaIntegration_AutoMigrate(t *testing.T) {
	ctx := context.Background()
	nc := startNeo4j(ctx, t)
	defer nc.close(ctx, t)
	defer cleanupSchema(ctx, t, nc.driver)

	t.Run("creates basic node schema", func(t *testing.T) {
		cleanupSchema(ctx, t, nc.driver)

		d, err := neogo.New(nc.boltURL, neo4j.NoAuth(), neogo.WithTypes(&schemaTestNode{}))
		require.NoError(t, err)

		actions, err := d.Schema().AutoMigrate(ctx)
		require.NoError(t, err)
		assert.NotEmpty(t, actions, "should have created some schema elements")

		// Verify indexes were created
		indexes, err := d.Schema().GetIndexes(ctx)
		require.NoError(t, err)

		// Should have at least the 'name' index
		indexNames := make([]string, len(indexes))
		for i, idx := range indexes {
			indexNames[i] = idx.Name
		}
		assert.Contains(t, indexNames, "idx_SchemaTestNode_name", "should have name index")

		// Verify constraints were created
		constraints, err := d.Schema().GetConstraints(ctx)
		require.NoError(t, err)

		constraintNames := make([]string, len(constraints))
		for i, con := range constraints {
			constraintNames[i] = con.Name
		}
		assert.Contains(t, constraintNames, "unique_SchemaTestNode_email", "should have email unique constraint")
		assert.Contains(t, constraintNames, "unique_SchemaTestNode_id", "should have auto ID constraint")
		assert.Contains(t, constraintNames, "notnull_SchemaTestNode_bio", "should have bio notNull constraint")
	})

	t.Run("creates fulltext index", func(t *testing.T) {
		cleanupSchema(ctx, t, nc.driver)

		d, err := neogo.New(nc.boltURL, neo4j.NoAuth(), neogo.WithTypes(&schemaTestNodeFulltext{}))
		require.NoError(t, err)

		_, err = d.Schema().AutoMigrate(ctx)
		require.NoError(t, err)

		indexes, err := d.Schema().GetIndexes(ctx)
		require.NoError(t, err)

		// Find fulltext indexes
		var fulltextIndexes []neogo.IndexInfo
		for _, idx := range indexes {
			if idx.Type == "FULLTEXT" {
				fulltextIndexes = append(fulltextIndexes, idx)
			}
		}
		assert.NotEmpty(t, fulltextIndexes, "should have created fulltext indexes")
	})

	t.Run("creates composite index", func(t *testing.T) {
		cleanupSchema(ctx, t, nc.driver)

		d, err := neogo.New(nc.boltURL, neo4j.NoAuth(), neogo.WithTypes(&schemaTestNodeComposite{}))
		require.NoError(t, err)

		_, err = d.Schema().AutoMigrate(ctx)
		require.NoError(t, err)

		indexes, err := d.Schema().GetIndexes(ctx)
		require.NoError(t, err)

		// Find the composite index
		var compositeIdx *neogo.IndexInfo
		for i, idx := range indexes {
			if idx.Name == "idx_test_name" {
				compositeIdx = &indexes[i]
				break
			}
		}
		require.NotNil(t, compositeIdx, "should have created composite index")
		assert.Len(t, compositeIdx.Properties, 2, "composite index should have 2 properties")
		assert.Equal(t, "first_name", compositeIdx.Properties[0])
		assert.Equal(t, "last_name", compositeIdx.Properties[1])
	})

	t.Run("creates node key constraint", func(t *testing.T) {
		cleanupSchema(ctx, t, nc.driver)

		d, err := neogo.New(nc.boltURL, neo4j.NoAuth(), neogo.WithTypes(&schemaTestNodeNodeKey{}))
		require.NoError(t, err)

		_, err = d.Schema().AutoMigrate(ctx)
		require.NoError(t, err)

		constraints, err := d.Schema().GetConstraints(ctx)
		require.NoError(t, err)

		// Find the node key constraint
		var nodeKeyConstraint *neogo.ConstraintInfo
		for i, con := range constraints {
			if con.Name == "key_test_tenant" {
				nodeKeyConstraint = &constraints[i]
				break
			}
		}
		require.NotNil(t, nodeKeyConstraint, "should have created node key constraint")
		assert.Equal(t, "NODE_KEY", nodeKeyConstraint.Type)
		assert.Len(t, nodeKeyConstraint.Properties, 2, "node key should have 2 properties")
	})

	t.Run("creates relationship schema", func(t *testing.T) {
		cleanupSchema(ctx, t, nc.driver)

		d, err := neogo.New(nc.boltURL, neo4j.NoAuth(), neogo.WithTypes(&schemaTestRel{}))
		require.NoError(t, err)

		_, err = d.Schema().AutoMigrate(ctx)
		require.NoError(t, err)

		indexes, err := d.Schema().GetIndexes(ctx)
		require.NoError(t, err)

		// Find the relationship index
		var relIdx *neogo.IndexInfo
		for i, idx := range indexes {
			if idx.EntityType == "RELATIONSHIP" {
				relIdx = &indexes[i]
				break
			}
		}
		require.NotNil(t, relIdx, "should have created relationship index")
		assert.Contains(t, relIdx.LabelsOrTypes, "SCHEMA_TEST_REL")
	})

	t.Run("idempotent - running twice succeeds", func(t *testing.T) {
		cleanupSchema(ctx, t, nc.driver)

		d, err := neogo.New(nc.boltURL, neo4j.NoAuth(), neogo.WithTypes(&schemaTestNode{}))
		require.NoError(t, err)

		// First migration
		actions1, err := d.Schema().AutoMigrate(ctx)
		require.NoError(t, err)
		assert.NotEmpty(t, actions1, "first migration should create schema")

		// Second migration should be a no-op
		actions2, err := d.Schema().AutoMigrate(ctx)
		require.NoError(t, err)
		assert.Empty(t, actions2, "second migration should be a no-op")
	})
}

func TestSchemaIntegration_NeedsMigration(t *testing.T) {
	ctx := context.Background()
	nc := startNeo4j(ctx, t)
	defer nc.close(ctx, t)
	defer cleanupSchema(ctx, t, nc.driver)

	t.Run("empty DB needs migration", func(t *testing.T) {
		cleanupSchema(ctx, t, nc.driver)

		d, err := neogo.New(nc.boltURL, neo4j.NoAuth(), neogo.WithTypes(&schemaTestNode{}))
		require.NoError(t, err)

		needsMigration, actions, err := d.Schema().NeedsMigration(ctx)
		require.NoError(t, err)
		assert.True(t, needsMigration, "empty DB should need migration")
		assert.NotEmpty(t, actions, "should have pending actions")
	})

	t.Run("fully migrated DB does not need migration", func(t *testing.T) {
		cleanupSchema(ctx, t, nc.driver)

		d, err := neogo.New(nc.boltURL, neo4j.NoAuth(), neogo.WithTypes(&schemaTestNode{}))
		require.NoError(t, err)

		// First migrate
		_, err = d.Schema().AutoMigrate(ctx)
		require.NoError(t, err)

		// Then check
		needsMigration, actions, err := d.Schema().NeedsMigration(ctx)
		require.NoError(t, err)
		assert.False(t, needsMigration, "migrated DB should not need migration")
		assert.Empty(t, actions, "should have no pending actions")
	})

	t.Run("partial migration detected correctly", func(t *testing.T) {
		cleanupSchema(ctx, t, nc.driver)

		// Create only the index, not the constraints
		session := nc.driver.NewSession(ctx, neo4j.SessionConfig{AccessMode: neo4j.AccessModeWrite})
		_, err := session.Run(ctx, "CREATE INDEX idx_SchemaTestNode_name IF NOT EXISTS FOR (n:SchemaTestNode) ON (n.name)", nil)
		require.NoError(t, err)
		_ = session.Close(ctx)

		d, err := neogo.New(nc.boltURL, neo4j.NoAuth(), neogo.WithTypes(&schemaTestNode{}))
		require.NoError(t, err)

		needsMigration, actions, err := d.Schema().NeedsMigration(ctx)
		require.NoError(t, err)
		assert.True(t, needsMigration, "partial migration should need more migration")
		assert.NotEmpty(t, actions, "should have pending actions for constraints")

		// The index should not be in pending actions
		for _, action := range actions {
			assert.NotEqual(t, "idx_SchemaTestNode_name", action.Name, "existing index should not be in pending actions")
		}
	})
}

func TestSchemaIntegration_RoundTrip(t *testing.T) {
	ctx := context.Background()
	nc := startNeo4j(ctx, t)
	defer nc.close(ctx, t)
	defer cleanupSchema(ctx, t, nc.driver)

	t.Run("full cycle: register -> check -> migrate -> check", func(t *testing.T) {
		cleanupSchema(ctx, t, nc.driver)

		// 1. Create driver with types
		d, err := neogo.New(nc.boltURL, neo4j.NoAuth(), neogo.WithTypes(
			&schemaTestNode{},
			&schemaTestNodeComposite{},
		))
		require.NoError(t, err)

		// 2. Check that migration is needed
		needsMigration, pendingActions, err := d.Schema().NeedsMigration(ctx)
		require.NoError(t, err)
		assert.True(t, needsMigration, "fresh DB should need migration")
		t.Logf("Pending actions before migration: %d", len(pendingActions))
		for _, action := range pendingActions {
			t.Logf("  - %s: %s", action.Type, action.Name)
		}

		// 3. Run migration
		executedActions, err := d.Schema().AutoMigrate(ctx)
		require.NoError(t, err)
		assert.NotEmpty(t, executedActions, "should have executed some actions")
		t.Logf("Executed actions: %d", len(executedActions))
		for _, action := range executedActions {
			t.Logf("  - %s: %s", action.Type, action.Name)
		}

		// 4. Check that migration is no longer needed
		needsMigration, pendingActions, err = d.Schema().NeedsMigration(ctx)
		require.NoError(t, err)
		assert.False(t, needsMigration, "after migration, should not need migration")
		assert.Empty(t, pendingActions, "after migration, should have no pending actions")

		// 5. Verify schema in database
		indexes, err := d.Schema().GetIndexes(ctx)
		require.NoError(t, err)
		t.Logf("Indexes in DB: %d", len(indexes))
		for _, idx := range indexes {
			t.Logf("  - %s (%s) on %v.%v", idx.Name, idx.Type, idx.LabelsOrTypes, idx.Properties)
		}

		constraints, err := d.Schema().GetConstraints(ctx)
		require.NoError(t, err)
		t.Logf("Constraints in DB: %d", len(constraints))
		for _, con := range constraints {
			t.Logf("  - %s (%s) on %v.%v", con.Name, con.Type, con.LabelsOrTypes, con.Properties)
		}
	})
}
