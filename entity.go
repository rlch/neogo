package neogo

import (
	"github.com/rlch/neogo/internal"
)

// NewNode creates a new node with a random ID.
func NewNode[N any, PN interface {
	INode
	internal.IDSetter
	*N
}]() PN {
	n := PN(new(N))
	n.GenerateID()
	return n
}

// NodeWithID creates a new node with the given ID.
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

type (
	// INode is an interface for nodes.
	// See [Node] for the default implementation.
	INode = internal.INode

	// IAbstract is an interface for abstract nodes.
	// See [Abstract] for the default implementation.
	IAbstract = internal.IAbstract

	// IRelationship is an interface for relationships.
	// See [Relationship] for the default implementation.
	IRelationship = internal.IRelationship

	// Node is a base type for all nodes.
	//
	// The neo4j tag is used to specify the label for the node. Multiple labels
	// may be specified idiomatically by nested [Node] types. See [internal/tests]
	// for examples.
	//
	//  type Person struct {
	//   neogo.Node `neo4j:"Person"`
	//
	//   Name string `neo4j:"name"`
	//   Age  int    `neo4j:"age"`
	//  }
	Node = internal.Node

	// Abstract is a base type for all abstract nodes. An abstract node can have
	// multiple concrete implementers, where each implementer must have a distinct
	// label. This means that each node will have at least 2 labels.
	//
	// A useful design pattern for constructing abstract nodes is to create a base
	// type which provides an implementation for [IAbstract] and embed [Abstract]
	// + [Node], then embed that type in all concrete implementers:
	//
	//  type Organism interface {
	//  	internal.IAbstract
	//  }
	//
	//  type BaseOrganism struct {
	//  	internal.Abstract `neo4j:"Organism"`
	//  	internal.Node
	//
	//  	Alive bool `neo4j:"alive"`
	//  }
	//
	//  func (b BaseOrganism) Implementers() []internal.IAbstract {
	//  	return []internal.IAbstract{
	//  		&Human{},
	//  		&Dog{},
	//  	}
	//  }
	//
	//  type Human struct {
	//  	BaseOrganism `neo4j:"Human"`
	//  	Name         string `neo4j:"name"`
	//  }
	//
	//  type Dog struct {
	//  	BaseOrganism `neo4j:"Dog"`
	//  	Borfs        bool `neo4j:"borfs"`
	//  }
	Abstract = internal.Abstract

	// Relationship is a base type for all relationships.
	//
	// The neo4j tag is used to specify the type for the relationship.
	//
	//  type ActedIn struct {
	//  	neogo.Relationship `neo4j:"ACTED_IN"`
	//
	//  	Role string `neo4j:"role"`
	//  }
	Relationship = internal.Relationship

	// Label is a used to specify a label for a node.
	// This allows for multiple labels to be specified idiomatically.
	//
	//  type Robot struct {
	//  	neogo.Label `neo4j:"Robot"`
	//  }
	Label = internal.Label
)

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
type One[R any] = internal.One[R]

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
type Many[R any] = internal.Many[R]
