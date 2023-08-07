package internal

import (
	"io"

	"github.com/oklog/ulid/v2"
)

var defaultEntropySource io.Reader

var (
	_ interface {
		INode
		IDSetter
	} = (*NodeEntity)(nil)
	_ IRelationship = (*RelationshipEntity)(nil)
)

func init() {
	// Seed the default entropy source.
	defaultEntropySource = ulid.DefaultEntropy()
}

type INode interface {
	IsNode()
}

type IDSetter interface {
	SetID(id any)
	GenerateID()
}

type NodeEntity struct {
	ID string `json:"id"`
}

func (NodeEntity) IsNode() {}

func (n *NodeEntity) SetID(id any) {
	if s, ok := id.(string); ok {
		n.ID = s
	}
}

func (n *NodeEntity) GenerateID() {
	n.ID = ulid.MustNew(ulid.Now(), defaultEntropySource).String()
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
