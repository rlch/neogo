package internal

import (
	"io"

	"github.com/oklog/ulid/v2"
)

var defaultEntropySource io.Reader

func init() {
	// Seed the default entropy source.
	defaultEntropySource = ulid.DefaultEntropy()
}

type (
	INode interface {
		IsNode()
		GetID() string
	}
	IDSetter interface {
		SetID(id any)
		GenerateID()
	}
	Node struct {
		ID string `json:"id"`
	}

	IAbstract interface {
		INode
		IsAbstract()
		Implementers() []IAbstract
	}
	Abstract      struct{}
	IRelationship interface {
		IsRelationship()
	}
	Relationship struct{}
	Label        struct{}

	Many[T any] struct {
		// V is the target node/relationship type.
		// It can be used to specify fields used in any constructed query.
		//
		// When unmarshalling a result, V will never be populated as it would violate the one/many-to-many
		// constraint. However, nested relationships may be populated in V when the query is non-contiguous.
		// For example, consider the following query:
		// 	(n:Person {id: 1})-[:FRIENDS_WITH]->(:Person)-[:LIKES]->(m:Movie)
		// In this case, the the only data we get from the query is the single Person node and Movie nodes.
		// Assuming the query root node is `n`, then we have that:
		// 	len(n.FriendsWith.V.Likes.S) > 0
		// S will only be populated for qualified nodes and relationships, otherwise V will be used.
		V T
		S []T
	}
)

var (
	_ interface {
		INode
		IDSetter
	} = (*Node)(nil)
	_ IRelationship = (*Relationship)(nil)
)

func (Node) IsNode() {}

func (n Node) GetID() string { return n.ID }

func (n *Node) SetID(id any) {
	if s, ok := id.(string); ok {
		n.ID = s
	}
}

func (n *Node) GenerateID() {
	n.ID = ulid.MustNew(ulid.Now(), defaultEntropySource).String()
}

func (*Abstract) IsAbstract()        {}
func (Relationship) IsRelationship() {}

func (m *Many[T]) Set(v T) T {
	m.V = v
	return v
}
