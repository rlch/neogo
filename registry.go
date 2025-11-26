package neogo

import (
	"reflect"

	"github.com/rlch/neogo/internal"
	"github.com/rlch/neogo/internal/codec"
)

// Registry manages type registration and metadata for neogo models.
// It provides schema introspection capabilities for external tools like scaf.
type Registry struct {
	internal *internal.Registry
}

// NewRegistry creates a new empty registry.
func NewRegistry() *Registry {
	return &Registry{
		internal: internal.NewRegistry(),
	}
}

// RegisterTypes registers multiple types with the registry.
// Types should be zero-value pointers (e.g., &Person{}, &Movie{}).
func (r *Registry) RegisterTypes(types ...any) {
	r.internal.RegisterTypes(types...)
}

// Nodes returns all registered node types.
func (r *Registry) Nodes() []*RegisteredNode {
	nodes := r.internal.Nodes
	result := make([]*RegisteredNode, len(nodes))
	for i, n := range nodes {
		result[i] = &RegisteredNode{internal: n, codecs: r.internal.Codecs()}
	}
	return result
}

// Relationships returns all registered relationship types.
func (r *Registry) Relationships() []*RegisteredRelationship {
	rels := r.internal.Relationships
	result := make([]*RegisteredRelationship, len(rels))
	for i, rel := range rels {
		result[i] = &RegisteredRelationship{internal: rel}
	}
	return result
}

// GetRelMeta returns relationship struct metadata by name.
// Returns nil if not found.
func (r *Registry) GetRelMeta(name string) *RelationshipMeta {
	meta := r.internal.Codecs().GetRelMeta(name)
	if meta == nil {
		return nil
	}
	return &RelationshipMeta{internal: meta}
}

// RegisteredNode represents a registered node type.
type RegisteredNode struct {
	internal *internal.RegisteredNode
	codecs   *codec.CodecRegistry
}

// Name returns the node type name (e.g., "Person").
func (n *RegisteredNode) Name() string {
	return n.internal.Name()
}

// Type returns the reflect.Type for this node.
func (n *RegisteredNode) Type() reflect.Type {
	return n.internal.Type()
}

// FieldsToProps returns a map from Go field names to database property names.
func (n *RegisteredNode) FieldsToProps() map[string]string {
	return n.internal.FieldsToProps()
}

// Relationships returns the relationship targets for this node.
// The map key is the Go field name.
func (n *RegisteredNode) Relationships() map[string]*RelationshipTarget {
	rels := n.internal.Relationships
	if rels == nil {
		return nil
	}
	result := make(map[string]*RelationshipTarget, len(rels))
	for fieldName, target := range rels {
		result[fieldName] = &RelationshipTarget{internal: target}
	}
	return result
}

// RegisteredRelationship represents a registered relationship type.
type RegisteredRelationship struct {
	internal *internal.RegisteredRelationship
}

// Name returns the relationship type name (e.g., "ActedIn").
// Returns empty string for shorthand relationships.
func (r *RegisteredRelationship) Name() string {
	return r.internal.Name()
}

// Type returns the reflect.Type for this relationship.
func (r *RegisteredRelationship) Type() reflect.Type {
	return r.internal.Type()
}

// FieldsToProps returns a map from Go field names to database property names.
func (r *RegisteredRelationship) FieldsToProps() map[string]string {
	return r.internal.FieldsToProps()
}

// RelationshipTarget represents a relationship from a node to another entity.
type RelationshipTarget struct {
	internal *internal.RelationshipTarget
}

// Many returns true if this is a one-to-many relationship (Many[T]).
func (t *RelationshipTarget) Many() bool {
	return t.internal.Many
}

// Dir returns true for outgoing relationships (->), false for incoming (<-).
func (t *RelationshipTarget) Dir() bool {
	return t.internal.Dir
}

// RelType returns the database relationship type (e.g., "ACTED_IN").
func (t *RelationshipTarget) RelType() string {
	return t.internal.Rel.Reltype
}

// RelName returns the relationship struct name, or empty string for shorthand relationships.
func (t *RelationshipTarget) RelName() string {
	return t.internal.Rel.Name()
}

// StartNode returns the start node target, or nil if not available.
func (t *RelationshipTarget) StartNode() *NodeTarget {
	if t.internal.Rel.StartNode.RegisteredNode == nil {
		return nil
	}
	return &NodeTarget{internal: &t.internal.Rel.StartNode}
}

// EndNode returns the end node target, or nil if not available.
func (t *RelationshipTarget) EndNode() *NodeTarget {
	if t.internal.Rel.EndNode.RegisteredNode == nil {
		return nil
	}
	return &NodeTarget{internal: &t.internal.Rel.EndNode}
}

// NodeTarget represents a node at one end of a relationship.
type NodeTarget struct {
	internal *internal.NodeTarget
}

// Name returns the node type name.
func (t *NodeTarget) Name() string {
	if t.internal.RegisteredNode == nil {
		return ""
	}
	return t.internal.RegisteredNode.Name()
}

// RelationshipMeta contains metadata about a relationship struct.
type RelationshipMeta struct {
	internal *codec.RelationshipStructMeta
}

// StartNode returns metadata about the start node field, or nil if not defined.
func (m *RelationshipMeta) StartNode() *RelationshipNodeMeta {
	if m.internal.StartNode == nil {
		return nil
	}
	return &RelationshipNodeMeta{internal: m.internal.StartNode}
}

// EndNode returns metadata about the end node field, or nil if not defined.
func (m *RelationshipMeta) EndNode() *RelationshipNodeMeta {
	if m.internal.EndNode == nil {
		return nil
	}
	return &RelationshipNodeMeta{internal: m.internal.EndNode}
}

// RelationshipNodeMeta contains metadata about a node field in a relationship struct.
type RelationshipNodeMeta struct {
	internal *codec.NodeFieldMeta
}

// NodeType returns the reflect.Type of the node (pointer type, e.g., *Person).
func (m *RelationshipNodeMeta) NodeType() reflect.Type {
	return m.internal.NodeType
}
