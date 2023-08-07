package neogo

import (
	"reflect"

	"github.com/rlch/neogo/internal"
)

func NewNode[N any, PN interface {
	INode
	internal.IDSetter
	*N
}]() PN {
	n := PN(new(N))
	n.GenerateID()
	return n
}

func NodeWithID[N any, PN interface {
	INode
	internal.IDSetter
	*N
}](id string,
) PN {
	n := PN(new(N))
	n.SetID(id)
	return n
}

var rAbstract = reflect.TypeOf((*IAbstract)(nil)).Elem()

type (
	INode         = internal.INode
	IAbstract     = internal.IAbstract
	IRelationship = internal.IRelationship

	Node         = internal.NodeEntity
	Abstract     = internal.AbstractEntity
	Relationship = internal.RelationshipEntity
)
