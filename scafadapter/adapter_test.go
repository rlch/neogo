package scafadapter

import (
	"encoding/json"
	"testing"

	"github.com/rlch/neogo"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Test models for the adapter tests
// Note: Self-referential relationships (e.g., Person -> Person) must go through
// a relationship struct to avoid Go's recursive type limitations.

type Person struct {
	neogo.Node `neo4j:"Person"`

	Name string `neo4j:"name"`
	Born int    `neo4j:"born"`

	// Relationship with struct
	ActedIn neogo.Many[ActedIn] `neo4j:"->"`

	// Shorthand relationship to a different type
	Directed neogo.Many[Movie] `neo4j:"DIRECTED>"`
}

type Movie struct {
	neogo.Node `neo4j:"Movie"`

	Title    string `neo4j:"title"`
	Released int    `neo4j:"released"`
	Tagline  string `neo4j:"tagline"`
}

type ActedIn struct {
	neogo.Relationship `neo4j:"ACTED_IN"`

	Roles []string `neo4j:"roles"`

	Person *Person `neo4j:"startNode"`
	Movie  *Movie  `neo4j:"endNode"`
}

// Separate types to test self-referential relationships via relationship struct
type User struct {
	neogo.Node `neo4j:"User"`

	Name string `neo4j:"name"`

	// Self-referential relationship through a relationship struct
	Following neogo.Many[Follows] `neo4j:"->"`
}

type Follows struct {
	neogo.Relationship `neo4j:"FOLLOWS"`

	Since int `neo4j:"since"`

	Follower *User `neo4j:"startNode"`
	Followed *User `neo4j:"endNode"`
}

func TestAdapter_ExtractSchema_Empty(t *testing.T) {
	adapter := New()
	schema, err := adapter.ExtractSchema()
	require.NoError(t, err)
	assert.NotNil(t, schema)
	assert.Empty(t, schema.Models)
}

func TestAdapter_ExtractSchema_SingleNode(t *testing.T) {
	adapter := New()
	adapter.Register(&Movie{})

	schema, err := adapter.ExtractSchema()
	require.NoError(t, err)

	assert.Len(t, schema.Models, 1)

	movie, ok := schema.Models["Movie"]
	require.True(t, ok, "Movie model should exist")

	assert.Equal(t, "Movie", movie.Name)
	assert.Len(t, movie.Fields, 4) // id, title, released, tagline

	// Check fields exist
	fieldNames := make(map[string]bool)
	for _, f := range movie.Fields {
		fieldNames[f.Name] = true
	}
	assert.True(t, fieldNames["id"], "should have id field")
	assert.True(t, fieldNames["title"], "should have title field")
	assert.True(t, fieldNames["released"], "should have released field")
	assert.True(t, fieldNames["tagline"], "should have tagline field")
}

func TestAdapter_ExtractSchema_NodeWithRelationships(t *testing.T) {
	adapter := New()
	adapter.Register(&Person{}, &Movie{}, &ActedIn{})

	schema, err := adapter.ExtractSchema()
	require.NoError(t, err)

	// Should have Person, Movie, and ActedIn models
	assert.Len(t, schema.Models, 3)

	person, ok := schema.Models["Person"]
	require.True(t, ok, "Person model should exist")

	// Person should have relationships
	assert.NotEmpty(t, person.Relationships, "Person should have relationships")

	// Find relationships
	var actedInRel *Relationship
	var directedRel *Relationship

	for _, rel := range person.Relationships {
		switch rel.Name {
		case "ActedIn":
			actedInRel = rel
		case "Directed":
			directedRel = rel
		}
	}

	// Check ActedIn relationship (uses relationship struct)
	require.NotNil(t, actedInRel, "ActedIn relationship should exist")
	assert.Equal(t, "ACTED_IN", actedInRel.RelType)
	assert.True(t, actedInRel.Many)
	assert.Equal(t, DirectionOutgoing, actedInRel.Direction)

	// Check Directed relationship (shorthand to Movie)
	require.NotNil(t, directedRel, "Directed relationship should exist")
	assert.Equal(t, "DIRECTED", directedRel.RelType)
	assert.True(t, directedRel.Many)
	assert.Equal(t, DirectionOutgoing, directedRel.Direction)
	assert.Equal(t, "Movie", directedRel.Target)
}

func TestAdapter_ExtractSchema_SelfReferentialRelationship(t *testing.T) {
	adapter := New()
	adapter.Register(&User{}, &Follows{})

	schema, err := adapter.ExtractSchema()
	require.NoError(t, err)

	// Should have User and Follows models
	assert.Len(t, schema.Models, 2)

	user, ok := schema.Models["User"]
	require.True(t, ok, "User model should exist")

	// User should have Following relationship
	require.Len(t, user.Relationships, 1)
	followingRel := user.Relationships[0]

	assert.Equal(t, "Following", followingRel.Name)
	assert.Equal(t, "FOLLOWS", followingRel.RelType)
	assert.True(t, followingRel.Many)
	assert.Equal(t, DirectionOutgoing, followingRel.Direction)
}

func TestAdapter_ExtractSchema_RelationshipModel(t *testing.T) {
	adapter := New()
	adapter.Register(&Person{}, &Movie{}, &ActedIn{})

	schema, err := adapter.ExtractSchema()
	require.NoError(t, err)

	actedIn, ok := schema.Models["ActedIn"]
	require.True(t, ok, "ActedIn model should exist")

	assert.Equal(t, "ActedIn", actedIn.Name)

	// ActedIn should have roles field
	var rolesField *Field
	for _, f := range actedIn.Fields {
		if f.Name == "roles" {
			rolesField = f
			break
		}
	}
	require.NotNil(t, rolesField, "ActedIn should have roles field")
	assert.Equal(t, "[]string", rolesField.Type)
}

func TestAdapter_ExtractSchema_FieldTypes(t *testing.T) {
	type TypeTest struct {
		neogo.Node `neo4j:"TypeTest"`

		StringField  string   `neo4j:"string_field"`
		IntField     int      `neo4j:"int_field"`
		BoolField    bool     `neo4j:"bool_field"`
		Float64Field float64  `neo4j:"float64_field"`
		SliceField   []string `neo4j:"slice_field"`
		PtrField     *string  `neo4j:"ptr_field"`
	}

	adapter := New()
	adapter.Register(&TypeTest{})

	schema, err := adapter.ExtractSchema()
	require.NoError(t, err)

	model, ok := schema.Models["TypeTest"]
	require.True(t, ok)

	fieldTypes := make(map[string]string)
	fieldRequired := make(map[string]bool)
	for _, f := range model.Fields {
		fieldTypes[f.Name] = f.Type
		fieldRequired[f.Name] = f.Required
	}

	assert.Equal(t, "string", fieldTypes["string_field"])
	assert.Equal(t, "int", fieldTypes["int_field"])
	assert.Equal(t, "bool", fieldTypes["bool_field"])
	assert.Equal(t, "float64", fieldTypes["float64_field"])
	assert.Equal(t, "[]string", fieldTypes["slice_field"])
	assert.Equal(t, "*string", fieldTypes["ptr_field"])

	// Pointer fields should not be required
	assert.True(t, fieldRequired["string_field"])
	assert.False(t, fieldRequired["ptr_field"])
}

func TestAdapter_ExtractSchema_JSONOutput(t *testing.T) {
	adapter := New()
	adapter.Register(&Person{}, &Movie{}, &ActedIn{})

	schema, err := adapter.ExtractSchema()
	require.NoError(t, err)

	// Verify we can marshal to JSON
	data, err := json.MarshalIndent(schema, "", "  ")
	require.NoError(t, err)

	t.Logf("Schema JSON:\n%s", string(data))

	// Verify we can unmarshal back
	var parsed TypeSchema
	err = json.Unmarshal(data, &parsed)
	require.NoError(t, err)

	assert.Equal(t, len(schema.Models), len(parsed.Models))
}

func TestAdapter_ExtractSchema_OneRelationship(t *testing.T) {
	// Test One[R] (single relationship) extraction
	type Manager struct {
		neogo.Node `neo4j:"Manager"`

		Name string `neo4j:"name"`
	}

	type Employee struct {
		neogo.Node `neo4j:"Employee"`

		Name string `neo4j:"name"`

		// Single relationship to manager
		ReportsTo neogo.One[Manager] `neo4j:"REPORTS_TO>"`
	}

	adapter := New()
	adapter.Register(&Employee{}, &Manager{})

	schema, err := adapter.ExtractSchema()
	require.NoError(t, err)

	employee, ok := schema.Models["Employee"]
	require.True(t, ok)

	require.Len(t, employee.Relationships, 1)
	rel := employee.Relationships[0]

	assert.Equal(t, "ReportsTo", rel.Name)
	assert.Equal(t, "REPORTS_TO", rel.RelType)
	assert.Equal(t, DirectionOutgoing, rel.Direction)
	assert.False(t, rel.Many, "One[R] should have Many=false")
	assert.Equal(t, "Manager", rel.Target)
}

func TestAdapter_ExtractSchema_IncomingRelationship(t *testing.T) {
	// Test incoming relationship using a separate type pair to avoid recursion
	type Comment struct {
		neogo.Node `neo4j:"Comment"`

		Text string `neo4j:"text"`
	}

	type Post struct {
		neogo.Node `neo4j:"Post"`

		Title string `neo4j:"title"`

		// Incoming relationship (Comments point TO this Post)
		Comments neogo.Many[Comment] `neo4j:"<HAS_COMMENT"`
	}

	adapter := New()
	adapter.Register(&Post{}, &Comment{})

	schema, err := adapter.ExtractSchema()
	require.NoError(t, err)

	post, ok := schema.Models["Post"]
	require.True(t, ok)

	require.Len(t, post.Relationships, 1)
	rel := post.Relationships[0]

	assert.Equal(t, "Comments", rel.Name)
	assert.Equal(t, "HAS_COMMENT", rel.RelType)
	assert.Equal(t, DirectionIncoming, rel.Direction)
	assert.True(t, rel.Many)
	assert.Equal(t, "Comment", rel.Target)
}
