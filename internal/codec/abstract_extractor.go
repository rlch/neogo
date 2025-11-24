package codec

import (
	"fmt"
	"reflect"
)

// AbstractNodeMetadata contains metadata extracted from an abstract node
type AbstractNodeMetadata struct {
	Name         string
	BaseNodeMeta *Neo4jNodeMetadata
	Implementers []*Neo4jNodeMetadata
}

// ExtractAbstractNodeMeta extracts abstract node metadata from a type
// This uses reflection ONCE during registration
func (r *CodecRegistry) ExtractAbstractNodeMeta(v any, abs IAbstract) (*AbstractNodeMetadata, error) {
	// First extract base node metadata
	baseNodeMeta, err := r.ExtractNeo4jNodeMeta(v)
	if err != nil {
		return nil, fmt.Errorf("failed to extract base node metadata: %w", err)
	}

	meta := &AbstractNodeMetadata{
		Name:         baseNodeMeta.Name,
		BaseNodeMeta: baseNodeMeta,
		Implementers: []*Neo4jNodeMetadata{},
	}

	// Extract implementers
	implementers := abs.Implementers()
	meta.Implementers = make([]*Neo4jNodeMetadata, len(implementers))

	for i, impl := range implementers {
		implMeta, err := r.ExtractNeo4jNodeMeta(impl)
		if err != nil {
			return nil, fmt.Errorf("failed to extract implementer %d metadata: %w", i, err)
		}
		meta.Implementers[i] = implMeta
	}

	return meta, nil
}

// RegistryMetadata contains all metadata needed by the registry
type RegistryMetadata struct {
	Name          string
	Type          reflect.Type
	FieldsToProps map[string]string
	Labels        []string
	Relationships map[string]*Neo4jRelationshipTarget

	// For abstract nodes
	IsAbstract   bool
	Implementers []*RegistryMetadata

	// For relationships
	IsRelationship bool
	RelType        string
	StartNode      *NodeFieldMeta
	EndNode        *NodeFieldMeta
}

// ExtractRegistryMetadata extracts all metadata needed by the registry
func (r *CodecRegistry) ExtractRegistryMetadata(v any) (*RegistryMetadata, error) {
	vv := UnwindValue(reflect.ValueOf(v))
	vvt := vv.Type()

	meta := &RegistryMetadata{
		Name: vvt.Name(),
		Type: vvt,
	}

	// Check what type of entity this is
	if abs, ok := v.(IAbstract); ok {
		// Abstract node
		meta.IsAbstract = true

		abstractMeta, err := r.ExtractAbstractNodeMeta(v, abs)
		if err != nil {
			return nil, err
		}

		meta.FieldsToProps = abstractMeta.BaseNodeMeta.FieldsToProps
		meta.Labels = abstractMeta.BaseNodeMeta.Labels
		meta.Relationships = abstractMeta.BaseNodeMeta.Relationships

		// Convert implementers
		meta.Implementers = make([]*RegistryMetadata, len(abstractMeta.Implementers))
		for i, impl := range abstractMeta.Implementers {
			meta.Implementers[i] = &RegistryMetadata{
				Name:          impl.Name,
				Type:          reflect.TypeOf(impl).Elem(), // Assuming implementers are pointers
				FieldsToProps: impl.FieldsToProps,
				Labels:        impl.Labels,
				Relationships: impl.Relationships,
			}
		}

	} else if _, ok := v.(INode); ok {
		// Regular node
		nodeMeta, err := r.ExtractNeo4jNodeMeta(v)
		if err != nil {
			return nil, err
		}

		meta.FieldsToProps = nodeMeta.FieldsToProps
		meta.Labels = nodeMeta.Labels
		meta.Relationships = nodeMeta.Relationships

	} else if _, ok := v.(IRelationship); ok {
		// Relationship
		meta.IsRelationship = true

		relMeta, err := r.ExtractRelationshipMeta(v)
		if err != nil {
			return nil, err
		}

		meta.RelType = relMeta.Type
		meta.StartNode = relMeta.StartNode
		meta.EndNode = relMeta.EndNode

		// For relationships, we don't have fields to props mapping typically
		meta.FieldsToProps = make(map[string]string)
	}

	return meta, nil
}
