package internal

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
)

type Alien struct {
	Abstract `neo4j:"Organism"`
	Node     `neo4j:"Alien"`
}

func TestGet(t *testing.T) {
	r := Registry{}
	r.RegisterTypes(&BaseOrganism{}, &ActedIn{})
	t.Run("gets a node", func(t *testing.T) {
		require := require.New(t)
		got := r.Get(Human{})
		fmt.Println(r.registeredTypes)
		require.Equal(
			&RegisteredNode{name: "Human", typ: &Human{}, labels: []string{"Organism", "Human"}},
			got,
		)
	})
	t.Run("gets a pointer to node", func(t *testing.T) {
		r.Get(Human{})
	})
}

func TestGetConcreteImplementation(t *testing.T) {
	t.Run("error when no abstract node found for labels", func(t *testing.T) {
		require := require.New(t)
		r := Registry{}
		impl, err := r.GetConcreteImplementation([]string{"Human", "Organism"})
		require.Nil(impl)
		require.Error(err)
	})

	t.Run("error when no concrete implementation found that satisfies labels", func(t *testing.T) {
		require := require.New(t)
		r := Registry{}
		r.RegisterTypes(&Alien{})
		impl, err := r.GetConcreteImplementation([]string{"Human", "Organism"})
		require.Nil(impl)
		require.Error(err)
	})

	t.Run("finds base type that satisfies labels", func(t *testing.T) {
		require := require.New(t)
		r := Registry{}
		r.RegisterTypes(&BaseOrganism{})
		impl, err := r.GetConcreteImplementation([]string{"Organism"})
		require.NoError(err)
		require.Equal(&BaseOrganism{}, impl.typ)
	})

	t.Run("finds concrete implementation that satisfies labels", func(t *testing.T) {
		require := require.New(t)
		r := Registry{}
		r.RegisterTypes(&BaseOrganism{})
		impl, err := r.GetConcreteImplementation([]string{"Human", "Organism"})
		require.NoError(err)
		require.Equal(&Human{}, impl.typ)
	})
}
