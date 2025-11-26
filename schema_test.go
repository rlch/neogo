package neogo_test

import (
	"reflect"
	"testing"

	"github.com/rlch/neogo"
	"github.com/rlch/neogo/internal"
	"github.com/rlch/neogo/internal/codec"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// =============================================================================
// Test Types
// =============================================================================

// personNode is a basic node for schema tests
type personNode struct {
	internal.Node `neo4j:"Person"`
	Email         string `neo4j:"email,unique"`
	Name          string `neo4j:"name,index"`
}

func (personNode) IsNode()         {}
func (p personNode) GetID() string { return p.ID }

// movieNode has multiple schema elements
type movieNode struct {
	internal.Node `neo4j:"Movie"`
	Title         string    `neo4j:"title,index"`
	Plot          string    `neo4j:"plot,fulltext"`
	Embedding     []float32 `neo4j:"embedding,vector:1536"`
}

func (movieNode) IsNode()         {}
func (m movieNode) GetID() string { return m.ID }

// companyNode with composite indexes
type companyNode struct {
	internal.Node `neo4j:"Company"`
	TenantID      string `neo4j:"tenant_id,unique:uniq_company_tenant,priority:1"`
	CompanyID     string `neo4j:"company_id,unique:uniq_company_tenant,priority:2"`
}

func (companyNode) IsNode()         {}
func (c companyNode) GetID() string { return c.ID }

// workedAtRel is a relationship with schema elements
type workedAtRel struct {
	internal.Relationship `neo4j:"WORKED_AT"`
	Role                  string `neo4j:"role,index"`
	StartDate             string `neo4j:"start_date,notNull"`
}

func (workedAtRel) IsRelationship() {}

// simpleNode has only auto ID constraint (no explicit schema tags)
type simpleNode struct {
	internal.Node `neo4j:"SimpleNode"`
	Name          string `neo4j:"name"`
}

func (simpleNode) IsNode()         {}
func (s simpleNode) GetID() string { return s.ID }

// =============================================================================
// Phase 3: Schema Migration Logic Tests
// =============================================================================

// computeMigrationActions is the core migration logic extracted for testing.
// It takes existing schema from DB and required schema from registry, returns needed actions.
func computeMigrationActions(
	existingIndexes []neogo.IndexInfo,
	existingConstraints []neogo.ConstraintInfo,
	schemas []*codec.SchemaMeta,
) []neogo.MigrationAction {
	// Build lookup maps by name
	indexByName := make(map[string]struct{}, len(existingIndexes))
	for _, idx := range existingIndexes {
		indexByName[idx.Name] = struct{}{}
	}

	constraintByName := make(map[string]struct{}, len(existingConstraints))
	for _, con := range existingConstraints {
		constraintByName[con.Name] = struct{}{}
	}

	// Find missing indexes and constraints
	var actions []neogo.MigrationAction

	for _, schema := range schemas {
		// Check indexes
		for _, idx := range schema.Indexes {
			if _, exists := indexByName[idx.Name]; !exists {
				cypher := idx.GenerateIndexCypher()
				if cypher != "" {
					actions = append(actions, neogo.MigrationAction{
						Type:   neogo.MigrationCreateIndex,
						Name:   idx.Name,
						Cypher: cypher,
					})
				}
			}
		}

		// Check constraints
		for _, con := range schema.Constraints {
			if _, exists := constraintByName[con.Name]; !exists {
				cypher := con.GenerateConstraintCypher()
				if cypher != "" {
					actions = append(actions, neogo.MigrationAction{
						Type:   neogo.MigrationCreateConstraint,
						Name:   con.Name,
						Cypher: cypher,
					})
				}
			}
		}
	}

	return actions
}

func TestMigrationLogic_EmptyDatabase(t *testing.T) {
	t.Run("all schema elements need migration when DB is empty", func(t *testing.T) {
		require := require.New(t)

		// Empty database
		existingIndexes := []neogo.IndexInfo{}
		existingConstraints := []neogo.ConstraintInfo{}

		// Schema from registry
		schema := &codec.SchemaMeta{
			TypeName: "Person",
			Labels:   []string{"Person"},
			IsNode:   true,
			Indexes: []codec.IndexDef{
				{
					Name:       "idx_Person_email",
					Type:       codec.IndexTypeRange,
					Label:      "Person",
					IsNode:     true,
					Properties: []codec.IndexProperty{{Name: "email", Priority: 10}},
				},
			},
			Constraints: []codec.ConstraintDef{
				{
					Name:       "unique_Person_id",
					Type:       codec.ConstraintTypeUnique,
					Label:      "Person",
					IsNode:     true,
					Properties: []string{"id"},
				},
			},
		}

		actions := computeMigrationActions(existingIndexes, existingConstraints, []*codec.SchemaMeta{schema})

		require.Len(actions, 2, "should have 2 actions for empty DB")

		// Check for index action
		var indexAction, constraintAction *neogo.MigrationAction
		for i := range actions {
			if actions[i].Type == neogo.MigrationCreateIndex {
				indexAction = &actions[i]
			} else if actions[i].Type == neogo.MigrationCreateConstraint {
				constraintAction = &actions[i]
			}
		}

		require.NotNil(indexAction)
		require.Equal("idx_Person_email", indexAction.Name)
		require.Contains(indexAction.Cypher, "CREATE INDEX")

		require.NotNil(constraintAction)
		require.Equal("unique_Person_id", constraintAction.Name)
		require.Contains(constraintAction.Cypher, "CREATE CONSTRAINT")
	})
}

func TestMigrationLogic_PartialSchema(t *testing.T) {
	t.Run("only missing elements need migration", func(t *testing.T) {
		require := require.New(t)

		// Database has the index already
		existingIndexes := []neogo.IndexInfo{
			{Name: "idx_Person_email", Type: "RANGE", EntityType: "NODE", State: "ONLINE"},
		}
		existingConstraints := []neogo.ConstraintInfo{}

		// Schema requires both index and constraint
		schema := &codec.SchemaMeta{
			TypeName: "Person",
			Labels:   []string{"Person"},
			IsNode:   true,
			Indexes: []codec.IndexDef{
				{
					Name:       "idx_Person_email",
					Type:       codec.IndexTypeRange,
					Label:      "Person",
					IsNode:     true,
					Properties: []codec.IndexProperty{{Name: "email", Priority: 10}},
				},
			},
			Constraints: []codec.ConstraintDef{
				{
					Name:       "unique_Person_id",
					Type:       codec.ConstraintTypeUnique,
					Label:      "Person",
					IsNode:     true,
					Properties: []string{"id"},
				},
			},
		}

		actions := computeMigrationActions(existingIndexes, existingConstraints, []*codec.SchemaMeta{schema})

		require.Len(actions, 1, "should only need constraint")
		require.Equal(neogo.MigrationCreateConstraint, actions[0].Type)
		require.Equal("unique_Person_id", actions[0].Name)
	})
}

func TestMigrationLogic_FullyMigrated(t *testing.T) {
	t.Run("no actions when DB has all schema elements", func(t *testing.T) {
		require := require.New(t)

		// Database has everything
		existingIndexes := []neogo.IndexInfo{
			{Name: "idx_Person_email", Type: "RANGE", EntityType: "NODE", State: "ONLINE"},
		}
		existingConstraints := []neogo.ConstraintInfo{
			{Name: "unique_Person_id", Type: "UNIQUENESS", EntityType: "NODE"},
		}

		// Schema matches
		schema := &codec.SchemaMeta{
			TypeName: "Person",
			Labels:   []string{"Person"},
			IsNode:   true,
			Indexes: []codec.IndexDef{
				{
					Name:       "idx_Person_email",
					Type:       codec.IndexTypeRange,
					Label:      "Person",
					IsNode:     true,
					Properties: []codec.IndexProperty{{Name: "email", Priority: 10}},
				},
			},
			Constraints: []codec.ConstraintDef{
				{
					Name:       "unique_Person_id",
					Type:       codec.ConstraintTypeUnique,
					Label:      "Person",
					IsNode:     true,
					Properties: []string{"id"},
				},
			},
		}

		actions := computeMigrationActions(existingIndexes, existingConstraints, []*codec.SchemaMeta{schema})

		require.Len(actions, 0, "should have no actions when fully migrated")
	})
}

func TestMigrationLogic_MultipleTypes(t *testing.T) {
	t.Run("handles multiple type schemas", func(t *testing.T) {
		require := require.New(t)

		// Database has Person index only
		existingIndexes := []neogo.IndexInfo{
			{Name: "idx_Person_email", Type: "RANGE", EntityType: "NODE", State: "ONLINE"},
		}
		existingConstraints := []neogo.ConstraintInfo{}

		// Two schemas
		personSchema := &codec.SchemaMeta{
			TypeName: "Person",
			Labels:   []string{"Person"},
			IsNode:   true,
			Indexes: []codec.IndexDef{
				{Name: "idx_Person_email", Type: codec.IndexTypeRange, Label: "Person", IsNode: true, Properties: []codec.IndexProperty{{Name: "email"}}},
			},
			Constraints: []codec.ConstraintDef{
				{Name: "unique_Person_id", Type: codec.ConstraintTypeUnique, Label: "Person", IsNode: true, Properties: []string{"id"}},
			},
		}

		movieSchema := &codec.SchemaMeta{
			TypeName: "Movie",
			Labels:   []string{"Movie"},
			IsNode:   true,
			Indexes: []codec.IndexDef{
				{Name: "idx_Movie_title", Type: codec.IndexTypeRange, Label: "Movie", IsNode: true, Properties: []codec.IndexProperty{{Name: "title"}}},
				{Name: "fulltext_Movie_plot", Type: codec.IndexTypeFulltext, Label: "Movie", IsNode: true, Properties: []codec.IndexProperty{{Name: "plot"}}},
			},
			Constraints: []codec.ConstraintDef{
				{Name: "unique_Movie_id", Type: codec.ConstraintTypeUnique, Label: "Movie", IsNode: true, Properties: []string{"id"}},
			},
		}

		actions := computeMigrationActions(existingIndexes, existingConstraints, []*codec.SchemaMeta{personSchema, movieSchema})

		// Should need: Person constraint + Movie index + Movie fulltext + Movie constraint
		require.Len(actions, 4)

		// Verify action names
		actionNames := make(map[string]bool)
		for _, a := range actions {
			actionNames[a.Name] = true
		}

		assert.True(t, actionNames["unique_Person_id"])
		assert.True(t, actionNames["idx_Movie_title"])
		assert.True(t, actionNames["fulltext_Movie_plot"])
		assert.True(t, actionNames["unique_Movie_id"])
	})
}

func TestMigrationLogic_RelationshipSchema(t *testing.T) {
	t.Run("handles relationship schemas", func(t *testing.T) {
		require := require.New(t)

		existingIndexes := []neogo.IndexInfo{}
		existingConstraints := []neogo.ConstraintInfo{}

		// Relationship schema
		relSchema := &codec.SchemaMeta{
			TypeName: "workedAtRel",
			RelType:  "WORKED_AT",
			IsNode:   false,
			Indexes: []codec.IndexDef{
				{Name: "idx_WORKED_AT_role", Type: codec.IndexTypeRange, Label: "WORKED_AT", IsNode: false, Properties: []codec.IndexProperty{{Name: "role"}}},
			},
			Constraints: []codec.ConstraintDef{
				{Name: "notnull_WORKED_AT_start_date", Type: codec.ConstraintTypeNotNull, Label: "WORKED_AT", IsNode: false, Properties: []string{"start_date"}},
			},
		}

		actions := computeMigrationActions(existingIndexes, existingConstraints, []*codec.SchemaMeta{relSchema})

		require.Len(actions, 2)

		// Verify relationship syntax in generated Cypher
		for _, a := range actions {
			require.Contains(a.Cypher, "()-[r:WORKED_AT]-()", "should have relationship pattern")
		}
	})
}

func TestMigrationLogic_CompositeConstraint(t *testing.T) {
	t.Run("handles composite unique constraints", func(t *testing.T) {
		require := require.New(t)

		existingIndexes := []neogo.IndexInfo{}
		existingConstraints := []neogo.ConstraintInfo{}

		schema := &codec.SchemaMeta{
			TypeName: "Company",
			Labels:   []string{"Company"},
			IsNode:   true,
			Constraints: []codec.ConstraintDef{
				{
					Name:       "uniq_company_tenant",
					Type:       codec.ConstraintTypeUnique,
					Label:      "Company",
					IsNode:     true,
					Properties: []string{"tenant_id", "company_id"},
				},
			},
		}

		actions := computeMigrationActions(existingIndexes, existingConstraints, []*codec.SchemaMeta{schema})

		require.Len(actions, 1)
		require.Contains(actions[0].Cypher, "(n.tenant_id, n.company_id)")
		require.Contains(actions[0].Cypher, "IS UNIQUE")
	})
}

func TestMigrationLogic_VectorIndex(t *testing.T) {
	t.Run("handles vector index with options", func(t *testing.T) {
		require := require.New(t)

		existingIndexes := []neogo.IndexInfo{}
		existingConstraints := []neogo.ConstraintInfo{}

		schema := &codec.SchemaMeta{
			TypeName: "Movie",
			Labels:   []string{"Movie"},
			IsNode:   true,
			Indexes: []codec.IndexDef{
				{
					Name:       "vector_Movie_embedding",
					Type:       codec.IndexTypeVector,
					Label:      "Movie",
					IsNode:     true,
					Properties: []codec.IndexProperty{{Name: "embedding"}},
					Options:    map[string]string{"dimensions": "1536", "similarity": "cosine"},
				},
			},
		}

		actions := computeMigrationActions(existingIndexes, existingConstraints, []*codec.SchemaMeta{schema})

		require.Len(actions, 1)
		require.Equal(neogo.MigrationCreateIndex, actions[0].Type)
		require.Contains(actions[0].Cypher, "CREATE VECTOR INDEX")
		require.Contains(actions[0].Cypher, "vector.dimensions`: 1536")
		require.Contains(actions[0].Cypher, "vector.similarity_function`: 'cosine'")
	})
}

func TestMigrationLogic_FulltextIndex(t *testing.T) {
	t.Run("handles fulltext index syntax", func(t *testing.T) {
		require := require.New(t)

		existingIndexes := []neogo.IndexInfo{}
		existingConstraints := []neogo.ConstraintInfo{}

		schema := &codec.SchemaMeta{
			TypeName: "Movie",
			Labels:   []string{"Movie"},
			IsNode:   true,
			Indexes: []codec.IndexDef{
				{
					Name:       "fulltext_Movie_plot",
					Type:       codec.IndexTypeFulltext,
					Label:      "Movie",
					IsNode:     true,
					Properties: []codec.IndexProperty{{Name: "plot"}},
				},
			},
		}

		actions := computeMigrationActions(existingIndexes, existingConstraints, []*codec.SchemaMeta{schema})

		require.Len(actions, 1)
		require.Contains(actions[0].Cypher, "CREATE FULLTEXT INDEX")
		require.Contains(actions[0].Cypher, "ON EACH [n.plot]")
	})
}

func TestMigrationLogic_NodeKeyConstraint(t *testing.T) {
	t.Run("handles node key constraint", func(t *testing.T) {
		require := require.New(t)

		existingIndexes := []neogo.IndexInfo{}
		existingConstraints := []neogo.ConstraintInfo{}

		schema := &codec.SchemaMeta{
			TypeName: "Company",
			Labels:   []string{"Company"},
			IsNode:   true,
			Constraints: []codec.ConstraintDef{
				{
					Name:       "key_Company_org_user",
					Type:       codec.ConstraintTypeNodeKey,
					Label:      "Company",
					IsNode:     true,
					Properties: []string{"org_id", "user_id"},
				},
			},
		}

		actions := computeMigrationActions(existingIndexes, existingConstraints, []*codec.SchemaMeta{schema})

		require.Len(actions, 1)
		require.Contains(actions[0].Cypher, "IS NODE KEY")
		require.Contains(actions[0].Cypher, "(n.org_id, n.user_id)")
	})
}

func TestMigrationLogic_EmptySchema(t *testing.T) {
	t.Run("handles schema with no indexes or constraints", func(t *testing.T) {
		require := require.New(t)

		existingIndexes := []neogo.IndexInfo{}
		existingConstraints := []neogo.ConstraintInfo{}

		// Schema with no indexes or constraints
		schema := &codec.SchemaMeta{
			TypeName:    "EmptyNode",
			Labels:      []string{"EmptyNode"},
			IsNode:      true,
			Indexes:     []codec.IndexDef{},
			Constraints: []codec.ConstraintDef{},
		}

		actions := computeMigrationActions(existingIndexes, existingConstraints, []*codec.SchemaMeta{schema})

		require.Len(actions, 0, "empty schema should produce no actions")
	})
}

func TestMigrationLogic_NilSchema(t *testing.T) {
	t.Run("handles nil schema slice", func(t *testing.T) {
		require := require.New(t)

		existingIndexes := []neogo.IndexInfo{}
		existingConstraints := []neogo.ConstraintInfo{}

		actions := computeMigrationActions(existingIndexes, existingConstraints, nil)

		require.Len(actions, 0, "nil schemas should produce no actions")
	})
}

func TestMigrationLogic_DBHasExtraSchema(t *testing.T) {
	t.Run("ignores extra DB schema not in code (additive only)", func(t *testing.T) {
		require := require.New(t)

		// Database has extra schema that's not in code
		existingIndexes := []neogo.IndexInfo{
			{Name: "idx_Person_email", Type: "RANGE", EntityType: "NODE", State: "ONLINE"},
			{Name: "idx_Person_legacy", Type: "RANGE", EntityType: "NODE", State: "ONLINE"}, // Not in code
		}
		existingConstraints := []neogo.ConstraintInfo{
			{Name: "unique_Person_id", Type: "UNIQUENESS", EntityType: "NODE"},
			{Name: "unique_Person_legacy", Type: "UNIQUENESS", EntityType: "NODE"}, // Not in code
		}

		// Code only defines email index and id constraint
		schema := &codec.SchemaMeta{
			TypeName: "Person",
			Labels:   []string{"Person"},
			IsNode:   true,
			Indexes: []codec.IndexDef{
				{Name: "idx_Person_email", Type: codec.IndexTypeRange, Label: "Person", IsNode: true, Properties: []codec.IndexProperty{{Name: "email"}}},
			},
			Constraints: []codec.ConstraintDef{
				{Name: "unique_Person_id", Type: codec.ConstraintTypeUnique, Label: "Person", IsNode: true, Properties: []string{"id"}},
			},
		}

		actions := computeMigrationActions(existingIndexes, existingConstraints, []*codec.SchemaMeta{schema})

		// Should NOT try to drop the legacy schema elements (additive only)
		require.Len(actions, 0, "additive migration should not drop extra DB schema")
	})
}

// =============================================================================
// Registry Integration Tests
// =============================================================================

func TestSchemaFromRegistry(t *testing.T) {
	t.Run("collects schema from registered nodes", func(t *testing.T) {
		require := require.New(t)
		reg := internal.NewRegistry()
		reg.RegisterTypes(&personNode{}, &movieNode{})

		// Collect schemas from registry (mimicking what NeedsMigration does)
		var schemas []*codec.SchemaMeta
		for _, node := range reg.Nodes {
			if schema := node.Schema(); schema != nil {
				schemas = append(schemas, schema)
			}
		}

		require.Len(schemas, 2)

		// Find Person schema
		var personSchema *codec.SchemaMeta
		for _, s := range schemas {
			if s.TypeName == "personNode" {
				personSchema = s
			}
		}
		require.NotNil(personSchema)
		require.True(personSchema.IsNode)
		require.Contains(personSchema.Labels, "Person")
		require.Len(personSchema.Indexes, 1) // name index
		require.Len(personSchema.Constraints, 2) // email unique + auto ID
	})

	t.Run("collects schema from registered relationships", func(t *testing.T) {
		require := require.New(t)
		reg := internal.NewRegistry()
		reg.RegisterTypes(&workedAtRel{})

		// Collect schemas from registry
		var schemas []*codec.SchemaMeta
		for _, rel := range reg.Relationships {
			if schema := rel.Schema(); schema != nil {
				schemas = append(schemas, schema)
			}
		}

		require.Len(schemas, 1)
		require.Equal("workedAtRel", schemas[0].TypeName)
		require.Equal("WORKED_AT", schemas[0].RelType)
		require.False(schemas[0].IsNode)
		require.Len(schemas[0].Indexes, 1)      // role index
		require.Len(schemas[0].Constraints, 1) // start_date notNull
	})
}

// =============================================================================
// MigrationAction Type Tests
// =============================================================================

func TestMigrationActionTypes(t *testing.T) {
	t.Run("MigrationCreateIndex constant", func(t *testing.T) {
		assert.Equal(t, neogo.MigrationActionType("CREATE_INDEX"), neogo.MigrationCreateIndex)
	})

	t.Run("MigrationCreateConstraint constant", func(t *testing.T) {
		assert.Equal(t, neogo.MigrationActionType("CREATE_CONSTRAINT"), neogo.MigrationCreateConstraint)
	})
}

// =============================================================================
// IndexInfo and ConstraintInfo Tests
// =============================================================================

func TestIndexInfo(t *testing.T) {
	t.Run("IndexInfo fields", func(t *testing.T) {
		idx := neogo.IndexInfo{
			Name:          "idx_test",
			Type:          "RANGE",
			EntityType:    "NODE",
			LabelsOrTypes: []string{"Person"},
			Properties:    []string{"email"},
			State:         "ONLINE",
			Options:       map[string]any{"provider": "lucene+native-3.0"},
		}

		assert.Equal(t, "idx_test", idx.Name)
		assert.Equal(t, "RANGE", idx.Type)
		assert.Equal(t, "NODE", idx.EntityType)
		assert.Equal(t, []string{"Person"}, idx.LabelsOrTypes)
		assert.Equal(t, []string{"email"}, idx.Properties)
		assert.Equal(t, "ONLINE", idx.State)
		assert.NotNil(t, idx.Options)
	})
}

func TestConstraintInfo(t *testing.T) {
	t.Run("ConstraintInfo fields", func(t *testing.T) {
		con := neogo.ConstraintInfo{
			Name:          "unique_test",
			Type:          "UNIQUENESS",
			EntityType:    "NODE",
			LabelsOrTypes: []string{"Person"},
			Properties:    []string{"email"},
		}

		assert.Equal(t, "unique_test", con.Name)
		assert.Equal(t, "UNIQUENESS", con.Type)
		assert.Equal(t, "NODE", con.EntityType)
		assert.Equal(t, []string{"Person"}, con.LabelsOrTypes)
		assert.Equal(t, []string{"email"}, con.Properties)
	})
}

// =============================================================================
// Index Type Constants Tests
// =============================================================================

func TestIndexTypeConstants(t *testing.T) {
	// Verify re-exported constants match codec constants
	assert.Equal(t, codec.IndexTypeRange, neogo.IndexTypeRange)
	assert.Equal(t, codec.IndexTypeText, neogo.IndexTypeText)
	assert.Equal(t, codec.IndexTypePoint, neogo.IndexTypePoint)
	assert.Equal(t, codec.IndexTypeFulltext, neogo.IndexTypeFulltext)
	assert.Equal(t, codec.IndexTypeVector, neogo.IndexTypeVector)
}

func TestConstraintTypeConstants(t *testing.T) {
	// Verify re-exported constants match codec constants
	assert.Equal(t, codec.ConstraintTypeUnique, neogo.ConstraintTypeUnique)
	assert.Equal(t, codec.ConstraintTypeNodeKey, neogo.ConstraintTypeNodeKey)
	assert.Equal(t, codec.ConstraintTypeNotNull, neogo.ConstraintTypeNotNull)
}

// =============================================================================
// Edge Case Tests
// =============================================================================

func TestMigrationLogic_AllIndexTypes(t *testing.T) {
	t.Run("handles all index types together", func(t *testing.T) {
		require := require.New(t)

		existingIndexes := []neogo.IndexInfo{}
		existingConstraints := []neogo.ConstraintInfo{}

		schema := &codec.SchemaMeta{
			TypeName: "AllIndexes",
			Labels:   []string{"AllIndexes"},
			IsNode:   true,
			Indexes: []codec.IndexDef{
				{Name: "idx_range", Type: codec.IndexTypeRange, Label: "AllIndexes", IsNode: true, Properties: []codec.IndexProperty{{Name: "prop1"}}},
				{Name: "idx_text", Type: codec.IndexTypeText, Label: "AllIndexes", IsNode: true, Properties: []codec.IndexProperty{{Name: "prop2"}}},
				{Name: "idx_point", Type: codec.IndexTypePoint, Label: "AllIndexes", IsNode: true, Properties: []codec.IndexProperty{{Name: "prop3"}}},
				{Name: "idx_fulltext", Type: codec.IndexTypeFulltext, Label: "AllIndexes", IsNode: true, Properties: []codec.IndexProperty{{Name: "prop4"}}},
				{Name: "idx_vector", Type: codec.IndexTypeVector, Label: "AllIndexes", IsNode: true, Properties: []codec.IndexProperty{{Name: "prop5"}}, Options: map[string]string{"dimensions": "512"}},
			},
		}

		actions := computeMigrationActions(existingIndexes, existingConstraints, []*codec.SchemaMeta{schema})

		require.Len(actions, 5)

		// Verify each index type generates correct Cypher prefix
		cypherByName := make(map[string]string)
		for _, a := range actions {
			cypherByName[a.Name] = a.Cypher
		}

		assert.Contains(t, cypherByName["idx_range"], "CREATE INDEX")
		assert.NotContains(t, cypherByName["idx_range"], "CREATE TEXT")

		assert.Contains(t, cypherByName["idx_text"], "CREATE TEXT INDEX")

		assert.Contains(t, cypherByName["idx_point"], "CREATE POINT INDEX")

		assert.Contains(t, cypherByName["idx_fulltext"], "CREATE FULLTEXT INDEX")

		assert.Contains(t, cypherByName["idx_vector"], "CREATE VECTOR INDEX")
	})
}

func TestMigrationLogic_AllConstraintTypes(t *testing.T) {
	t.Run("handles all constraint types together", func(t *testing.T) {
		require := require.New(t)

		existingIndexes := []neogo.IndexInfo{}
		existingConstraints := []neogo.ConstraintInfo{}

		schema := &codec.SchemaMeta{
			TypeName: "AllConstraints",
			Labels:   []string{"AllConstraints"},
			IsNode:   true,
			Constraints: []codec.ConstraintDef{
				{Name: "con_unique", Type: codec.ConstraintTypeUnique, Label: "AllConstraints", IsNode: true, Properties: []string{"prop1"}},
				{Name: "con_notnull", Type: codec.ConstraintTypeNotNull, Label: "AllConstraints", IsNode: true, Properties: []string{"prop2"}},
				{Name: "con_nodekey", Type: codec.ConstraintTypeNodeKey, Label: "AllConstraints", IsNode: true, Properties: []string{"prop3", "prop4"}},
			},
		}

		actions := computeMigrationActions(existingIndexes, existingConstraints, []*codec.SchemaMeta{schema})

		require.Len(actions, 3)

		// Verify each constraint type generates correct Cypher
		cypherByName := make(map[string]string)
		for _, a := range actions {
			cypherByName[a.Name] = a.Cypher
		}

		assert.Contains(t, cypherByName["con_unique"], "IS UNIQUE")
		assert.Contains(t, cypherByName["con_notnull"], "IS NOT NULL")
		assert.Contains(t, cypherByName["con_nodekey"], "IS NODE KEY")
	})
}

func TestMigrationLogic_CompositeIndex(t *testing.T) {
	t.Run("handles composite index with multiple properties", func(t *testing.T) {
		require := require.New(t)

		existingIndexes := []neogo.IndexInfo{}
		existingConstraints := []neogo.ConstraintInfo{}

		schema := &codec.SchemaMeta{
			TypeName: "CompositeTest",
			Labels:   []string{"CompositeTest"},
			IsNode:   true,
			Indexes: []codec.IndexDef{
				{
					Name:   "idx_composite",
					Type:   codec.IndexTypeRange,
					Label:  "CompositeTest",
					IsNode: true,
					Properties: []codec.IndexProperty{
						{Name: "first_name", Priority: 1},
						{Name: "last_name", Priority: 2},
						{Name: "middle_name", Priority: 3},
					},
				},
			},
		}

		actions := computeMigrationActions(existingIndexes, existingConstraints, []*codec.SchemaMeta{schema})

		require.Len(actions, 1)
		// Composite index should list all properties
		assert.Contains(t, actions[0].Cypher, "n.first_name")
		assert.Contains(t, actions[0].Cypher, "n.last_name")
		assert.Contains(t, actions[0].Cypher, "n.middle_name")
	})
}

func TestMigrationLogic_MultiPropertyFulltext(t *testing.T) {
	t.Run("handles fulltext index with multiple properties", func(t *testing.T) {
		require := require.New(t)

		existingIndexes := []neogo.IndexInfo{}
		existingConstraints := []neogo.ConstraintInfo{}

		schema := &codec.SchemaMeta{
			TypeName: "FulltextTest",
			Labels:   []string{"FulltextTest"},
			IsNode:   true,
			Indexes: []codec.IndexDef{
				{
					Name:   "ft_search",
					Type:   codec.IndexTypeFulltext,
					Label:  "FulltextTest",
					IsNode: true,
					Properties: []codec.IndexProperty{
						{Name: "title", Priority: 1},
						{Name: "description", Priority: 2},
						{Name: "keywords", Priority: 3},
					},
				},
			},
		}

		actions := computeMigrationActions(existingIndexes, existingConstraints, []*codec.SchemaMeta{schema})

		require.Len(actions, 1)
		// Fulltext uses ON EACH [...] syntax
		assert.Contains(t, actions[0].Cypher, "ON EACH [n.title, n.description, n.keywords]")
	})
}

func TestMigrationLogic_CaseByNameMatching(t *testing.T) {
	t.Run("matching is case-sensitive by name", func(t *testing.T) {
		require := require.New(t)

		// DB has index with different case
		existingIndexes := []neogo.IndexInfo{
			{Name: "IDX_Person_email", Type: "RANGE"}, // Different case
		}
		existingConstraints := []neogo.ConstraintInfo{}

		schema := &codec.SchemaMeta{
			TypeName: "Person",
			Labels:   []string{"Person"},
			IsNode:   true,
			Indexes: []codec.IndexDef{
				{Name: "idx_Person_email", Type: codec.IndexTypeRange, Label: "Person", IsNode: true, Properties: []codec.IndexProperty{{Name: "email"}}},
			},
		}

		actions := computeMigrationActions(existingIndexes, existingConstraints, []*codec.SchemaMeta{schema})

		// Should create index because name is case-sensitive
		require.Len(actions, 1, "case-sensitive name matching should detect difference")
	})
}

func TestMigrationLogic_EmptyProperties(t *testing.T) {
	t.Run("skips index with empty properties", func(t *testing.T) {
		require := require.New(t)

		existingIndexes := []neogo.IndexInfo{}
		existingConstraints := []neogo.ConstraintInfo{}

		schema := &codec.SchemaMeta{
			TypeName: "EmptyProps",
			Labels:   []string{"EmptyProps"},
			IsNode:   true,
			Indexes: []codec.IndexDef{
				{Name: "idx_empty", Type: codec.IndexTypeRange, Label: "EmptyProps", IsNode: true, Properties: []codec.IndexProperty{}},
			},
			Constraints: []codec.ConstraintDef{
				{Name: "con_empty", Type: codec.ConstraintTypeUnique, Label: "EmptyProps", IsNode: true, Properties: []string{}},
			},
		}

		actions := computeMigrationActions(existingIndexes, existingConstraints, []*codec.SchemaMeta{schema})

		// Should skip because GenerateIndexCypher/GenerateConstraintCypher return "" for empty properties
		require.Len(actions, 0, "should skip schema elements with empty properties")
	})
}

// =============================================================================
// getSchemaForType Helper Tests (via registry behavior)
// =============================================================================

func TestSchemaLookupByType(t *testing.T) {
	t.Run("looks up node schema by pointer type", func(t *testing.T) {
		require := require.New(t)
		reg := internal.NewRegistry()
		reg.RegisterTypes(&personNode{})

		// Simulate what getSchemaForType does
		entity := reg.Get(typeOf(&personNode{}))
		require.NotNil(entity)

		node, ok := entity.(*internal.RegisteredNode)
		require.True(ok)

		schema := node.Schema()
		require.NotNil(schema)
		require.Equal("personNode", schema.TypeName)
	})

	t.Run("looks up node schema by value type", func(t *testing.T) {
		require := require.New(t)
		reg := internal.NewRegistry()
		reg.RegisterTypes(&personNode{})

		entity := reg.Get(typeOf(personNode{}))
		require.NotNil(entity)

		node, ok := entity.(*internal.RegisteredNode)
		require.True(ok)

		schema := node.Schema()
		require.NotNil(schema)
	})

	t.Run("returns nil for unregistered type", func(t *testing.T) {
		require := require.New(t)
		reg := internal.NewRegistry()

		entity := reg.Get(typeOf(&personNode{}))
		require.Nil(entity)
	})
}

func typeOf(v any) reflect.Type {
	t := reflect.TypeOf(v)
	if t.Kind() == reflect.Ptr {
		t = t.Elem()
	}
	return t
}
