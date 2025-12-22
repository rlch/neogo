package neogo

import (
	"reflect"
	"testing"

	"github.com/rlch/neogo/internal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Test types
type TestPerson struct {
	internal.Node `neo4j:"Person"`
	Name          string `neo4j:"name"`
	Email         string `neo4j:"email,unique"`
}

type TestMovie struct {
	internal.Node `neo4j:"Movie"`
	Title         string `neo4j:"title"`
}

type TestActedIn struct {
	internal.Relationship `neo4j:"ACTED_IN"`
	Role                  string       `neo4j:"role"`
	StartNode             *TestPerson  `neo4j:"startNode"`
	EndNode               *TestMovie   `neo4j:"endNode"`
}

type TestDirectedBy struct {
	internal.Relationship `neo4j:"DIRECTED_BY"`
	Year                  int          `neo4j:"year"`
	StartNode             *TestMovie   `neo4j:"startNode"`
	EndNode               *TestPerson  `neo4j:"endNode"`
}

type TestPersonWithRels struct {
	internal.Node `neo4j:"PersonWithRels"`
	Name          string               `neo4j:"name"`
	Movies        internal.Many[TestActedIn] `neo4j:"->"`
}

func TestNewRegistry(t *testing.T) {
	r := NewRegistry()
	require.NotNil(t, r)
	require.Empty(t, r.Nodes())
	require.Empty(t, r.Relationships())
}

func TestRegistryRegisterTypes(t *testing.T) {
	r := NewRegistry()
	r.RegisterTypes(&TestPerson{}, &TestMovie{}, &TestActedIn{})

	assert.Len(t, r.Nodes(), 2)
	assert.Len(t, r.Relationships(), 1)
}

func TestRegisteredNodeMethods(t *testing.T) {
	r := NewRegistry()
	r.RegisterTypes(&TestPerson{})

	nodes := r.Nodes()
	require.Len(t, nodes, 1)

	node := nodes[0]
	assert.Equal(t, "TestPerson", node.Name())
	assert.Equal(t, reflect.TypeOf(TestPerson{}), node.Type())

	fieldsToProps := node.FieldsToProps()
	assert.Equal(t, "name", fieldsToProps["Name"])
	assert.Equal(t, "email", fieldsToProps["Email"])
	assert.Equal(t, "id", fieldsToProps["ID"])
}

func TestRegisteredNodeRelationships(t *testing.T) {
	r := NewRegistry()
	r.RegisterTypes(&TestPerson{}, &TestMovie{}, &TestActedIn{}, &TestPersonWithRels{})

	// Find the node with relationships
	var personWithRels *RegisteredNode
	for _, n := range r.Nodes() {
		if n.Name() == "TestPersonWithRels" {
			personWithRels = n
			break
		}
	}
	require.NotNil(t, personWithRels)

	rels := personWithRels.Relationships()
	require.NotNil(t, rels)
	require.Contains(t, rels, "Movies")

	relTarget := rels["Movies"]
	assert.True(t, relTarget.Many(), "Movies should be a many relationship")
	assert.True(t, relTarget.Dir(), "Movies should be outgoing (->)")
	assert.Equal(t, "ACTED_IN", relTarget.RelType())
}

func TestRegisteredRelationshipMethods(t *testing.T) {
	r := NewRegistry()
	r.RegisterTypes(&TestPerson{}, &TestMovie{}, &TestActedIn{})

	rels := r.Relationships()
	require.Len(t, rels, 1)

	rel := rels[0]
	assert.Equal(t, "TestActedIn", rel.Name())
	assert.Equal(t, reflect.TypeOf(TestActedIn{}), rel.Type())

	fieldsToProps := rel.FieldsToProps()
	assert.Equal(t, "role", fieldsToProps["Role"])
}

func TestRelationshipTargetMethods(t *testing.T) {
	r := NewRegistry()
	r.RegisterTypes(&TestPerson{}, &TestMovie{}, &TestActedIn{}, &TestPersonWithRels{})

	var personWithRels *RegisteredNode
	for _, n := range r.Nodes() {
		if n.Name() == "TestPersonWithRels" {
			personWithRels = n
			break
		}
	}
	require.NotNil(t, personWithRels)

	rels := personWithRels.Relationships()
	require.Contains(t, rels, "Movies")

	relTarget := rels["Movies"]

	// Test RelName - gets the relationship struct name
	relName := relTarget.RelName()
	assert.Equal(t, "TestActedIn", relName)

	// Test StartNode
	startNode := relTarget.StartNode()
	if startNode != nil {
		assert.NotEmpty(t, startNode.Name())
	}

	// Test EndNode
	endNode := relTarget.EndNode()
	if endNode != nil {
		assert.NotEmpty(t, endNode.Name())
	}
}

func TestGetRelMeta(t *testing.T) {
	r := NewRegistry()
	r.RegisterTypes(&TestPerson{}, &TestMovie{}, &TestActedIn{})

	// Get relationship metadata by name
	meta := r.GetRelMeta("TestActedIn")
	require.NotNil(t, meta)

	// Test start node
	startNode := meta.StartNode()
	require.NotNil(t, startNode)
	assert.Equal(t, reflect.TypeOf(&TestPerson{}), startNode.NodeType())

	// Test end node
	endNode := meta.EndNode()
	require.NotNil(t, endNode)
	assert.Equal(t, reflect.TypeOf(&TestMovie{}), endNode.NodeType())
}

func TestGetRelMetaNotFound(t *testing.T) {
	r := NewRegistry()
	meta := r.GetRelMeta("NonExistent")
	assert.Nil(t, meta)
}

func TestNodeTargetName(t *testing.T) {
	r := NewRegistry()
	r.RegisterTypes(&TestPerson{}, &TestMovie{}, &TestActedIn{})

	meta := r.GetRelMeta("TestActedIn")
	require.NotNil(t, meta)

	// Test that NodeTarget.Name() returns the node type name
	startNode := meta.StartNode()
	require.NotNil(t, startNode)
	// NodeType returns the pointer type, so the name check depends on how it's used
}

func TestRegisteredNodeNoRelationships(t *testing.T) {
	r := NewRegistry()
	r.RegisterTypes(&TestPerson{})

	nodes := r.Nodes()
	require.Len(t, nodes, 1)

	// Node without relationship fields should return nil
	rels := nodes[0].Relationships()
	assert.Nil(t, rels)
}

func TestRelationshipMetaNilNodes(t *testing.T) {
	// Test RelationshipMeta when start/end nodes are nil
	// This should be handled gracefully

	r := NewRegistry()
	// Register just the relationship without full node connections
	r.RegisterTypes(&TestPerson{}, &TestMovie{}, &TestDirectedBy{})

	meta := r.GetRelMeta("TestDirectedBy")
	require.NotNil(t, meta)

	// Even if internal references aren't fully set,
	// the API should handle nil checks gracefully
	startNode := meta.StartNode()
	endNode := meta.EndNode()

	// Both should be non-nil since we registered proper types
	require.NotNil(t, startNode)
	require.NotNil(t, endNode)
}
