package neogo

import (
	"context"

	"github.com/rlch/neogo/internal/codec"
)

// Schema provides schema introspection and migration capabilities for Neo4j.
// It allows querying existing indexes/constraints and performing additive migrations.
type Schema interface {
	// GetIndexes returns all indexes from the database.
	// Excludes internal LOOKUP indexes used by Neo4j.
	GetIndexes(ctx context.Context) ([]IndexInfo, error)

	// GetConstraints returns all constraints from the database.
	GetConstraints(ctx context.Context) ([]ConstraintInfo, error)

	// AutoMigrate creates missing indexes and constraints for registered types.
	// If no types are provided, migrates all registered types.
	// It is ADDITIVE ONLY - it never drops existing schema elements.
	// Returns the list of actions that were executed.
	AutoMigrate(ctx context.Context, types ...any) ([]MigrationAction, error)

	// NeedsMigration checks if any registered types need schema migration.
	// If no types are provided, checks all registered types.
	// Returns true if migration is needed, along with the pending actions.
	NeedsMigration(ctx context.Context, types ...any) (bool, []MigrationAction, error)
}

// IndexInfo represents an index as returned by SHOW INDEXES.
type IndexInfo struct {
	Name          string         // Index name
	Type          string         // RANGE, TEXT, POINT, FULLTEXT, VECTOR, LOOKUP
	EntityType    string         // NODE or RELATIONSHIP
	LabelsOrTypes []string       // Labels (for nodes) or types (for relationships)
	Properties    []string       // Property names included in the index
	State         string         // ONLINE, POPULATING, FAILED
	Options       map[string]any // Index-specific options (e.g., vector dimensions)
}

// ConstraintInfo represents a constraint as returned by SHOW CONSTRAINTS.
type ConstraintInfo struct {
	Name          string   // Constraint name
	Type          string   // UNIQUENESS, NODE_KEY, NODE_PROPERTY_EXISTENCE, etc.
	EntityType    string   // NODE or RELATIONSHIP
	LabelsOrTypes []string // Labels (for nodes) or types (for relationships)
	Properties    []string // Property names included in the constraint
}

// MigrationAction represents a pending or executed schema change.
type MigrationAction struct {
	Type   MigrationActionType // CREATE_INDEX or CREATE_CONSTRAINT
	Name   string              // Schema element name
	Cypher string              // The Cypher statement to execute
}

// MigrationActionType represents the type of migration action.
type MigrationActionType string

const (
	// MigrationCreateIndex represents a CREATE INDEX action.
	MigrationCreateIndex MigrationActionType = "CREATE_INDEX"
	// MigrationCreateConstraint represents a CREATE CONSTRAINT action.
	MigrationCreateConstraint MigrationActionType = "CREATE_CONSTRAINT"
)

// IndexType re-exports codec.IndexType for external use.
type IndexType = codec.IndexType

// Index type constants re-exported from codec.
const (
	IndexTypeRange    = codec.IndexTypeRange
	IndexTypeText     = codec.IndexTypeText
	IndexTypePoint    = codec.IndexTypePoint
	IndexTypeFulltext = codec.IndexTypeFulltext
	IndexTypeVector   = codec.IndexTypeVector
)

// ConstraintType re-exports codec.ConstraintType for external use.
type ConstraintType = codec.ConstraintType

// Constraint type constants re-exported from codec.
const (
	ConstraintTypeUnique  = codec.ConstraintTypeUnique
	ConstraintTypeNodeKey = codec.ConstraintTypeNodeKey
	ConstraintTypeNotNull = codec.ConstraintTypeNotNull
)
