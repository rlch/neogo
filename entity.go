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
	//   Name string `json:"name"`
	//   Age  int    `json:"age"`
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
	//  	Alive bool `json:"alive"`
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
	//  	Name         string `json:"name"`
	//  }
	//
	//  type Dog struct {
	//  	BaseOrganism `neo4j:"Dog"`
	//  	Borfs        bool `json:"borfs"`
	//  }
	Abstract = internal.Abstract

	// Relationship is a base type for all relationships.
	//
	// The neo4j tag is used to specify the type for the relationship.
	//
	//  type ActedIn struct {
	//  	neogo.Relationship `neo4j:"ACTED_IN"`
	//
	//  	Role string `json:"role"`
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
