package internal

import (
	"testing"

	require "github.com/stretchr/testify/assert"
)

type person struct {
	Node `neo4j:"Person"`
}

type swedishPerson struct {
	person `neo4j:"Swedish"`
}

type swedishRobot struct {
	swedishPerson
	robot
}

type foreignPerson struct {
	swedishPerson `neo4j:"Foreign"`
}

type personWithNonStructLabels struct {
	swedishPerson
	Name string `json:"name"`
}

type personWithAnonymousStructLabels struct {
	swedishPerson
	S swedishPerson `neo4j:"Concrete"`
}

type robot struct {
	Label `neo4j:"Robot"`
}

func TestExtractNodeLabel(t *testing.T) {
	r := NewRegistry()
	r.RegisterTypes(
		&person{},
		&swedishPerson{},
		&swedishRobot{},
		&foreignPerson{},
		&personWithNonStructLabels{},
		&personWithAnonymousStructLabels{},
		&robot{},
		&BaseOrganism{},
	)
	t.Run("nil when node nil", func(t *testing.T) {
		require.Nil(t, r.ExtractNodeLabels(nil))
	})

	t.Run("extracts node label", func(t *testing.T) {
		require.Equal(t, []string{"Person"}, r.ExtractNodeLabels(person{}))
	})

	t.Run("postpends nested node labels", func(t *testing.T) {
		require.Equal(t, []string{"Person", "Swedish", "Foreign"}, r.ExtractNodeLabels(foreignPerson{}))
	})

	t.Run("extract node label from slice of node", func(t *testing.T) {
		require.Equal(t, []string{"Person"}, r.ExtractNodeLabels([]*person{}))
	})

	t.Run("extract node label from pointer to slice of node", func(t *testing.T) {
		require.Equal(t, []string{"Person"}, r.ExtractNodeLabels(&[]*person{}))
	})

	t.Run("only extract the labels from structs", func(t *testing.T) {
		require.Equal(t, []string{"Person", "Swedish"}, r.ExtractNodeLabels(personWithNonStructLabels{}))
	})

	t.Run("only extract the labels from anonymous structs", func(t *testing.T) {
		require.Equal(t, []string{"Person", "Swedish"}, r.ExtractNodeLabels(personWithAnonymousStructLabels{}))
	})

	t.Run("extracts from abstract types", func(t *testing.T) {
		var o Organism = &BaseOrganism{}
		require.Equal(t, []string{"Organism"}, r.ExtractNodeLabels(o))
	})

	t.Run("extracts from pointers to abstract types", func(t *testing.T) {
		var o Organism = &BaseOrganism{}
		o1 := &o
		o2 := &o1
		require.Equal(t, []string{"Organism"}, r.ExtractNodeLabels(o2))
	})

	t.Run("extracts from structs embedding Label, ordered by DFS", func(t *testing.T) {
		require.Equal(
			t,
			[]string{"Person", "Swedish", "Robot"},
			r.ExtractNodeLabels(&swedishRobot{}),
		)
	})
}

type friendship struct {
	Relationship `neo4j:"Friendship"`
}

type family struct {
	Relationship `neo4j:"Family"`
}

func TestExtractRelationshipType(t *testing.T) {
	r := NewRegistry()
	r.RegisterTypes(
		person{},
		friendship{},
		family{},
		swedishPerson{},
		foreignPerson{},
		personWithNonStructLabels{},
		personWithAnonymousStructLabels{},
		BaseOrganism{},
		robot{},
	)

	t.Run("empty string when relationship nil", func(t *testing.T) {
		require.Equal(t, "", r.ExtractRelationshipType(nil))
	})

	t.Run("extracts relationship type", func(t *testing.T) {
		require.Equal(t, "Friendship", r.ExtractRelationshipType(friendship{}))
	})

	t.Run("panic on multiple relationship types", func(t *testing.T) {
		typ := r.ExtractRelationshipType([]interface{}{friendship{}, family{}})
		require.Equal(t, "", typ)
	})

	t.Run("empty string when relationship type is not found", func(t *testing.T) {
		require.Equal(t, "", r.ExtractRelationshipType(person{}))
	})

	t.Run("extract relationship type from slice of relationship", func(t *testing.T) {
		require.Equal(t, "Friendship", r.ExtractRelationshipType([]*friendship{}))
	})

	t.Run("extract relationship type from pointer to slice of relationship", func(t *testing.T) {
		require.Equal(t, "Friendship", r.ExtractRelationshipType(&[]*friendship{}))
	})
}
