package neo4jgorm

import _ "github.com/sanity-io/litter"

func NodeWithID[N any, PN interface {
	idSetter

	*N
}](id string,
) PN {
	n := PN(new(N))
	n.SetID(id)
	return n
}

type INode interface {
	IsNode()
}

type Node struct {
	ID string `json:"id"`
}

func (Node) IsNode() {}

type idSetter interface{ SetID(id string) }

func (n *Node) SetID(id string) {
	n.ID = id
}
