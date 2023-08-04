package internal

type INode interface {
	IsNode()
}

type NodeEntity struct {
	ID string `json:"id"`
}

func (NodeEntity) IsNode() {}

func (n *NodeEntity) SetID(id string) {
	n.ID = id
}

type IAbstract interface {
	INode
	IsAbstract()
	Implementers() []IAbstract
}

type AbstractEntity struct{}

func (*AbstractEntity) IsAbstract() {}

type IRelationship interface {
	IsRelationship()
}

type RelationshipEntity struct{}

func (RelationshipEntity) IsRelationship() {}

type IDSetter interface{ SetID(id string) }
