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
	Schema        *SchemaMeta  // Aggregated schema (indexes + constraints)
	RType         reflect.Type // Stored reflect.Type for delegation
}

// Type returns the reflect.Type for this node
func (m *Neo4jNodeMetadata) Type() reflect.Type {
	return m.RType
}

// Neo4jRelationshipTarget represents a relationship field in a node
type Neo4jRelationshipTarget struct {
	FieldName string
	Many      bool         // true for Many[R], false for One[R]
	Dir       bool         // true = ->, false = <-
	RelType   string       // Relationship type (e.g., "ACTED_IN")
	NodeType  reflect.Type // Target type (node or relationship struct) - use registry to get labels
	Required  bool         // true if relationship is required (existence constraint)
}

// ExtractNeo4jNodeMeta extracts Neo4j node metadata from a struct.
// REGISTRATION PHASE: Uses heavy reflection, called only during type registration.
// Deprecated: Use extractNeo4jNodeMetaFromType for new code.
func (r *CodecRegistry) ExtractNeo4jNodeMeta(v any) (*Neo4jNodeMetadata, error) {
	typ := reflect.TypeOf(v)
	if typ.Kind() == reflect.Ptr {
		typ = typ.Elem()
	}
	return r.extractNeo4jNodeMetaFromType(typ)
}

// extractNeo4jNodeMetaFromType extracts Neo4j node metadata from a reflect.Type.
// This is the internal method that does all the work, including schema aggregation.
// REGISTRATION PHASE: Uses heavy reflection, called only during type registration.
func (r *CodecRegistry) extractNeo4jNodeMetaFromType(typ reflect.Type) (*Neo4jNodeMetadata, error) {
	if typ.Kind() == reflect.Ptr {
		typ = typ.Elem()
	}
	if typ.Kind() != reflect.Struct {
		return nil, fmt.Errorf("expected struct, got %s", typ.Kind())
	}

	// Create a zero value for walking (needed for embedded struct recursion)
	val := reflect.New(typ).Elem()

	meta := &Neo4jNodeMetadata{
		Name:          typ.Name(),
		Labels:        []string{},
		FieldsToProps: make(map[string]string),
		Relationships: make(map[string]*Neo4jRelationshipTarget),
		RType:         typ,
	}

	// Track labels to append at the end (from anonymous fields)
	postpendLabels := []string{}

	// Collect field schema info for aggregation
	var fieldSchemas []FieldSchemaInfo

	// Walk through all fields
	err := r.walkStructFieldsWithSchema(typ, val, meta, &postpendLabels, &fieldSchemas)
	if err != nil {
		return nil, err
	}

	// Append labels from anonymous fields
	meta.Labels = append(meta.Labels, postpendLabels...)

	if len(meta.Labels) == 0 {
		return nil, fmt.Errorf("node %s has no labels", meta.Name)
	}

	// Use most specific label (first label) for schema naming
	primaryLabel := meta.Labels[0]

	// Auto-add unique constraint for ID field if present
	if idProp, hasID := meta.FieldsToProps["ID"]; hasID {
		fieldSchemas = append(fieldSchemas, FieldSchemaInfo{
			DBName: idProp,
			Constraint: &ConstraintSpec{
				Type:     ConstraintTypeUnique,
				Priority: 10,
			},
		})
	}

	// Aggregate schema from field specs
	meta.Schema = &SchemaMeta{
		TypeName:    meta.Name,
		Labels:      meta.Labels,
		IsNode:      true,
		Indexes:     AggregateIndexes(primaryLabel, true, fieldSchemas),
		Constraints: AggregateConstraints(primaryLabel, true, fieldSchemas),
	}

	return meta, nil
}

// walkStructFields walks through struct fields and extracts Neo4j metadata
// Deprecated: Use walkStructFieldsWithSchema instead.
func (r *CodecRegistry) walkStructFields(typ reflect.Type, val reflect.Value, meta *Neo4jNodeMetadata, postpendLabels *[]string) error {
	var fieldSchemas []FieldSchemaInfo
	return r.walkStructFieldsWithSchema(typ, val, meta, postpendLabels, &fieldSchemas)
}

// walkStructFieldsWithSchema walks through struct fields and extracts Neo4j metadata + schema info
func (r *CodecRegistry) walkStructFieldsWithSchema(typ reflect.Type, val reflect.Value, meta *Neo4jNodeMetadata, postpendLabels *[]string, fieldSchemas *[]FieldSchemaInfo) error {
	for i := 0; i < typ.NumField(); i++ {
		field := typ.Field(i)
		fieldVal := val.Field(i)

		// Parse field info to get both DB name and schema
		fieldInfo := parseFieldInfo(field)

		// Extract field name mapping from neo4j tags (only for non-anonymous fields)
		if !field.Anonymous && !fieldInfo.IsSkip {
			if propName, ok := r.extractFieldName(field); ok {
				meta.FieldsToProps[field.Name] = propName

				// Collect schema info for this field
				if fieldInfo.Index != nil || fieldInfo.Constraint != nil {
					*fieldSchemas = append(*fieldSchemas, FieldSchemaInfo{
						DBName:     propName,
						Index:      fieldInfo.Index,
						Constraint: fieldInfo.Constraint,
					})
				}
			}
		}

		// Handle anonymous fields (embedded structs)
		var shouldRecurse bool
		if field.Anonymous {
			shouldRecurse = r.handleAnonymousFieldWithSchema(field, meta, postpendLabels, fieldSchemas)
		}

		// Parse neo4j tag
		tag, hasTag := field.Tag.Lookup("neo4j")
		if !hasTag || tag == "" {
			// If no neo4j tag but should recurse, continue recursion
			if shouldRecurse {
				// Recurse into embedded struct
				embeddedVal := fieldVal
				if embeddedVal.Kind() == reflect.Ptr {
					if embeddedVal.IsNil() {
						// Create a new instance for nil pointers
						embeddedVal = reflect.New(field.Type.Elem()).Elem()
					} else {
						embeddedVal = embeddedVal.Elem()
					}
				}
				err := r.walkStructFieldsWithSchema(embeddedVal.Type(), embeddedVal, meta, postpendLabels, fieldSchemas)
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
			embeddedVal := fieldVal
			if embeddedVal.Kind() == reflect.Ptr {
				if embeddedVal.IsNil() {
					// Create a new instance for nil pointers
					embeddedVal = reflect.New(field.Type.Elem()).Elem()
				} else {
					embeddedVal = embeddedVal.Elem()
				}
			}
			err := r.walkStructFieldsWithSchema(embeddedVal.Type(), embeddedVal, meta, postpendLabels, fieldSchemas)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

// extractFieldName extracts field name from neo4j tag for property fields.
// Returns false for relationship fields (One[R]/Many[R]) and special markers.
func (r *CodecRegistry) extractFieldName(field reflect.StructField) (string, bool) {
	neo4jTag := field.Tag.Get("neo4j")
	if neo4jTag == "" || neo4jTag == "-" {
		return "", false
	}

	// Skip One[R] and Many[R] relationship fields - they're not properties
	fieldTypeName := field.Type.Name()
	if strings.HasPrefix(fieldTypeName, "One[") || strings.HasPrefix(fieldTypeName, "Many[") {
		return "", false
	}

	parts := strings.Split(neo4jTag, ",")
	if len(parts) > 0 {
		fieldName := strings.TrimSpace(parts[0])
		// Skip relationship direction markers and other special values
		if fieldName == "" || fieldName == "->" || fieldName == "<-" || fieldName == "startNode" || fieldName == "endNode" {
			return "", false
		}
		// Skip shorthand relationship syntax (e.g., "FRIEND>", "<FOLLOWS")
		if strings.HasSuffix(fieldName, ">") || strings.HasPrefix(fieldName, "<") {
			return "", false
		}
		return fieldName, true
	}

	return "", false
}

// handleAnonymousField handles embedded/anonymous struct fields
// Deprecated: Use handleAnonymousFieldWithSchema instead.
func (r *CodecRegistry) handleAnonymousField(field reflect.StructField, meta *Neo4jNodeMetadata, postpendLabels *[]string) bool {
	var fieldSchemas []FieldSchemaInfo
	return r.handleAnonymousFieldWithSchema(field, meta, postpendLabels, &fieldSchemas)
}

// handleAnonymousFieldWithSchema handles embedded/anonymous struct fields and collects schema info
func (r *CodecRegistry) handleAnonymousFieldWithSchema(field reflect.StructField, meta *Neo4jNodeMetadata, postpendLabels *[]string, fieldSchemas *[]FieldSchemaInfo) bool {
	fieldType := field.Type

	// Check if it's the base Node type (special case - always recurse)
	if r.isBaseNodeType(fieldType) {
		return true // Should recurse but don't extract as node
	}

	// Check if it implements INode interface
	if r.implementsINode(fieldType) {
		// Try to extract the nested node metadata
		nestedMeta, err := r.extractNeo4jNodeMetaFromType(fieldType)
		if err == nil && len(nestedMeta.Labels) > 0 {
			// Successfully extracted - merge labels, fields, relationships, and schema
			meta.Labels = append(meta.Labels, nestedMeta.Labels...)
			for k, v := range nestedMeta.FieldsToProps {
				meta.FieldsToProps[k] = v
			}
			for k, v := range nestedMeta.Relationships {
				meta.Relationships[k] = v
			}
			// Merge schema from nested node (but not auto-ID constraint - that will be regenerated)
			if nestedMeta.Schema != nil {
				for _, idx := range nestedMeta.Schema.Indexes {
					for _, prop := range idx.Properties {
						*fieldSchemas = append(*fieldSchemas, FieldSchemaInfo{
							DBName: prop.Name,
							Index: &IndexSpec{
								Name:     idx.Name,
								Type:     idx.Type,
								Priority: prop.Priority,
								Options:  idx.Options,
							},
						})
					}
				}
				// Don't merge constraints from nested - they'll be regenerated with correct label
				// Only merge explicit constraints (not auto-ID)
				for _, con := range nestedMeta.Schema.Constraints {
					// Skip auto-generated ID constraints (they have auto-generated names)
					if strings.HasPrefix(con.Name, "unique_") && len(con.Properties) == 1 && con.Properties[0] == "id" {
						continue
					}
					for i, prop := range con.Properties {
						*fieldSchemas = append(*fieldSchemas, FieldSchemaInfo{
							DBName: prop,
							Constraint: &ConstraintSpec{
								Name:     con.Name,
								Type:     con.Type,
								Priority: i + 1,
							},
						})
					}
				}
			}
			return false // Don't recurse further, we already extracted
		}
		// Extraction failed (no labels) - this is a base type, recurse to get fields
		return true
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
	options := parts[1:] // Additional options like "required"

	switch ident {
	case "<-", "->":
		// Direct relationship definition
		return r.registerRelationshipField(field, ident == "->", "", options, meta, typeName)

	case "":
		return fmt.Errorf("field has empty neo4j label / direction: %s.%s", typeName, field.Name)

	default:
		// Label or shorthand relationship
		if field.Anonymous {
			*postpendLabels = append(*postpendLabels, ident)
		} else {
			// Check if it's a shorthand relationship (starts with < or ends with >)
			if ident[0] == '<' {
				return r.registerRelationshipField(field, false, ident[1:], options, meta, typeName)
			} else if ident[len(ident)-1] == '>' {
				return r.registerRelationshipField(field, true, ident[:len(ident)-1], options, meta, typeName)
			}
			// If it's not a relationship and not anonymous, it's ignored
			// Labels only come from anonymous embedded structs
		}
	}

	return nil
}

// registerRelationshipField registers a relationship field.
// Only One[R] and Many[R] wrapper types are supported for relationship definitions.
func (r *CodecRegistry) registerRelationshipField(field reflect.StructField, dir bool, shorthand string, options []string, meta *Neo4jNodeMetadata, typeName string) error {
	relType := field.Type
	isMany := false
	isRequired := false

	// Parse options
	for _, opt := range options {
		opt = strings.TrimSpace(opt)
		if opt == "required" {
			isRequired = true
		}
	}

	// Only One[R] and Many[R] zero-cost wrapper types are supported
	if relType.Kind() != reflect.Struct {
		return fmt.Errorf("relationship field %s.%s must use One[R] or Many[R] type, got %s", typeName, field.Name, relType)
	}

	relWrapperName := relType.Name()
	if !strings.HasPrefix(relWrapperName, "One[") && !strings.HasPrefix(relWrapperName, "Many[") {
		return fmt.Errorf("relationship field %s.%s must use One[R] or Many[R] type, got %s", typeName, field.Name, relWrapperName)
	}

	isMany = strings.HasPrefix(relWrapperName, "Many[")

	// Extract the type parameter R from One[R] or Many[R]
	// The type parameter is accessible via the struct's first field type
	if relType.NumField() == 0 {
		return fmt.Errorf("relationship field %s.%s has invalid One/Many type structure", typeName, field.Name)
	}

	// The phantom field is [0]R, so its element type is R
	phantomField := relType.Field(0)
	if phantomField.Type.Kind() != reflect.Array || phantomField.Type.Len() != 0 {
		return fmt.Errorf("relationship field %s.%s has invalid One/Many type structure", typeName, field.Name)
	}

	innerType := phantomField.Type.Elem()
	// Wrap in pointer for consistency with existing code
	relType = reflect.PointerTo(innerType)

	// Determine relationship type
	var relTypeName string

	if shorthand != "" {
		relTypeName = shorthand
	} else {
		// Extract relationship type from db/neo4j tag on the relationship struct
		relStructType := relType.Elem()
		if r.implementsIRelationship(relType) {
			// Look for neo4j tag on the relationship struct itself
			if relStructType.Kind() == reflect.Struct {
				for i := 0; i < relStructType.NumField(); i++ {
					f := relStructType.Field(i)
					if f.Anonymous && f.Type.Name() == "Relationship" {
						if tagVal := f.Tag.Get("neo4j"); tagVal != "" {
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
		Required:  isRequired,
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