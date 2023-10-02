package internal

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

type organism interface {
	IAbstract
}

type baseOrganism struct {
	Abstract `neo4j:"Organism"`
	Node
}

func (baseOrganism) Implementers() []IAbstract {
	return []IAbstract{}
}

type person struct {
	Node `neo4j:"Person"`
}

type swedishPerson struct {
	person `neo4j:"Swedish"`
}

type foreignPerson struct {
	swedishPerson `neo4j:"Foreign"`
}

type personWithNonStructLabels struct {
	swedishPerson
	Name string `neo4j:"Name"`
}

func TestExtractNodeLabel(t *testing.T) {
	t.Run("nil when node nil", func(t *testing.T) {
		assert.Nil(t, ExtractNodeLabels(nil))
	})

	t.Run("extracts node label", func(t *testing.T) {
		assert.Equal(t, []string{"Person"}, ExtractNodeLabels(person{}))
	})

	t.Run("postpends nested node labels", func(t *testing.T) {
		assert.Equal(t, []string{"Person", "Swedish", "Foreign"}, ExtractNodeLabels(foreignPerson{}))
	})

	t.Run("extract node label from slice of node", func(t *testing.T) {
		assert.Equal(t, []string{"Person"}, ExtractNodeLabels([]*person{}))
	})

	t.Run("extract node label from pointer to slice of node", func(t *testing.T) {
		assert.Equal(t, []string{"Person"}, ExtractNodeLabels(&[]*person{}))
	})

	t.Run("only extract the labels from structs", func(t *testing.T) {
		assert.Equal(t, []string{"Person", "Swedish"}, ExtractNodeLabels(personWithNonStructLabels{}))
	})

	t.Run("extracts from abstract types", func(t *testing.T) {
		var o organism = &baseOrganism{}
		assert.Equal(t, []string{"Organism"}, ExtractNodeLabels(o))
	})

	t.Run("extracts from pointers to abstract types", func(t *testing.T) {
		var o organism = &baseOrganism{}
		o1 := &o
		o2 := &o1
		assert.Equal(t, []string{"Organism"}, ExtractNodeLabels(o2))
	})
}

type friendship struct {
	Relationship `neo4j:"Friendship"`
}

type family struct {
	Relationship `neo4j:"Family"`
}

func TestExtractRelationshipType(t *testing.T) {
	t.Run("empty string when relationship nil", func(t *testing.T) {
		assert.Equal(t, "", ExtractRelationshipType(nil))
	})

	t.Run("extracts relationship type", func(t *testing.T) {
		assert.Equal(t, "Friendship", ExtractRelationshipType(friendship{}))
	})

	t.Run("panic on multiple relationship types", func(t *testing.T) {
		typ := ExtractRelationshipType([]interface{}{friendship{}, family{}})
		assert.Equal(t, "", typ)
	})

	t.Run("empty string when relationship type is not found", func(t *testing.T) {
		assert.Equal(t, "", ExtractRelationshipType(person{}))
	})

	t.Run("extract relationship type from slice of relationship", func(t *testing.T) {
		assert.Equal(t, "Friendship", ExtractRelationshipType([]*friendship{}))
	})

	t.Run("extract relationship type from pointer to slice of relationship", func(t *testing.T) {
		assert.Equal(t, "Friendship", ExtractRelationshipType(&[]*friendship{}))
	})
}
