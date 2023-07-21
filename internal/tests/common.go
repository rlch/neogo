package tests

import (
	"reflect"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	neo4jgorm "github.com/rlch/neo4j-gorm"
	"github.com/rlch/neo4j-gorm/internal"
)

func canon(cypher string) string {
	s := strings.TrimSpace(cypher)
	s = strings.ReplaceAll(s, "\t", "")
	return s
}

func check(t *testing.T, cy *internal.CompiledCypher, err error, want internal.CompiledCypher) {
	require.NoError(t, err)
	want.Cypher = canon(want.Cypher)
	if want.Parameters == nil {
		want.Parameters = map[string]any{}
	}
	if want.Bindings == nil {
		want.Bindings = map[string]reflect.Value{}
	}
	require.Equal(t, want.Cypher, cy.Cypher)
	require.Equal(t, want.Parameters, cy.Parameters)
	require.Equal(t, want.Bindings, cy.Bindings)
}

type (
	Movie struct {
		neo4jgorm.Node `neo4j:"Movie"`

		Title    string `json:"title"`
		Released int    `json:"released"`
		Tagline  string `json:"tagline"`
	}
	Person struct {
		neo4jgorm.Node `neo4j:"Person"`

		Name        string  `json:"name"`
		Email       string  `json:"email"`
		Belt        *string `json:"belt"`
		Nationality string  `json:"nationality"`
		Age         int     `json:"age"`
		BornIn      int     `json:"bornIn"`
	}
)

type (
	ActedIn struct {
		neo4jgorm.Relationship `neo4j:"ACTED_IN"`

		Role string `json:"role"`
	}
	Directed struct {
		neo4jgorm.Relationship `neo4j:"DIRECTED"`
	}
	Produced struct {
		neo4jgorm.Relationship `neo4j:"PRODUCED"`
	}
	Wrote struct {
		neo4jgorm.Relationship `neo4j:"WROTE"`
	}
	Reviewed struct {
		neo4jgorm.Relationship `neo4j:"REVIEWED"`

		Rating float64 `json:"rating"`
	}
	Knows struct {
		neo4jgorm.Relationship `neo4j:"KNOWS"`

		Since int `json:"since"`
	}
)
