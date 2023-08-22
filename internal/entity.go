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
	} = (*Node)(nil)
	_ IRelationship = (*Relationship)(nil)
)

func init() {
	// Seed the default entropy source.
	defaultEntropySource = ulid.DefaultEntropy()
}

type INode interface {
	IsNode()
}

type IDSetter interface {
	GetID() string
	SetID(id any)
	GenerateID()
}

type Node struct {
	ID string `json:"id"`
}

func (Node) IsNode() {}

func (n Node) GetID() string {
	return n.ID
}

func (n *Node) SetID(id any) {
	if s, ok := id.(string); ok {
		n.ID = s
	}
}

func (n *Node) GenerateID() {
	n.ID = NewID()
}

func NewID() string {
	return ulid.MustNew(ulid.Now(), defaultEntropySource).String()
}

type IAbstract interface {
	INode
	IsAbstract()
	Implementers() []IAbstract
}

type Abstract struct{}

func (*Abstract) IsAbstract() {}

type IRelationship interface {
	IsRelationship()
}

type Relationship struct{}

func (Relationship) IsRelationship() {}
