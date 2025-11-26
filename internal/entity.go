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

	// One represents a single relationship to another node or through a relationship struct.
	// This is a zero-cost abstraction - the struct has zero size (0 bytes) and only carries
	// type information at compile time for schema extraction and type-safe query building.
	//
	// Note: Place One/Many fields before other fields in your struct to avoid alignment padding.
	// Go adds padding when zero-sized types are the last field in a struct.
	//
	// Usage with relationship struct:
	//
	//	type Person struct {
	//		neogo.Node `neo4j:"Person"`
	//		BestRole One[ActedIn] `neo4j:"->"`
	//	}
	//
	// Usage with shorthand syntax (direct node reference with relationship type in tag):
	//
	//	type Person struct {
	//		neogo.Node `neo4j:"Person"`
	//		BestFriend One[Person] `neo4j:"BEST_FRIEND>"`  // Outgoing BEST_FRIEND
	//		Mentor     One[Person] `neo4j:"<MENTORS"`      // Incoming MENTORS
	//	}
	//
	// Use "required" option to mark relationships that must exist:
	//
	//	Manager One[Person] `neo4j:"REPORTS_TO>,required"`
	One[R any] struct {
		_ [0]R // Zero-sized phantom field for type information
	}

	// Many represents multiple relationships to other nodes or through relationship structs.
	// This is a zero-cost abstraction - the struct has zero size (0 bytes) and only carries
	// type information at compile time for schema extraction and type-safe query building.
	//
	// Note: Place One/Many fields before other fields in your struct to avoid alignment padding.
	// Go adds padding when zero-sized types are the last field in a struct.
	//
	// Usage with relationship struct:
	//
	//	type Person struct {
	//		neogo.Node `neo4j:"Person"`
	//		ActedIn Many[ActedIn] `neo4j:"->"`
	//	}
	//
	// Usage with shorthand syntax (direct node reference with relationship type in tag):
	//
	//	type Person struct {
	//		neogo.Node `neo4j:"Person"`
	//		Friends   Many[Person] `neo4j:"FRIENDS>"`   // Outgoing FRIENDS
	//		Followers Many[Person] `neo4j:"<FOLLOWS"`   // Incoming FOLLOWS
	//	}
	//
	// Use "required" option to mark relationships that must have at least one:
	//
	//	Skills Many[Skill] `neo4j:"HAS_SKILL>,required"`
	Many[R any] struct {
		_ [0]R // Zero-sized phantom field for type information
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
