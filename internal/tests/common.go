package tests

import (
	"reflect"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/rlch/neogo/internal"
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
		want.Bindings = map[string]reflect.Value{}
	}
	require.Equal(t, want.Cypher, cy.Cypher)
	require.Equal(t, want.Parameters, cy.Parameters)
	require.Equal(t, want.Bindings, cy.Bindings)
}

type (
	Movie struct {
		internal.Node `neo4j:"Movie"`

		Title    string `json:"title"`
		Released int    `json:"released"`
		Tagline  string `json:"tagline"`
	}
	Person struct {
		internal.Node `neo4j:"Person"`

		Name          string  `json:"name"`
		Surname       string  `json:"surname"`
		Position      string  `json:"position"`
		Email         string  `json:"email"`
		Belt          *string `json:"belt"`
		Nationality   string  `json:"nationality"`
		Age           int     `json:"age"`
		BornIn        int     `json:"bornIn"`
		Created       int     `json:"created"`
		LastSeen      int     `json:"lastSeen"`
		Found         bool    `json:"found"`
		ChauffeurName string  `json:"chauffeurName"`
	}

	Company struct {
		internal.Node `neo4j:"Company"`

		Name string `json:"name"`
	}
	Location struct {
		internal.Node `neo4j:"Location"`

		Name string `json:"name"`
	}
)

type (
	ActedIn struct {
		internal.Relationship `neo4j:"ACTED_IN"`

		Role string `json:"role"`
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

		Rating float64 `json:"rating"`
	}
	Knows struct {
		internal.Relationship `neo4j:"KNOWS"`

		Since int `json:"since"`
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
	Alive bool `json:"alive"`
}

type BasePet struct {
	internal.Abstract `neo4j:"Pet"`
	BaseOrganism

	Cute bool `json:"cute"`
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
		&Chimera{},
	}
}

type Human struct {
	BaseOrganism `neo4j:"Human"`
	Name         string `json:"name"`
}

type Dog struct {
	BasePet `neo4j:"Dog"`
	Borfs   bool `json:"borfs"`
}

type CursedOrganism interface {
	Organism
	Masochistic() bool
}

type BaseCursedOrganism struct {
	internal.Abstract `neo4j:"CursedOrganism"`
}

type Chimera struct {
	Human
	BaseCursedOrganism `neo4j:"Chimera"`
}

func (c Chimera) Masochistic() bool {
	return true
}
