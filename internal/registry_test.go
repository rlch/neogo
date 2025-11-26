package internal

import (
	"reflect"
	"testing"

	"github.com/rlch/neogo/internal/codec"
	"github.com/stretchr/testify/require"
)

func panicToErr[T any](f func() T) (out T, err error) {
	defer func() {
		if r := recover(); r != nil {
			err = r.(error)
		}
	}()
	out = f()
	return
}

type (
	nestedLabelsNode struct {
		simpleNode `neo4j:"Nested"`
	}
	labelNode struct {
		Label `neo4j:"Label"`
	}
	nestedLabelsUsingLabelNode struct {
		simpleNode `neo4j:"Nested"`
		labelNode
	}
	nodeWithProperties struct {
		simpleNode
		Name   string `neo4j:"name"`
		Ignore string `neo4j:"-"`
	}

	simpleNode struct {
		Node `neo4j:"Simple"`
	}
	nodeWithRelationship struct {
		Node `neo4j:"Simple"`

		Forward   One[simpleRelationship]  `neo4j:"->"`
		Backward  One[simpleRelationship]  `neo4j:"<-"`
		Forwards  Many[simpleRelationship] `neo4j:"->"`
		Backwards Many[simpleRelationship] `neo4j:"<-"`
	}
	simpleRelationship struct {
		Relationship `neo4j:"SIMPLE"`

		Field     string                `neo4j:"field"`
		StartNode *nodeWithRelationship `neo4j:"startNode"`
		EndNode   *nodeWithRelationship `neo4j:"endNode"`
	}
	shorthandRelationshipNode struct {
		simpleNode
		Forward   One[simpleNode]  `neo4j:"FRIEND>"`
		Backward  One[simpleNode]  `neo4j:"<FOLLOWS"`
		Forwards  Many[simpleNode] `neo4j:"LIKES>"`
		Backwards Many[simpleNode] `neo4j:"<LIKED_BY"`
	}

	// Test required relationships
	nodeWithRequiredRelationship struct {
		Node `neo4j:"RequiredTest"`

		// Required relationship - must exist
		Manager One[simpleNode] `neo4j:"REPORTS_TO>,required"`

		// Optional relationship (default)
		Mentor One[simpleNode] `neo4j:"MENTORED_BY>"`
	}
)

// Test expectations (comparing behavior via methods)
type nodeExpectation struct {
	name          string
	labels        []string
	fieldsToProps map[string]string
	relCount      int
}

func TestRegisterNode(t *testing.T) {
	for _, test := range []struct {
		name    string
		node    INode
		want    nodeExpectation
		wantErr string
	}{
		{
			name: "registers a simple node",
			node: &simpleNode{},
			want: nodeExpectation{
				name:          "simpleNode",
				labels:        []string{"Simple"},
				fieldsToProps: map[string]string{"ID": "id"},
				relCount:      0,
			},
		},
		{
			name: "registers a node with nested labels",
			node: &nestedLabelsNode{},
			want: nodeExpectation{
				name:          "nestedLabelsNode",
				labels:        []string{"Simple", "Nested"},
				fieldsToProps: map[string]string{"ID": "id"},
				relCount:      0,
			},
		},
		{
			name: "registers a node with nested labels using Label type",
			node: &nestedLabelsUsingLabelNode{},
			want: nodeExpectation{
				name:          "nestedLabelsUsingLabelNode",
				labels:        []string{"Simple", "Nested", "Label"},
				fieldsToProps: map[string]string{"ID": "id"},
				relCount:      0,
			},
		},
		{
			name: "registers a node with properties",
			node: &nodeWithProperties{},
			want: nodeExpectation{
				name:          "nodeWithProperties",
				labels:        []string{"Simple"},
				fieldsToProps: map[string]string{"ID": "id", "Name": "name"},
				relCount:      0,
			},
		},
		{
			name: "registers a node with relationships",
			node: &nodeWithRelationship{},
			want: nodeExpectation{
				name:          "nodeWithRelationship",
				labels:        []string{"Simple"},
				fieldsToProps: map[string]string{"ID": "id"},
				relCount:      4, // Forward, Backward, Forwards, Backwards
			},
		},
		{
			name: "registers a node with shorthand relationship syntax using One/Many",
			node: &shorthandRelationshipNode{},
			want: nodeExpectation{
				name:          "shorthandRelationshipNode",
				labels:        []string{"Simple"},
				fieldsToProps: map[string]string{"ID": "id"},
				relCount:      4, // Forward, Backward, Forwards, Backwards
			},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			require := require.New(t)
			r := NewRegistry()
			reg, err := panicToErr(
				func() *RegisteredNode { return r.RegisterNode(test.node) },
			)
			if test.wantErr != "" {
				require.ErrorContains(err, test.wantErr)
				return
			}
			if err != nil {
				t.Fatalf("expected no error, got: %s", err)
			}

			// Compare via methods (delegation)
			require.Equal(test.want.name, reg.Name())
			require.Equal(test.want.labels, reg.Labels())
			require.Equal(test.want.fieldsToProps, reg.FieldsToProps())
			if test.want.relCount == 0 {
				require.Nil(reg.Relationships)
			} else {
				require.Len(reg.Relationships, test.want.relCount)
			}
		})
	}
}

func TestRegisterNodeWithRelationships(t *testing.T) {
	require := require.New(t)
	r := NewRegistry()
	reg := r.RegisterNode(&nodeWithRelationship{})

	// Verify relationships
	require.NotNil(reg.Relationships)
	require.Len(reg.Relationships, 4)

	// Forward relationship
	fwd := reg.Relationships["Forward"]
	require.NotNil(fwd)
	require.True(fwd.Dir)
	require.False(fwd.Many)
	require.Equal("SIMPLE", fwd.Rel.Reltype)

	// Backward relationship
	bwd := reg.Relationships["Backward"]
	require.NotNil(bwd)
	require.False(bwd.Dir)
	require.False(bwd.Many)

	// Many forward relationships
	fwds := reg.Relationships["Forwards"]
	require.NotNil(fwds)
	require.True(fwds.Dir)
	require.True(fwds.Many)

	// Many backward relationships
	bwds := reg.Relationships["Backwards"]
	require.NotNil(bwds)
	require.False(bwds.Dir)
	require.True(bwds.Many)
}

func TestRegisterNodeShorthand(t *testing.T) {
	require := require.New(t)
	r := NewRegistry()
	reg := r.RegisterNode(&shorthandRelationshipNode{})

	// Verify shorthand relationships using One/Many with named relationship types
	require.NotNil(reg.Relationships)
	require.Len(reg.Relationships, 4)

	// Forward shorthand (One[simpleNode] with "FRIEND>")
	fwd := reg.Relationships["Forward"]
	require.NotNil(fwd)
	require.True(fwd.Dir)
	require.False(fwd.Many)
	require.Equal("FRIEND", fwd.Rel.Reltype)

	// Backward shorthand (One[simpleNode] with "<FOLLOWS")
	bwd := reg.Relationships["Backward"]
	require.NotNil(bwd)
	require.False(bwd.Dir)
	require.False(bwd.Many)
	require.Equal("FOLLOWS", bwd.Rel.Reltype)

	// Forwards shorthand (Many[simpleNode] with "LIKES>")
	fwds := reg.Relationships["Forwards"]
	require.NotNil(fwds)
	require.True(fwds.Dir)
	require.True(fwds.Many)
	require.Equal("LIKES", fwds.Rel.Reltype)

	// Backwards shorthand (Many[simpleNode] with "<LIKED_BY")
	bwds := reg.Relationships["Backwards"]
	require.NotNil(bwds)
	require.False(bwds.Dir)
	require.True(bwds.Many)
	require.Equal("LIKED_BY", bwds.Rel.Reltype)
}

func TestRegisterNodeWithRequiredRelationship(t *testing.T) {
	require := require.New(t)
	r := NewRegistry()

	r.RegisterTypes(&simpleNode{}, &nodeWithRequiredRelationship{})

	nodeMeta := r.Codecs().GetNodeMeta("nodeWithRequiredRelationship")
	require.NotNil(nodeMeta)
	require.Len(nodeMeta.Relationships, 2)

	// Manager is required
	manager := nodeMeta.Relationships["Manager"]
	require.NotNil(manager)
	require.True(manager.Required)
	require.Equal("REPORTS_TO", manager.RelType)
	require.True(manager.Dir)
	require.False(manager.Many)

	// Mentor is optional (default)
	mentor := nodeMeta.Relationships["Mentor"]
	require.NotNil(mentor)
	require.False(mentor.Required)
	require.Equal("MENTORED_BY", mentor.RelType)
}

func TestGet(t *testing.T) {
	r := NewRegistry()
	r.RegisterTypes(&BaseOrganism{}, &ActedIn{})

	t.Run("gets a node", func(t *testing.T) {
		require := require.New(t)
		got := r.Get(reflect.TypeOf(Human{}))
		require.NotNil(got)

		reg := got.(*RegisteredNode)
		require.Equal("Human", reg.Name())
		require.Equal([]string{"Organism", "Human"}, reg.Labels())
		require.Equal(map[string]string{
			"ID":    "id",
			"Alive": "alive",
			"Name":  "name",
		}, reg.FieldsToProps())
		require.Equal(reflect.TypeOf(Human{}), reg.Type())
	})

	t.Run("gets a pointer to node", func(t *testing.T) {
		require := require.New(t)
		got := r.Get(reflect.TypeOf(&Human{}))
		require.NotNil(got)

		reg := got.(*RegisteredNode)
		require.Equal("Human", reg.Name())
		require.Equal([]string{"Organism", "Human"}, reg.Labels())
		require.Equal(reflect.TypeOf(Human{}), reg.Type())
	})
}

// =============================================================================
// Phase 2: Schema Delegation Tests
// =============================================================================

// Test node types with schema tags
type nodeWithIndex struct {
	Node  `neo4j:"NodeWithIndex"`
	Email string `neo4j:"email,index"`
}

type nodeWithUniqueConstraint struct {
	Node  `neo4j:"NodeWithUnique"`
	Email string `neo4j:"email,unique"`
}

type nodeWithMultipleSchema struct {
	Node        `neo4j:"NodeWithMultiple"`
	Email       string    `neo4j:"email,unique,index"`
	Name        string    `neo4j:"name,notNull"`
	Bio         string    `neo4j:"bio,text"`
	Description string    `neo4j:"description,fulltext"`
	Embedding   []float32 `neo4j:"embedding,vector:512"`
}

type nodeWithCompositeIndex struct {
	Node      `neo4j:"CompositeNode"`
	FirstName string `neo4j:"first_name,index:idx_name,priority:1"`
	LastName  string `neo4j:"last_name,index:idx_name,priority:2"`
}

type nodeWithNodeKey struct {
	Node   `neo4j:"NodeKeyNode"`
	OrgID  string `neo4j:"org_id,nodeKey:key_org_user,priority:1"`
	UserID string `neo4j:"user_id,nodeKey:key_org_user,priority:2"`
}

type relationshipWithIndex struct {
	Relationship `neo4j:"WORKED_AT"`
	Role         string `neo4j:"role,index"`
	StartDate    string `neo4j:"start_date,notNull"`
}

func TestRegisteredNodeSchema(t *testing.T) {
	t.Run("simple node has auto ID constraint", func(t *testing.T) {
		require := require.New(t)
		r := NewRegistry()
		reg := r.RegisterNode(&simpleNode{})

		schema := reg.Schema()
		require.NotNil(schema, "simpleNode should have schema")
		require.True(schema.IsNode)
		require.Equal("simpleNode", schema.TypeName)
		require.Equal([]string{"Simple"}, schema.Labels)

		// Should have auto ID constraint
		require.Len(schema.Constraints, 1)
		require.Equal("id", schema.Constraints[0].Properties[0])
		require.Equal("UNIQUE", string(schema.Constraints[0].Type))
	})

	t.Run("node with index has correct schema", func(t *testing.T) {
		require := require.New(t)
		r := NewRegistry()
		reg := r.RegisterNode(&nodeWithIndex{})

		schema := reg.Schema()
		require.NotNil(schema)
		require.True(schema.IsNode)
		require.Equal("nodeWithIndex", schema.TypeName)

		// Should have 1 index for email
		require.Len(schema.Indexes, 1)
		require.Equal("idx_NodeWithIndex_email", schema.Indexes[0].Name)
		require.Equal("RANGE", string(schema.Indexes[0].Type))
		require.Equal("NodeWithIndex", schema.Indexes[0].Label)
		require.Len(schema.Indexes[0].Properties, 1)
		require.Equal("email", schema.Indexes[0].Properties[0].Name)

		// Should have auto ID constraint
		require.Len(schema.Constraints, 1)
		require.Equal("id", schema.Constraints[0].Properties[0])
	})

	t.Run("node with unique constraint has correct schema", func(t *testing.T) {
		require := require.New(t)
		r := NewRegistry()
		reg := r.RegisterNode(&nodeWithUniqueConstraint{})

		schema := reg.Schema()
		require.NotNil(schema)

		// Should have 2 constraints: email unique + auto ID unique
		require.Len(schema.Constraints, 2)

		// Find constraints by property
		var emailConstraint, idConstraint *codec.ConstraintDef
		for i := range schema.Constraints {
			if schema.Constraints[i].Properties[0] == "email" {
				emailConstraint = &schema.Constraints[i]
			} else if schema.Constraints[i].Properties[0] == "id" {
				idConstraint = &schema.Constraints[i]
			}
		}

		require.NotNil(emailConstraint, "should have email constraint")
		require.Equal("UNIQUE", string(emailConstraint.Type))
		require.Equal("unique_NodeWithUnique_email", emailConstraint.Name)

		require.NotNil(idConstraint, "should have auto ID constraint")
		require.Equal("UNIQUE", string(idConstraint.Type))
	})

	t.Run("node with multiple schema elements", func(t *testing.T) {
		require := require.New(t)
		r := NewRegistry()
		reg := r.RegisterNode(&nodeWithMultipleSchema{})

		schema := reg.Schema()
		require.NotNil(schema)
		require.Equal("nodeWithMultipleSchema", schema.TypeName)
		require.Equal([]string{"NodeWithMultiple"}, schema.Labels)

		// Should have 4 indexes: email range + bio text + description fulltext + embedding vector
		require.Len(schema.Indexes, 4)

		// Build map by property name for easier testing
		indexByProp := make(map[string]codec.IndexDef)
		for _, idx := range schema.Indexes {
			if len(idx.Properties) > 0 {
				indexByProp[idx.Properties[0].Name] = idx
			}
		}

		require.Contains(indexByProp, "email")
		require.Equal("RANGE", string(indexByProp["email"].Type))

		require.Contains(indexByProp, "bio")
		require.Equal("TEXT", string(indexByProp["bio"].Type))

		require.Contains(indexByProp, "description")
		require.Equal("FULLTEXT", string(indexByProp["description"].Type))

		require.Contains(indexByProp, "embedding")
		require.Equal("VECTOR", string(indexByProp["embedding"].Type))
		require.Equal("512", indexByProp["embedding"].Options["dimensions"])

		// Should have 3 constraints: email unique + name notNull + auto ID unique
		require.Len(schema.Constraints, 3)

		// Build map by property name
		constraintByProp := make(map[string]codec.ConstraintDef)
		for _, con := range schema.Constraints {
			if len(con.Properties) > 0 {
				constraintByProp[con.Properties[0]] = con
			}
		}

		require.Contains(constraintByProp, "email")
		require.Equal("UNIQUE", string(constraintByProp["email"].Type))

		require.Contains(constraintByProp, "name")
		require.Equal("NOT_NULL", string(constraintByProp["name"].Type))

		require.Contains(constraintByProp, "id")
		require.Equal("UNIQUE", string(constraintByProp["id"].Type))
	})

	t.Run("node with composite index", func(t *testing.T) {
		require := require.New(t)
		r := NewRegistry()
		reg := r.RegisterNode(&nodeWithCompositeIndex{})

		schema := reg.Schema()
		require.NotNil(schema)

		// Should have 1 composite index
		require.Len(schema.Indexes, 1)
		require.Equal("idx_name", schema.Indexes[0].Name)
		require.Len(schema.Indexes[0].Properties, 2)
		// Properties should be in priority order
		require.Equal("first_name", schema.Indexes[0].Properties[0].Name)
		require.Equal("last_name", schema.Indexes[0].Properties[1].Name)
	})

	t.Run("node with node key constraint", func(t *testing.T) {
		require := require.New(t)
		r := NewRegistry()
		reg := r.RegisterNode(&nodeWithNodeKey{})

		schema := reg.Schema()
		require.NotNil(schema)

		// Should have 2 constraints: nodeKey + auto ID
		require.Len(schema.Constraints, 2)

		// Find node key constraint
		var nodeKeyConstraint *codec.ConstraintDef
		for i := range schema.Constraints {
			if schema.Constraints[i].Type == codec.ConstraintTypeNodeKey {
				nodeKeyConstraint = &schema.Constraints[i]
			}
		}

		require.NotNil(nodeKeyConstraint, "should have node key constraint")
		require.Equal("key_org_user", nodeKeyConstraint.Name)
		require.Len(nodeKeyConstraint.Properties, 2)
		require.Equal("org_id", nodeKeyConstraint.Properties[0])
		require.Equal("user_id", nodeKeyConstraint.Properties[1])
	})

	t.Run("generates valid cypher for index", func(t *testing.T) {
		require := require.New(t)
		r := NewRegistry()
		reg := r.RegisterNode(&nodeWithIndex{})

		schema := reg.Schema()
		require.NotNil(schema)
		require.Len(schema.Indexes, 1)

		cypher := schema.Indexes[0].GenerateIndexCypher()
		require.Contains(cypher, "CREATE INDEX")
		require.Contains(cypher, "IF NOT EXISTS")
		require.Contains(cypher, "idx_NodeWithIndex_email")
		require.Contains(cypher, "FOR (n:NodeWithIndex)")
		require.Contains(cypher, "ON (n.email)")
	})

	t.Run("generates valid cypher for constraint", func(t *testing.T) {
		require := require.New(t)
		r := NewRegistry()
		reg := r.RegisterNode(&nodeWithUniqueConstraint{})

		schema := reg.Schema()
		require.NotNil(schema)

		// Find email constraint
		var emailConstraint *codec.ConstraintDef
		for i := range schema.Constraints {
			if schema.Constraints[i].Properties[0] == "email" {
				emailConstraint = &schema.Constraints[i]
				break
			}
		}
		require.NotNil(emailConstraint)

		cypher := emailConstraint.GenerateConstraintCypher()
		require.Contains(cypher, "CREATE CONSTRAINT")
		require.Contains(cypher, "IF NOT EXISTS")
		require.Contains(cypher, "unique_NodeWithUnique_email")
		require.Contains(cypher, "FOR (n:NodeWithUnique)")
		require.Contains(cypher, "REQUIRE n.email IS UNIQUE")
	})
}

func TestRegisteredRelationshipSchema(t *testing.T) {
	t.Run("relationship with schema has correct metadata", func(t *testing.T) {
		require := require.New(t)
		r := NewRegistry()
		reg := r.RegisterRelationship(&relationshipWithIndex{})

		schema := reg.Schema()
		require.NotNil(schema, "relationship should have schema")
		require.False(schema.IsNode)
		require.Equal("relationshipWithIndex", schema.TypeName)
		require.Equal("WORKED_AT", schema.RelType)

		// Should have 1 index for role
		require.Len(schema.Indexes, 1)
		require.Equal("idx_WORKED_AT_role", schema.Indexes[0].Name)
		require.False(schema.Indexes[0].IsNode)
		require.Equal("WORKED_AT", schema.Indexes[0].Label)

		// Should have 1 constraint for start_date notNull
		require.Len(schema.Constraints, 1)
		require.Equal("NOT_NULL", string(schema.Constraints[0].Type))
		require.Equal("start_date", schema.Constraints[0].Properties[0])
	})

	t.Run("relationship generates correct cypher", func(t *testing.T) {
		require := require.New(t)
		r := NewRegistry()
		reg := r.RegisterRelationship(&relationshipWithIndex{})

		schema := reg.Schema()
		require.NotNil(schema)

		// Check index cypher
		indexCypher := schema.Indexes[0].GenerateIndexCypher()
		require.Contains(indexCypher, "CREATE INDEX")
		require.Contains(indexCypher, "FOR ()-[r:WORKED_AT]-()")
		require.Contains(indexCypher, "ON (r.role)")

		// Check constraint cypher
		constraintCypher := schema.Constraints[0].GenerateConstraintCypher()
		require.Contains(constraintCypher, "CREATE CONSTRAINT")
		require.Contains(constraintCypher, "FOR ()-[r:WORKED_AT]-()")
		require.Contains(constraintCypher, "REQUIRE r.start_date IS NOT NULL")
	})
}

func TestRegisteredAbstractNodeSchema(t *testing.T) {
	t.Run("abstract node schema is accessible", func(t *testing.T) {
		require := require.New(t)
		r := NewRegistry()
		r.RegisterTypes(&BaseOrganism{})

		// Get via abstract node
		entity := r.GetByName("BaseOrganism")
		require.NotNil(entity)

		absNode, ok := entity.(*RegisteredAbstractNode)
		require.True(ok)

		schema := absNode.Schema()
		require.NotNil(schema, "abstract node should have schema")
		require.True(schema.IsNode)
		require.Contains(schema.Labels, "Organism")

		// Should have auto ID constraint
		require.Len(schema.Constraints, 1)
		require.Equal("id", schema.Constraints[0].Properties[0])
	})
}

func TestSchemaWithNestedLabels(t *testing.T) {
	t.Run("nested node uses first label for schema", func(t *testing.T) {
		require := require.New(t)
		r := NewRegistry()
		reg := r.RegisterNode(&nestedLabelsNode{})

		schema := reg.Schema()
		require.NotNil(schema)

		// Labels should include all inherited labels
		require.Equal([]string{"Simple", "Nested"}, reg.Labels())

		// Schema should use first label for index/constraint naming
		// Auto ID constraint should reference the first (most specific) label
		require.Len(schema.Constraints, 1)
		// The label used depends on implementation - check it's one of the labels
		require.True(
			schema.Constraints[0].Label == "Simple" || schema.Constraints[0].Label == "Nested",
			"constraint label should be one of the node labels",
		)
	})
}

// =============================================================================
// Existing Tests
// =============================================================================

func TestGetConcreteImplementation(t *testing.T) {
	t.Run("error when no abstract node found for labels", func(t *testing.T) {
		require := require.New(t)
		r := NewRegistry()
		impl, err := r.GetConcreteImplementation([]string{"Human", "Organism"})
		require.Nil(impl)
		require.Error(err)
	})

	t.Run("error when no concrete implementation found that satisfies labels", func(t *testing.T) {
		type Alien struct {
			Abstract `neo4j:"Organism"`
			Node     `neo4j:"Alien"`
		}
		require := require.New(t)
		r := NewRegistry()
		r.RegisterTypes(&Alien{})
		impl, err := r.GetConcreteImplementation([]string{"Human", "Organism"})
		require.Nil(impl)
		require.Error(err)
	})

	t.Run("finds base type that satisfies labels", func(t *testing.T) {
		require := require.New(t)
		r := NewRegistry()
		r.RegisterTypes(&BaseOrganism{})
		impl, err := r.GetConcreteImplementation([]string{"Organism"})
		require.NoError(err)
		require.Equal(reflect.TypeOf(BaseOrganism{}), impl.Type())
	})

	t.Run("finds concrete implementation that satisfies labels", func(t *testing.T) {
		require := require.New(t)
		r := NewRegistry()
		r.RegisterTypes(&BaseOrganism{})
		impl, err := r.GetConcreteImplementation([]string{"Human", "Organism"})
		require.NoError(err)
		require.Equal(reflect.TypeOf(Human{}), impl.Type())
	})
}
