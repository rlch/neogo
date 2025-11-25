// Package tests provides shared test utilities, fixtures, and helper functions
// for testing neogo's Cypher query generation and struct mapping.
package tests

import (
	"strings"
	"testing"

	"github.com/rlch/neogo/internal"
	"github.com/stretchr/testify/require"
)

func canon(cypher string) string {
	s := strings.TrimSpace(cypher)
	s = strings.ReplaceAll(s, "\t", "")
	return s
}

func Check(t *testing.T, cy *internal.CompiledCypher, err error, want internal.CompiledCypher) {
	require.NoError(t, err)
	want.Cypher = canon(want.Cypher)
	if want.Parameters == nil {
		want.Parameters = map[string]any{}
	}
	if want.Bindings == nil {
		want.Bindings = map[string]any{}
	}
	require.Equal(t, want.Cypher, cy.Cypher)
	require.Equal(t, want.Parameters, cy.Parameters)
	require.Equal(t, want.Bindings, cy.Bindings)
}

func CheckCypher(t *testing.T, cy *internal.CompiledCypher, err error, want string) {
	require.NoError(t, err)
	want = canon(want)
	require.Equal(t, want, cy.Cypher)
}

var r = internal.NewRegistry()

func init() {
	r.RegisterTypes(
		&Movie{},
		&Person{},
		&Company{},
		&Location{},
		&ActedIn{},
		&Directed{},
		&Produced{},
		&Wrote{},
		&Reviewed{},
		&Knows{},
		&BornIn{},
		&WorksAt{},
		&BaseOrganism{},
		&BasePet{},
		&Human{},
		&Dog{},
	)
}

type (
	Movie struct {
		internal.Node `neo4j:"Movie"`

		Title    string `neo4j:"title"`
		Released int    `neo4j:"released"`
		Tagline  string `neo4j:"tagline"`

		ActedIn internal.Many[*ActedIn] `neo4j:"<-"`
	}
	Person struct {
		internal.Node `neo4j:"Person"`

		Name          string  `neo4j:"name"`
		Surname       string  `neo4j:"surname"`
		Position      string  `neo4j:"position"`
		Email         string  `neo4j:"email"`
		Belt          *string `neo4j:"belt"`
		Nationality   string  `neo4j:"nationality"`
		Age           int     `neo4j:"age"`
		BornIn        int     `neo4j:"bornIn"`
		Created       int     `neo4j:"created"`
		LastSeen      int     `neo4j:"lastSeen"`
		Found         bool    `neo4j:"found"`
		ChauffeurName string  `neo4j:"chauffeurName"`

		ActedIn internal.Many[*ActedIn] `neo4j:"->"`
	}

	Company struct {
		internal.Node `neo4j:"Company"`

		Name string `neo4j:"name"`
	}
	Location struct {
		internal.Node `neo4j:"Location"`

		Name string `neo4j:"name"`
	}
)

type (
	ActedIn struct {
		internal.Relationship `neo4j:"ACTED_IN"`

		Role string `neo4j:"role"`

		Actor *Person `neo4j:"startNode"`
		Movie *Movie  `neo4j:"endNode"`
	}
	Directed struct {
		internal.Relationship `neo4j:"DIRECTED"`
	}
	Produced struct {
		internal.Relationship `neo4j:"PRODUCED"`
	}
	Wrote struct {
		internal.Relationship `neo4j:"WROTE"`
	}
	Reviewed struct {
		internal.Relationship `neo4j:"REVIEWED"`

		Rating float64 `neo4j:"rating"`
	}
	Knows struct {
		internal.Relationship `neo4j:"KNOWS"`

		Since int `neo4j:"since"`
	}
	BornIn struct {
		internal.Relationship `neo4j:"BORN_IN"`
	}
	WorksAt struct {
		internal.Relationship `neo4j:"WORKS_AT"`
	}
)

type Organism interface {
	internal.IAbstract
}

type Pet interface {
	internal.IAbstract
	Organism
	IsCute() bool
}

type BaseOrganism struct {
	internal.Abstract `neo4j:"Organism"`
	internal.Node
	Alive bool `neo4j:"alive"`
}

type BasePet struct {
	internal.Abstract `neo4j:"Pet"`
	BaseOrganism

	Cute bool `neo4j:"cute"`
}

func (b BasePet) IsCute() bool {
	return b.Cute
}

func (b BasePet) Implementers() []internal.IAbstract {
	return []internal.IAbstract{
		&Dog{},
	}
}

func (b BaseOrganism) Implementers() []internal.IAbstract {
	return []internal.IAbstract{
		&Human{},
		&Dog{},
	}
}

type Human struct {
	BaseOrganism `neo4j:"Human"`
	Name         string `neo4j:"name"`
}

type Dog struct {
	BasePet `neo4j:"Dog"`
	Borfs   bool `neo4j:"borfs"`
}
