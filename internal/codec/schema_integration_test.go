package codec_test

import (
	"fmt"
	"testing"

	"github.com/rlch/neogo/internal/codec"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Example node types for schema demo
type DemoNode struct {
	ID string `neo4j:"id"`
}

func (DemoNode) IsNode()         {}
func (n DemoNode) GetID() string { return n.ID }

type DemoPerson struct {
	DemoNode `neo4j:"Person"`
	Name     string `neo4j:"name"`
	Email    string `neo4j:"email,unique"`
}

type DemoMovie struct {
	DemoNode `neo4j:"Movie"`
	Title    string `neo4j:"title,index"`
	Plot     string `neo4j:"plot,fulltext"`
}

func TestAutoIDConstraint(t *testing.T) {
	reg := codec.NewCodecRegistry()

	// Register node types
	reg.RegisterTypes(&DemoPerson{}, &DemoMovie{})

	t.Run("Person gets auto ID constraint", func(t *testing.T) {
		meta := reg.GetNodeMeta("DemoPerson")
		require.NotNil(t, meta)
		require.NotNil(t, meta.Schema)

		// Print for visibility
		fmt.Println("\n=== DemoPerson Schema ===")
		fmt.Printf("Labels: %v\n", meta.Labels)
		fmt.Printf("FieldsToProps: %v\n", meta.FieldsToProps)

		fmt.Println("\nConstraints:")
		for _, con := range meta.Schema.Constraints {
			cypher := con.GenerateConstraintCypher()
			fmt.Printf("  %s\n", cypher)
		}

		// Should have 2 constraints: unique for email + auto unique for ID
		assert.Len(t, meta.Schema.Constraints, 2)

		// Find the ID constraint
		var idConstraint *codec.ConstraintDef
		var emailConstraint *codec.ConstraintDef
		for i := range meta.Schema.Constraints {
			con := &meta.Schema.Constraints[i]
			if len(con.Properties) > 0 {
				switch con.Properties[0] {
				case "id":
					idConstraint = con
				case "email":
					emailConstraint = con
				}
			}
		}

		// Verify ID constraint
		require.NotNil(t, idConstraint, "should have ID constraint")
		assert.Equal(t, codec.ConstraintTypeUnique, idConstraint.Type)
		assert.Equal(t, "Person", idConstraint.Label) // Uses first/most specific label
		assert.Contains(t, idConstraint.Name, "unique_Person_id")

		// Verify email constraint
		require.NotNil(t, emailConstraint, "should have email constraint")
		assert.Equal(t, codec.ConstraintTypeUnique, emailConstraint.Type)
	})

	t.Run("Movie gets auto ID constraint plus explicit index", func(t *testing.T) {
		meta := reg.GetNodeMeta("DemoMovie")
		require.NotNil(t, meta)
		require.NotNil(t, meta.Schema)

		fmt.Println("\n=== DemoMovie Schema ===")
		fmt.Printf("Labels: %v\n", meta.Labels)

		fmt.Println("\nIndexes:")
		for _, idx := range meta.Schema.Indexes {
			cypher := idx.GenerateIndexCypher()
			fmt.Printf("  %s\n", cypher)
		}

		fmt.Println("\nConstraints:")
		for _, con := range meta.Schema.Constraints {
			cypher := con.GenerateConstraintCypher()
			fmt.Printf("  %s\n", cypher)
		}

		// Should have indexes: title (range) + plot (fulltext)
		assert.Len(t, meta.Schema.Indexes, 2)

		// Should have 1 constraint: auto unique for ID
		assert.Len(t, meta.Schema.Constraints, 1)
		assert.Equal(t, "id", meta.Schema.Constraints[0].Properties[0])
	})

}

func TestGeneratedCypherStatements(t *testing.T) {
	reg := codec.NewCodecRegistry()
	reg.RegisterTypes(&DemoPerson{}, &DemoMovie{})

	t.Run("generates valid CREATE CONSTRAINT for ID", func(t *testing.T) {
		meta := reg.GetNodeMeta("DemoPerson")
		require.NotNil(t, meta.Schema)

		// Find ID constraint
		for _, con := range meta.Schema.Constraints {
			if len(con.Properties) > 0 && con.Properties[0] == "id" {
				cypher := con.GenerateConstraintCypher()
				assert.Contains(t, cypher, "CREATE CONSTRAINT")
				assert.Contains(t, cypher, "IF NOT EXISTS")
				assert.Contains(t, cypher, "FOR (n:Person)")
				assert.Contains(t, cypher, "REQUIRE n.id IS UNIQUE")
				return
			}
		}
		t.Fatal("ID constraint not found")
	})

	t.Run("generates valid CREATE INDEX for title", func(t *testing.T) {
		meta := reg.GetNodeMeta("DemoMovie")
		require.NotNil(t, meta.Schema)

		// Find title index
		for _, idx := range meta.Schema.Indexes {
			if len(idx.Properties) > 0 && idx.Properties[0].Name == "title" {
				cypher := idx.GenerateIndexCypher()
				assert.Contains(t, cypher, "CREATE INDEX")
				assert.Contains(t, cypher, "IF NOT EXISTS")
				assert.Contains(t, cypher, "FOR (n:Movie)")
				assert.Contains(t, cypher, "ON (n.title)")
				return
			}
		}
		t.Fatal("title index not found")
	})

	t.Run("generates valid FULLTEXT INDEX for plot", func(t *testing.T) {
		meta := reg.GetNodeMeta("DemoMovie")
		require.NotNil(t, meta.Schema)

		// Find plot fulltext index
		for _, idx := range meta.Schema.Indexes {
			if idx.Type == codec.IndexTypeFulltext {
				cypher := idx.GenerateIndexCypher()
				assert.Contains(t, cypher, "CREATE FULLTEXT INDEX")
				assert.Contains(t, cypher, "IF NOT EXISTS")
				assert.Contains(t, cypher, "FOR (n:Movie)")
				assert.Contains(t, cypher, "ON EACH [n.plot]")
				return
			}
		}
		t.Fatal("fulltext index not found")
	})
}
