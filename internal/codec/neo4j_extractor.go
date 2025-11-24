package codec

import (
	"fmt"
	"reflect"
	"strings"
)

// Neo4jNodeMetadata contains metadata extracted from a node struct
type Neo4jNodeMetadata struct {
	Name          string
	Labels        []string
	FieldsToProps map[string]string
	Relationships map[string]*Neo4jRelationshipTarget
}

// Neo4jRelationshipTarget represents a relationship field in a node
type Neo4jRelationshipTarget struct {
	FieldName string
	Many      bool // true for slice relationships
	Dir       bool // true = ->, false = <-
	RelType   string
	NodeType  reflect.Type // Target node type
}

// ExtractNeo4jNodeMeta extracts Neo4j node metadata from a struct
// This uses reflection ONCE during registration
func (r *CodecRegistry) ExtractNeo4jNodeMeta(v any) (*Neo4jNodeMetadata, error) {
	typ := reflect.TypeOf(v)
	val := reflect.ValueOf(v)

	// Unwrap pointer
	if typ.Kind() == reflect.Ptr {
		typ = typ.Elem()
		val = val.Elem()
	}
	if typ.Kind() != reflect.Struct {
		return nil, fmt.Errorf("expected struct, got %s", typ.Kind())
	}

	meta := &Neo4jNodeMetadata{
		Name:          typ.Name(),
		Labels:        []string{},
		FieldsToProps: make(map[string]string),
		Relationships: make(map[string]*Neo4jRelationshipTarget),
	}

	// Track labels to append at the end (from anonymous fields)
	postpendLabels := []string{}

	// Walk through all fields
	err := r.walkStructFields(typ, val, meta, &postpendLabels)
	if err != nil {
		return nil, err
	}

	// Append labels from anonymous fields
	meta.Labels = append(meta.Labels, postpendLabels...)

	if len(meta.Labels) == 0 {
		return nil, fmt.Errorf("node %s has no labels", meta.Name)
	}

	return meta, nil
}

// walkStructFields walks through struct fields and extracts Neo4j metadata
func (r *CodecRegistry) walkStructFields(typ reflect.Type, val reflect.Value, meta *Neo4jNodeMetadata, postpendLabels *[]string) error {
	for i := 0; i < typ.NumField(); i++ {
		field := typ.Field(i)
		fieldVal := val.Field(i)

		// Extract field name mapping from db or json tags (only for non-anonymous fields)
		if !field.Anonymous {
			if dbName, ok := r.extractFieldName(field); ok {
				meta.FieldsToProps[field.Name] = dbName
			}
		}

		// Handle anonymous fields (embedded structs)
		var shouldRecurse bool
		if field.Anonymous {
			shouldRecurse = r.handleAnonymousField(field, meta, postpendLabels)
		}

		// Parse DB or Neo4j tags (support both formats)
		var tag string
		var hasTag bool
		if tag, hasTag = field.Tag.Lookup("db"); !hasTag || tag == "" {
			tag, hasTag = field.Tag.Lookup("neo4j")
		}
		if !hasTag || tag == "" {
			// If no neo4j tag but should recurse, continue recursion
			if shouldRecurse {
				// Recurse into embedded struct
				if fieldVal.Kind() == reflect.Ptr {
					if !fieldVal.IsNil() {
						fieldVal = fieldVal.Elem()
					} else {
						continue
					}
				}
				err := r.walkStructFields(fieldVal.Type(), fieldVal, meta, postpendLabels)
				if err != nil {
					return err
				}
			}
			continue
		}

		// Parse tag
		err := r.parseNeo4jTag(field, tag, meta, postpendLabels, typ.Name())
		if err != nil {
			return err
		}

		// Handle recursion if needed
		if shouldRecurse {
			if fieldVal.Kind() == reflect.Ptr {
				if !fieldVal.IsNil() {
					fieldVal = fieldVal.Elem()
				} else {
					continue
				}
			}
			err := r.walkStructFields(fieldVal.Type(), fieldVal, meta, postpendLabels)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

// extractFieldName extracts field name from db, json, or neo4j tags
func (r *CodecRegistry) extractFieldName(field reflect.StructField) (string, bool) {
	// Try db tag first (primary for Neo4j properties)
	if dbTag := field.Tag.Get("db"); dbTag != "" && dbTag != "-" {
		parts := strings.Split(dbTag, ",")
		if len(parts) > 0 {
			fieldName := strings.TrimSpace(parts[0])
			// Skip relationship direction markers and other special values
			if fieldName != "" && fieldName != "->" && fieldName != "<-" && fieldName != "startNode" && fieldName != "endNode" {
				return fieldName, true
			}
		}
	}

	// Fallback to json tag
	if jsonTag := field.Tag.Get("json"); jsonTag != "" && jsonTag != "-" {
		parts := strings.Split(jsonTag, ",")
		if len(parts) > 0 {
			fieldName := strings.TrimSpace(parts[0])
			if fieldName != "" {
				return fieldName, true
			}
		}
	}

	return "", false
}

// extractJSONFieldName extracts JSON field name from struct field (deprecated - use extractFieldName)
func (r *CodecRegistry) extractJSONFieldName(field reflect.StructField) (string, bool) {
	return r.extractFieldName(field)
}

// handleAnonymousField handles embedded/anonymous struct fields
func (r *CodecRegistry) handleAnonymousField(field reflect.StructField, meta *Neo4jNodeMetadata, postpendLabels *[]string) bool {
	fieldType := field.Type

	// Check if it's the base Node type (special case)
	if r.isBaseNodeType(fieldType) {
		return true // Should recurse but don't register
	}

	// Check if it implements INode interface
	if r.implementsINode(fieldType) {
		// Register the nested node and merge its metadata
		nestedInstance := reflect.New(fieldType).Interface()
		nestedMeta, err := r.ExtractNeo4jNodeMeta(nestedInstance)
		if err == nil {
			// Merge labels, fields, and relationships
			meta.Labels = append(meta.Labels, nestedMeta.Labels...)
			for k, v := range nestedMeta.FieldsToProps {
				meta.FieldsToProps[k] = v
			}
			for k, v := range nestedMeta.Relationships {
				meta.Relationships[k] = v
			}
		}
		return false // Don't recurse further
	}

	return true // Should recurse for other anonymous fields
}

// parseNeo4jTag parses neo4j struct tag and updates metadata
func (r *CodecRegistry) parseNeo4jTag(field reflect.StructField, neo4jTag string, meta *Neo4jNodeMetadata, postpendLabels *[]string, typeName string) error {
	parts := strings.Split(neo4jTag, ",")
	if len(parts) == 0 {
		return fmt.Errorf("invalid tag format for field %s.%s: %s", typeName, field.Name, neo4jTag)
	}

	ident := parts[0]
	switch ident {
	case "<-", "->":
		// Direct relationship definition
		return r.registerRelationshipField(field, ident == "->", "", meta, typeName)

	case "":
		return fmt.Errorf("field has empty neo4j label / direction: %s.%s", typeName, field.Name)

	default:
		// Label or shorthand relationship
		if field.Anonymous {
			*postpendLabels = append(*postpendLabels, ident)
		} else {
			// Check if it's a shorthand relationship (starts with < or ends with >)
			if ident[0] == '<' {
				return r.registerRelationshipField(field, false, ident[1:], meta, typeName)
			} else if ident[len(ident)-1] == '>' {
				return r.registerRelationshipField(field, true, ident[:len(ident)-1], meta, typeName)
			}
			// If it's not a relationship and not anonymous, it's ignored
			// Labels only come from anonymous embedded structs
		}
	}

	return nil
}

// registerRelationshipField registers a relationship field
func (r *CodecRegistry) registerRelationshipField(field reflect.StructField, dir bool, shorthand string, meta *Neo4jNodeMetadata, typeName string) error {
	relType := field.Type
	isMany := false

	// Check if it's a slice type (many relationship)
	if relType.Kind() == reflect.Slice {
		// Direct slice like []*simpleRelationship
		relType = relType.Elem() // Get *simpleRelationship from []*simpleRelationship
		isMany = true
	} else if relType.Kind() == reflect.Struct {
		// Many[T] wrapper type
		relField, ok := relType.FieldByName("S")
		if ok {
			relType = relField.Type.Elem() // Get T from []T
			isMany = true
		}
	}

	if relType.Kind() != reflect.Ptr {
		return fmt.Errorf("invalid relationship for field %s.%s. Got %s", typeName, field.Name, relType)
	}

	// Determine relationship type and target node type
	var relTypeName string
	if shorthand != "" {
		relTypeName = shorthand
	} else {
		// Extract relationship type from db/neo4j tag on the relationship struct
		relStructType := relType.Elem()
		if r.implementsIRelationship(relType) {
			// Look for db or neo4j tag on the relationship struct itself
			if relStructType.Kind() == reflect.Struct {
				for i := 0; i < relStructType.NumField(); i++ {
					field := relStructType.Field(i)
					if field.Anonymous && field.Type.Name() == "Relationship" {
						if tagVal := field.Tag.Get("db"); tagVal != "" {
							relTypeName = tagVal
							break
						} else if tagVal := field.Tag.Get("neo4j"); tagVal != "" {
							relTypeName = tagVal
							break
						}
					}
				}
			}
		}
		// Fallback to default relationship type for shorthand relationships
		if relTypeName == "" {
			// For shorthand relationships (where target is a node, not a relationship struct),
			// use "SHORTHAND" as the default relationship type
			if r.implementsINode(relType) {
				relTypeName = "SHORTHAND"
			} else {
				// For regular relationship structs, use the struct name
				relTypeName = relStructType.Name()
			}
		}
	}

	meta.Relationships[field.Name] = &Neo4jRelationshipTarget{
		FieldName: field.Name,
		Many:      isMany,
		Dir:       dir,
		RelType:   relTypeName,
		NodeType:  relType,
	}

	return nil
}

// Helper methods to check type implementations
func (r *CodecRegistry) isBaseNodeType(typ reflect.Type) bool {
	// Check if it's the base Node type - this should be configurable
	return typ.Name() == "Node"
}

func (r *CodecRegistry) implementsINode(typ reflect.Type) bool {
	// Create INode interface type
	iNodeType := reflect.TypeOf((*interface{ IsNode() })(nil)).Elem()
	return typ.Implements(iNodeType)
}

func (r *CodecRegistry) implementsIRelationship(typ reflect.Type) bool {
	// Create IRelationship interface type
	iRelType := reflect.TypeOf((*interface{ IsRelationship() })(nil)).Elem()
	return typ.Implements(iRelType)
}
