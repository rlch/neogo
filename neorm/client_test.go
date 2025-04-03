package neorm

import (
	"testing"

	"github.com/rlch/neogo/db"
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

	err := client.
		Preload(
			"Cats.Cat",
			db.Where(),
		).
		Preload("Cats.Cat.Owner").
		Find(&adam)
	require.NoError(err)

	for _, cat := range adam.Cats {
		require.Equal(cat.Active, true)
		require.Equal(cat.Cat.Owner.ID, adam.ID)
	}
}
