package tests

import (
	"reflect"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	neogo "github.com/rlch/neogo"
	"github.com/rlch/neogo/internal"
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
		neogo.Node `neo4j:"Movie"`

		Title    string `json:"title"`
		Released int    `json:"released"`
		Tagline  string `json:"tagline"`
	}
	Person struct {
		neogo.Node `neo4j:"Person"`

		Name         string  `json:"name"`
		Surname      string  `json:"surname"`
		Position     string  `json:"position"`
		Email        string  `json:"email"`
		Belt         *string `json:"belt"`
		Nationality  string  `json:"nationality"`
		Age          int     `json:"age"`
		BornIn       int     `json:"bornIn"`
		Created      int     `json:"created"`
		LastSeen     int     `json:"lastSeen"`
		Found        bool    `json:"found"`
		ChauffeurName string  `json:"chauffeurName"`
	}

	Company struct {
		neogo.Node `neo4j:"Company"`

		Name string `json:"name"`
	}
	Location struct {
		neogo.Node `neo4j:"Location"`

		Name string `json:"name"`
	}
)

type (
	ActedIn struct {
		neogo.Relationship `neo4j:"ACTED_IN"`

		Role string `json:"role"`
	}
	Directed struct {
		neogo.Relationship `neo4j:"DIRECTED"`
	}
	Produced struct {
		neogo.Relationship `neo4j:"PRODUCED"`
	}
	Wrote struct {
		neogo.Relationship `neo4j:"WROTE"`
	}
	Reviewed struct {
		neogo.Relationship `neo4j:"REVIEWED"`

		Rating float64 `json:"rating"`
	}
	Knows struct {
		neogo.Relationship `neo4j:"KNOWS"`

		Since int `json:"since"`
	}
	BornIn struct {
		neogo.Relationship `neo4j:"BORN_IN"`
	}
	WorksAt struct {
		neogo.Relationship `neo4j:"WORKS_AT"`
	}
)
