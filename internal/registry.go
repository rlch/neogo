package internal

import (
	"fmt"
	"reflect"
	"strings"

	"github.com/rlch/neogo/internal/codec"
)

type (
	Registry struct {
		AbstractNodes   []*RegisteredAbstractNode
		Nodes           []*RegisteredNode
		Relationships   []*RegisteredRelationship
		registeredTypes map[string]RegisteredEntity
		codecs          *codec.CodecRegistry
	}
	RegisteredEntity interface {
		Name() string
		Type() reflect.Type
		FieldsToProps() map[string]string
	}
	RegisteredAbstractNode struct {
		*RegisteredNode
		Implementers []*RegisteredNode
	}
	RegisteredNode struct {
		name          string
		rType         reflect.Type
		fieldsToProps map[string]string

		Labels        []string
		Relationships map[string]*RelationshipTarget
	}
	RegisteredRelationship struct {
		name     string
		typeName string // Store type name instead of reflect.Type

		Reltype   string
		StartNode NodeTarget
		EndNode   NodeTarget
	}
	NodeTarget struct {
		Field string
		*RegisteredNode
	}
	RelationshipTarget struct {
		// Many will be true if there is a (one/many)-to-many relationship between the source and target node.
		Many bool
		// true = ->, false = <-
		Dir bool
		Rel *RegisteredRelationship
	}
)

func (r *RegisteredNode) Name() string {
	return r.name
}

func (r *RegisteredNode) Type() reflect.Type {
	return r.rType
}

func (r *RegisteredNode) ReflectType() any {
	return r.rType
}

func (r *RegisteredNode) FieldsToProps() map[string]string {
	return r.fieldsToProps
}

func (r *RegisteredRelationship) Name() string {
	return r.name
}

func (r *RegisteredRelationship) Type() reflect.Type {
	// This method is deprecated and should not be used
	// Use codec registry instead for type information
	panic("RegisteredRelationship.Type() is deprecated - use codec registry")
}

func (r *RegisteredRelationship) FieldsToProps() map[string]string {
	// This method is deprecated and should not be used
	// Use codec registry instead for field mapping
	panic("RegisteredRelationship.FieldsToProps() is deprecated - use codec registry")
}

func (r RelationshipTarget) Target() *NodeTarget {
	if r.Dir {
		return &r.Rel.EndNode
	} else {
		return &r.Rel.StartNode
	}
}

var (
	rAbstract      = reflect.TypeOf((*IAbstract)(nil)).Elem()
	rINode         = reflect.TypeOf((*INode)(nil)).Elem()
	rIRelationship = reflect.TypeOf((*IRelationship)(nil)).Elem()
	rNode          = reflect.TypeOf(Node{})
)

func NewRegistry() *Registry {
	return &Registry{
		AbstractNodes:   []*RegisteredAbstractNode{},
		Nodes:           []*RegisteredNode{},
		Relationships:   []*RegisteredRelationship{},
		registeredTypes: make(map[string]RegisteredEntity),
		codecs:          codec.NewCodecRegistry(),
	}
}

func (r *Registry) RegisterTypes(types ...any) {
	// IMPORTANT: Register types with codec registry FIRST
	// This must happen before individual RegisterType calls
	r.codecs.RegisterTypes(types...)

	// Then register with legacy registry
	for _, t := range types {
		r.RegisterType(t)
	}
}

// Codecs returns the codec registry for high-performance serialization
func (r *Registry) Codecs() *codec.CodecRegistry {
	return r.codecs
}

func (r *Registry) RegisterType(typ any) (registered any) {
	if abs, ok := typ.(IAbstract); ok {
		return r.RegisterAbstractNode(typ, abs)
	} else if v, ok := typ.(INode); ok {
		return r.RegisterNode(v)
	} else if v, ok := typ.(IRelationship); ok {
		return r.RegisterRelationship(v)
	}
	return
}

func (r *Registry) RegisterNode(v INode) *RegisteredNode {
	vv := codec.UnwindValue(reflect.ValueOf(v))
	vvt := vv.Type()
	name := vvt.Name()
	if n, ok := r.registeredTypes[name]; ok {
		if reg, ok := n.(*RegisteredNode); ok {
			return reg
		} else {
			return n.(*RegisteredAbstractNode).RegisteredNode
		}
	}

	// Extract metadata using zero-reflection codec system
	nodeMeta, err := r.codecs.ExtractNeo4jNodeMeta(v)
	if err != nil {
		panic(fmt.Errorf("failed to extract Neo4j metadata for %s: %w", name, err))
	}

	registered := &RegisteredNode{
		rType:         vvt,
		name:          name,
		Labels:        nodeMeta.Labels,
		fieldsToProps: nodeMeta.FieldsToProps,
		Relationships: make(map[string]*RelationshipTarget),
	}

	r.Nodes = append(r.Nodes, registered)
	r.registeredTypes[name] = registered

	// Convert Neo4j relationship targets to registry relationship targets
	for fieldName, neoRel := range nodeMeta.Relationships {
		var relReg *RegisteredRelationship

		// Create target instance and check its type
		targetInstance, err := r.codecs.CreateNodeInstance(neoRel.NodeType)
		if err != nil {
			panic(fmt.Errorf("failed to create target instance: %w", err))
		}

		// Check if it's actually a relationship type (not a node)
		if _, isRel := targetInstance.(IRelationship); isRel {
			// This is a relationship - analyze its struct to find StartNode/EndNode
			relTypeName := neoRel.NodeType.Elem().Name()
			relReg = &RegisteredRelationship{
				name:     relTypeName,
				typeName: relTypeName,
				Reltype:  neoRel.RelType,
			}

			// Analyze the relationship struct to find StartNode/EndNode fields
			relStruct := neoRel.NodeType.Elem()
			for i := 0; i < relStruct.NumField(); i++ {
				relField := relStruct.Field(i)
				// Support both db and neo4j tags
				var tag string
				if tag = relField.Tag.Get("db"); tag == "" {
					tag = relField.Tag.Get("neo4j")
				}

				if tag == "startNode" {
					// This field represents the start node
					relReg.StartNode = NodeTarget{
						Field:          relField.Name,
						RegisteredNode: registered, // The current node being processed
					}
				} else if tag == "endNode" {
					// This field represents the end node
					relReg.EndNode = NodeTarget{
						Field:          relField.Name,
						RegisteredNode: registered, // The current node being processed
					}
				}
			}
		} else {
			// It's a node - this is a shorthand relationship (no relationship struct)
			targetNode, ok := targetInstance.(INode)
			if !ok {
				panic(fmt.Errorf("target %s does not implement INode or IRelationship", neoRel.NodeType))
			}

			targetReg := r.RegisterNode(targetNode)

			// Create relationship registration with empty name/typeName for shorthand
			relReg = &RegisteredRelationship{
				name:     "", // Empty for shorthand relationships
				typeName: "", // Empty for shorthand relationships
				Reltype:  neoRel.RelType,
			}
			if neoRel.Dir {
				relReg.StartNode = NodeTarget{RegisteredNode: targetReg}
				relReg.EndNode = NodeTarget{RegisteredNode: registered}
			} else {
				relReg.StartNode = NodeTarget{RegisteredNode: registered}
				relReg.EndNode = NodeTarget{RegisteredNode: targetReg}
			}
		}

		// Note: Don't store relationships in registeredTypes map to avoid conflicts with node names
		// Relationships are stored separately in r.Relationships slice

		registered.Relationships[fieldName] = &RelationshipTarget{
			Dir:  neoRel.Dir,
			Rel:  relReg,
			Many: neoRel.Many,
		}
	}

	if len(registered.Relationships) == 0 {
		registered.Relationships = nil
	}

	return registered
}

func (r *Registry) RegisterAbstractNode(typ any, typAbs IAbstract) *RegisteredAbstractNode {
	vv := codec.UnwindValue(reflect.ValueOf(typ))
	name := vv.Type().Name()
	// There's a chance that the abstract node is registered as a concrete node, in which case we re-register
	var node *RegisteredNode
	if n, ok := r.registeredTypes[name]; ok {
		if reg, ok := n.(*RegisteredNode); ok {
			node = reg
		} else {
			return n.(*RegisteredAbstractNode)
		}
	} else {
		node = r.RegisterNode(typ.(INode))
	}
	registered := &RegisteredAbstractNode{
		RegisteredNode: node,
	}
	impls := typAbs.Implementers()
	registered.Implementers = make([]*RegisteredNode, len(impls))
	for i, impl := range impls {
		registered.Implementers[i] = r.RegisterNode(impl)
	}
	r.AbstractNodes = append(r.AbstractNodes, registered)
	r.registeredTypes[name] = registered
	return registered
}

func (r *Registry) RegisterRelationship(v IRelationship) *RegisteredRelationship {
	// Extract metadata using codec registry (zero-reflection approach)
	relMeta, err := r.codecs.ExtractRelationshipMeta(v)
	if err != nil {
		panic(err)
	}

	name := relMeta.Name
	if existing, ok := r.registeredTypes[name]; ok {
		return existing.(*RegisteredRelationship)
	}

	registered := &RegisteredRelationship{
		name:     name,
		typeName: name, // Use name instead of reflection
		Reltype:  relMeta.Type,
	}
	r.Relationships = append(r.Relationships, registered)
	r.registeredTypes[name] = registered

	// Register start and end nodes (optional)
	if relMeta.StartNode != nil {
		startNodeInstance, err := r.codecs.CreateNodeInstance(relMeta.StartNode.NodeType)
		if err != nil {
			panic(fmt.Errorf("failed to create start node instance: %w", err))
		}
		node, ok := startNodeInstance.(INode)
		if !ok {
			panic(fmt.Errorf("start node %s does not implement INode", relMeta.StartNode.NodeType))
		}
		nodeReg := r.RegisterNode(node)
		registered.StartNode = NodeTarget{
			Field:          relMeta.StartNode.FieldName,
			RegisteredNode: nodeReg,
		}
	}

	if relMeta.EndNode != nil {
		endNodeInstance, err := r.codecs.CreateNodeInstance(relMeta.EndNode.NodeType)
		if err != nil {
			panic(fmt.Errorf("failed to create end node instance: %w", err))
		}
		node, ok := endNodeInstance.(INode)
		if !ok {
			panic(fmt.Errorf("end node %s does not implement INode", relMeta.EndNode.NodeType))
		}
		nodeReg := r.RegisterNode(node)
		registered.EndNode = NodeTarget{
			Field:          relMeta.EndNode.FieldName,
			RegisteredNode: nodeReg,
		}
	}

	return registered
}

func (r *Registry) Get(typ reflect.Type) (entity RegisteredEntity) {
	if typ == nil {
		return nil
	}
	name := codec.UnwindType(typ).Name()
	if v, ok := r.registeredTypes[name]; ok {
		return v
	}
	if typ.Implements(rAbstract) {
		return r.registeredTypes["Base"+name]
	}
	return nil
}

func (r *Registry) GetByName(name string) (entity RegisteredEntity) {
	return r.registeredTypes[name]
}

func (r *Registry) GetConcreteImplementation(nodeLabels []string) (*RegisteredNode, error) {
	var (
		abstractNode       *RegisteredAbstractNode
		closestImpl        *RegisteredNode
		inheritanceCounter int
		isNodeLabel        = make(map[string]struct{}, len(nodeLabels))
	)
	for _, label := range nodeLabels {
		isNodeLabel[label] = struct{}{}
	}
	// We find the abstract node (or exact implementation if registered) that has
	// a inheritance chain closest to the database node we're extracting from.
Bases:
	for _, base := range r.AbstractNodes {
		labels := base.Labels
		if len(labels) == 0 {
			continue
		}
		currentInheritanceCounter := 0
		for _, label := range labels {
			if _, ok := isNodeLabel[label]; !ok {
				continue Bases
			}
			currentInheritanceCounter++
		}
		if currentInheritanceCounter > inheritanceCounter {
			abstractNode = base
			inheritanceCounter = currentInheritanceCounter
		}
	}
	if abstractNode == nil {
		return nil, fmt.Errorf(
			"no abstract node found for labels: %s\nDid you forget to register the base node using neogo.WithTypes(...)?",
			strings.Join(nodeLabels, ", "),
		)
	}
	if inheritanceCounter == len(nodeLabels) {
		return abstractNode.RegisteredNode, nil
	}
Impls:
	for _, nextImpl := range abstractNode.Implementers {
		for _, label := range nextImpl.Labels {
			if _, ok := isNodeLabel[label]; !ok {
				continue Impls
			}
		}
		closestImpl = nextImpl
		break
	}
	if closestImpl == nil {
		return nil, fmt.Errorf(
			"no concrete implementation found for labels: %s\nDid you forget to register the base node using neogo.WithTypes(...)?",
			strings.Join(nodeLabels, ", "),
		)
	}
	return closestImpl, nil
}
