// Package internal provides core implementation types and utilities for neogo.
// This package contains the foundational types for nodes, relationships, and
// the Cypher query client used internally by the public API.
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
		ID string `neo4j:"id"`
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
