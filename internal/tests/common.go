package tests

import (
	"reflect"
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
		want.Bindings = map[string]reflect.Value{}
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

		Title    string `db:"title"`
		Released int    `db:"released"`
		Tagline  string `db:"tagline"`

		ActedIn internal.Many[*ActedIn] `neo4j:"<-" db:"-"`
	}
	Person struct {
		internal.Node `neo4j:"Person"`

		Name          string  `db:"name"`
		Surname       string  `db:"surname"`
		Position      string  `db:"position"`
		Email         string  `db:"email"`
		Belt          *string `db:"belt"`
		Nationality   string  `db:"nationality"`
		Age           int     `db:"age"`
		BornIn        int     `db:"bornIn"`
		Created       int     `db:"created"`
		LastSeen      int     `db:"lastSeen"`
		Found         bool    `db:"found"`
		ChauffeurName string  `db:"chauffeurName"`

		ActedIn internal.Many[*ActedIn] `neo4j:"->" db:"-"`
	}

	Company struct {
		internal.Node `neo4j:"Company"`

		Name string `db:"name"`
	}
	Location struct {
		internal.Node `neo4j:"Location"`

		Name string `db:"name"`
	}
)

type (
	ActedIn struct {
		internal.Relationship `neo4j:"ACTED_IN"`

		Role string `db:"role"`

		Actor *Person `neo4j:"startNode" db:"-"`
		Movie *Movie  `neo4j:"endNode" db:"-"`
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

		Rating float64 `db:"rating"`
	}
	Knows struct {
		internal.Relationship `neo4j:"KNOWS"`

		Since int `db:"since"`
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
	Alive bool `db:"alive"`
}

type BasePet struct {
	internal.Abstract `neo4j:"Pet"`
	BaseOrganism

	Cute bool `db:"cute"`
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
	Name         string `db:"name"`
}

type Dog struct {
	BasePet `neo4j:"Dog"`
	Borfs   bool `db:"borfs"`
}
