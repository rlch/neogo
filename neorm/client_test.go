package neorm

import (
	"testing"

	"github.com/rlch/neogo/internal"
	"github.com/stretchr/testify/require"
)

func TestLoad(t *testing.T) {
	require := require.New(t)
	adam := Person{}
	adam.ID = "adam"

	t.Run("loads a single node", func(t *testing.T) {
		client := NewTransaction()
		err := client.Find(&adam)
		require.NoError(err)
		require.Equal(adam.Name, "Adam")
	})

	client := NewTransaction()

	client.Save(Person{
		Node: internal.Node{ID: "adam"},
		Cats: []*IsOwner{
			{
				ID:  "alskdjfh",
				Cat: &Cat{},
			},
		},
	})

	err := client.
		Query(
			"r{id}:Cats c{}:.",
			"r.active = false AND c.alive = ?", true,
		).
		Query("Friends {}:.").
		Find(&adam)
		// MATCH (adam:Person {id: $id})
		// CALL {
		//   WITH adam
		//   MATCH (adam)-[r:IS_OWNER]->(c:Cat)
		//   WHERE r.active = false AND c.alive = true
		//   RETURN collect(o) as owners, collect(c) as cats
		// }
		// CALL {
		//   WITH adam
		//   MATCH (adam)-[:FRIENDS_WITH]->(f:Person)
		//   RETURN collect(f) as friends
		// }
		// RETURN adam as person, owners, cats, friends

	err = client.
		Query(":Friends {name}:.").
		Delete(&adam)
		// MATCH (adam:Person {id: $id})
		// CALL {
		//   WITH adam
		//   MATCH (adam)-[v1:FRIENDS_WITH]->(v2:Person)
		//   REMOVE v2.name
		// }
		// DELETE adam

	require.NoError(err)

	for _, cat := range adam.Cats {
		require.Equal(cat.Active, true)
		require.Equal(cat.Cat.Owner.ID, adam.ID)
	}
}
