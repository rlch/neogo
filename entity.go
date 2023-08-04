package neogo

import "github.com/rlch/neogo/internal"

func NodeWithID[N any, PN interface {
	internal.IDSetter
	*N
}](id string,
) PN {
	n := PN(new(N))
	n.SetID(id)
	return n
}

type (
	INode         = internal.INode
	IAbstract     = internal.IAbstract
	IRelationship = internal.IRelationship

	Node         = internal.NodeEntity
	Abstract     = internal.AbstractEntity
	Relationship = internal.RelationshipEntity
)
