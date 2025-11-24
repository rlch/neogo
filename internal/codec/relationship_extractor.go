package codec

import (
	"fmt"
	"reflect"
	"strings"
)

// RelationshipStructMeta contains metadata extracted from a relationship struct
type RelationshipStructMeta struct {
	Name      string
	Type      string // Relationship type from db tag
	StartNode *NodeFieldMeta
	EndNode   *NodeFieldMeta
}

// NodeFieldMeta contains metadata about a node field in a relationship
type NodeFieldMeta struct {
	FieldName string
	NodeType  reflect.Type
}

// ExtractRelationshipMeta extracts relationship metadata from a struct type
// This uses reflection ONCE during registration, then caches the results
func (r *CodecRegistry) ExtractRelationshipMeta(v any) (*RelationshipStructMeta, error) {
	typ := reflect.TypeOf(v)
	if typ.Kind() == reflect.Ptr {
		typ = typ.Elem()
	}
	if typ.Kind() != reflect.Struct {
		return nil, fmt.Errorf("expected struct, got %s", typ.Kind())
	}

	meta := &RelationshipStructMeta{
		Name: typ.Name(),
	}

	// Walk through struct fields to extract metadata
	for i := 0; i < typ.NumField(); i++ {
		field := typ.Field(i)

		// Parse neo4j tag (will be changed to db later)
		dbTag := field.Tag.Get("neo4j")
		if dbTag == "" {
			continue
		}

		parts := strings.Split(dbTag, ",")
		if len(parts) == 0 {
			continue
		}

		tagValue := strings.TrimSpace(parts[0])
		switch tagValue {
		case "startNode":
			if meta.StartNode != nil {
				return nil, fmt.Errorf("relationship %s has multiple startNode fields", meta.Name)
			}
			nodeType := field.Type
			if nodeType.Kind() != reflect.Ptr {
				return nil, fmt.Errorf("expected pointer to struct for startNode field %s.%s", meta.Name, field.Name)
			}
			meta.StartNode = &NodeFieldMeta{
				FieldName: field.Name,
				NodeType:  nodeType,
			}

		case "endNode":
			if meta.EndNode != nil {
				return nil, fmt.Errorf("relationship %s has multiple endNode fields", meta.Name)
			}
			nodeType := field.Type
			if nodeType.Kind() != reflect.Ptr {
				return nil, fmt.Errorf("expected pointer to struct for endNode field %s.%s", meta.Name, field.Name)
			}
			meta.EndNode = &NodeFieldMeta{
				FieldName: field.Name,
				NodeType:  nodeType,
			}

		default:
			// This should be the relationship type
			if meta.Type != "" {
				return nil, fmt.Errorf("relationship %s has multiple type definitions", meta.Name)
			}
			meta.Type = tagValue
		}
	}

	// Validate required fields - only type is required, startNode/endNode are optional
	if meta.Type == "" {
		return nil, fmt.Errorf("relationship %s missing type definition in neo4j tag", meta.Name)
	}
	// startNode and endNode are optional - some relationships don't define explicit endpoints

	return meta, nil
}

// CreateNodeInstance creates a new instance of a node type for registration
// This is used to trigger node registration from relationship registration
func (r *CodecRegistry) CreateNodeInstance(nodeType reflect.Type) (any, error) {
	if nodeType.Kind() != reflect.Ptr {
		return nil, fmt.Errorf("expected pointer type, got %s", nodeType.Kind())
	}

	elemType := nodeType.Elem()
	if elemType.Kind() != reflect.Struct {
		return nil, fmt.Errorf("expected pointer to struct, got pointer to %s", elemType.Kind())
	}

	// Create new instance
	instance := reflect.New(elemType).Interface()
	return instance, nil
}
