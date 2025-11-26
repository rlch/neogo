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
	Schema    *SchemaMeta  // Aggregated schema (indexes + constraints)
	RType     reflect.Type // Stored reflect.Type for delegation
}

// ReflectType returns the reflect.Type for this relationship
func (m *RelationshipStructMeta) ReflectType() reflect.Type {
	return m.RType
}

// NodeFieldMeta contains metadata about a node field in a relationship
type NodeFieldMeta struct {
	FieldName string
	NodeType  reflect.Type
}

// ExtractRelationshipMeta extracts relationship metadata from a struct type.
// REGISTRATION PHASE: Uses heavy reflection, called only during type registration.
// Deprecated: Use extractRelationshipMetaFromType for new code.
func (r *CodecRegistry) ExtractRelationshipMeta(v any) (*RelationshipStructMeta, error) {
	typ := reflect.TypeOf(v)
	if typ.Kind() == reflect.Ptr {
		typ = typ.Elem()
	}
	return r.extractRelationshipMetaFromType(typ)
}

// extractRelationshipMetaFromType extracts relationship metadata from a reflect.Type.
// This is the internal method that does all the work, including schema aggregation.
// REGISTRATION PHASE: Uses heavy reflection, called only during type registration.
func (r *CodecRegistry) extractRelationshipMetaFromType(typ reflect.Type) (*RelationshipStructMeta, error) {
	if typ.Kind() == reflect.Ptr {
		typ = typ.Elem()
	}
	if typ.Kind() != reflect.Struct {
		return nil, fmt.Errorf("expected struct, got %s", typ.Kind())
	}

	meta := &RelationshipStructMeta{
		Name:  typ.Name(),
		RType: typ,
	}

	// Collect field schema info for aggregation
	var fieldSchemas []FieldSchemaInfo

	// Walk through struct fields to extract metadata
	for i := 0; i < typ.NumField(); i++ {
		field := typ.Field(i)

		// Parse field info for schema
		fieldInfo := parseFieldInfo(field)

		// Parse neo4j tag
		neo4jTag := field.Tag.Get("neo4j")
		if neo4jTag == "" {
			continue
		}

		parts := strings.Split(neo4jTag, ",")
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
			// Only embedded Relationship struct's tag defines the relationship type
			// Other fields with neo4j tags are property fields
			if field.Anonymous && field.Type.Name() == "Relationship" {
				if meta.Type != "" {
					return nil, fmt.Errorf("relationship %s has multiple type definitions", meta.Name)
				}
				meta.Type = tagValue
			} else if !field.Anonymous && !fieldInfo.IsSkip {
				// Collect schema info for property fields
				if fieldInfo.Index != nil || fieldInfo.Constraint != nil {
					*&fieldSchemas = append(fieldSchemas, FieldSchemaInfo{
						DBName:     fieldInfo.DBName,
						Index:      fieldInfo.Index,
						Constraint: fieldInfo.Constraint,
					})
				}
			}
		}
	}

	// Validate required fields - only type is required, startNode/endNode are optional
	if meta.Type == "" {
		return nil, fmt.Errorf("relationship %s missing type definition in neo4j tag", meta.Name)
	}

	// Aggregate schema using relationship type as label
	meta.Schema = &SchemaMeta{
		TypeName:    meta.Name,
		RelType:     meta.Type,
		IsNode:      false,
		Indexes:     AggregateIndexes(meta.Type, false, fieldSchemas),
		Constraints: AggregateConstraints(meta.Type, false, fieldSchemas),
	}

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
