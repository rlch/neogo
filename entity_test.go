package neogo_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/rlch/neogo"
)

func TestNewNode(t *testing.T) {
	n := neogo.NewNode[neogo.Node]()
	assert.NotEmpty(t, n.ID)
}

func TestNodeWithID(t *testing.T) {
	n := neogo.NodeWithID[neogo.Node]("test")
	assert.Equal(t, "test", n.ID)
}
