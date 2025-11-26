// Package scafadapter provides a schema adapter for the scaf testing framework.
// It extracts type schema information from neogo models for LSP completions
// and type information in .scaf files.
//
// This package lives in neogo because it needs access to neogo's internal
// registry and codec packages. Users should import this package directly
// when generating schema files.
//
// Usage:
//
//	adapter := scafadapter.New()
//	adapter.Register(
//	    &models.Person{},
//	    &models.Movie{},
//	    &models.ActedIn{},
//	)
//	schema, err := adapter.ExtractSchema()
//	if err != nil {
//	    log.Fatal(err)
//	}
//	json.NewEncoder(os.Stdout).Encode(schema)
package scafadapter

import (
	"fmt"
	"reflect"
	"strings"

	"github.com/rlch/neogo/internal"
	"github.com/rlch/neogo/internal/codec"
)

// TypeSchema represents the database schema extracted from user code.
// This is JSON-serializable and compatible with scaf's analysis.TypeSchema.
type TypeSchema struct {
	Models map[string]*Model `json:"models"`
}

// Model represents a database entity (node, relationship, table, etc.).
type Model struct {
	Name          string          `json:"name"`
	Fields        []*Field        `json:"fields,omitempty"`
	Relationships []*Relationship `json:"relationships,omitempty"`
}

// Field represents a property/column on a model.
type Field struct {
	Name     string `json:"name"`
	Type     string `json:"type"`
	Required bool   `json:"required,omitempty"`
}

// Relationship represents an edge from one model to another.
type Relationship struct {
	Name      string    `json:"name"`
	RelType   string    `json:"relType"`
	Target    string    `json:"target"`
	Many      bool      `json:"many,omitempty"`
	Direction Direction `json:"direction"`
}

// Direction represents the direction of a relationship.
type Direction string

const (
	DirectionOutgoing Direction = "outgoing"
	DirectionIncoming Direction = "incoming"
)

// Adapter extracts schema information from neogo models.
type Adapter struct {
	registry *internal.Registry
	types    []any
}

// New creates a new schema adapter.
func New() *Adapter {
	return &Adapter{
		registry: internal.NewRegistry(),
		types:    make([]any, 0),
	}
}

// Register adds types to be included in the schema.
// Types should be zero-value pointers (e.g., &Person{}).
func (a *Adapter) Register(types ...any) {
	a.types = append(a.types, types...)
}

// ExtractSchema discovers models from registered types and returns a TypeSchema.
func (a *Adapter) ExtractSchema() (*TypeSchema, error) {
	if len(a.types) == 0 {
		return &TypeSchema{Models: make(map[string]*Model)}, nil
	}

	// Register all types with neogo's registry
	a.registry.RegisterTypes(a.types...)

	schema := &TypeSchema{
		Models: make(map[string]*Model),
	}

	// Extract nodes
	for _, node := range a.registry.Nodes {
		model, err := a.extractNodeModel(node)
		if err != nil {
			return nil, fmt.Errorf("extracting node %s: %w", node.Name(), err)
		}
		schema.Models[model.Name] = model
	}

	// Extract relationships
	for _, rel := range a.registry.Relationships {
		model, err := a.extractRelationshipModel(rel)
		if err != nil {
			return nil, fmt.Errorf("extracting relationship %s: %w", rel.Name(), err)
		}
		if model != nil {
			schema.Models[model.Name] = model
		}
	}

	return schema, nil
}

// extractNodeModel converts a RegisteredNode to a Model.
func (a *Adapter) extractNodeModel(node *internal.RegisteredNode) (*Model, error) {
	model := &Model{
		Name:   node.Name(),
		Fields: make([]*Field, 0),
	}

	// Extract fields from FieldsToProps
	fieldsToProps := node.FieldsToProps()
	if fieldsToProps != nil {
		nodeType := node.Type()
		if nodeType != nil {
			model.Fields = a.extractFields(nodeType, fieldsToProps)
		}
	}

	// Extract relationships
	if node.Relationships != nil {
		model.Relationships = make([]*Relationship, 0, len(node.Relationships))
		for fieldName, relTarget := range node.Relationships {
			rel := a.extractRelationship(fieldName, relTarget, model.Name)
			model.Relationships = append(model.Relationships, rel)
		}
	}

	return model, nil
}

// extractRelationshipModel converts a RegisteredRelationship to a Model.
func (a *Adapter) extractRelationshipModel(rel *internal.RegisteredRelationship) (*Model, error) {
	// Skip empty/shorthand relationships (they don't have their own model)
	if rel.Name() == "" {
		return nil, nil
	}

	model := &Model{
		Name:   rel.Name(),
		Fields: make([]*Field, 0),
	}

	// Extract fields
	fieldsToProps := rel.FieldsToProps()
	if fieldsToProps != nil {
		relType := rel.Type()
		if relType != nil {
			model.Fields = a.extractFields(relType, fieldsToProps)
		}
	}

	return model, nil
}

// extractFields extracts Field definitions from a struct type.
func (a *Adapter) extractFields(typ reflect.Type, fieldsToProps map[string]string) []*Field {
	if typ.Kind() == reflect.Ptr {
		typ = typ.Elem()
	}
	if typ.Kind() != reflect.Struct {
		return nil
	}

	fields := make([]*Field, 0)

	for goFieldName, dbFieldName := range fieldsToProps {
		// Skip startNode/endNode - these are relationship navigation, not stored properties
		if dbFieldName == "startNode" || dbFieldName == "endNode" {
			continue
		}

		// Find the struct field
		field, ok := typ.FieldByName(goFieldName)
		if !ok {
			continue
		}

		// Skip embedded Node/Relationship base types
		if field.Anonymous {
			continue
		}

		// Determine the type string
		typeStr := a.typeToString(field.Type)

		// Check if required (pointer types are optional)
		required := field.Type.Kind() != reflect.Ptr

		fields = append(fields, &Field{
			Name:     dbFieldName,
			Type:     typeStr,
			Required: required,
		})
	}

	return fields
}

// extractRelationship converts a RelationshipTarget to a Relationship.
func (a *Adapter) extractRelationship(fieldName string, target *internal.RelationshipTarget, sourceNodeName string) *Relationship {
	rel := &Relationship{
		Name:    fieldName,
		RelType: target.Rel.Reltype,
		Many:    target.Many,
	}

	// Direction: Dir=true means outgoing (->), Dir=false means incoming (<-)
	if target.Dir {
		rel.Direction = DirectionOutgoing
	} else {
		rel.Direction = DirectionIncoming
	}

	// Determine target based on relationship type
	relName := target.Rel.Name()
	if relName != "" {
		// This is a relationship struct
		// Try to find the actual end/start node from the relationship struct metadata
		if relMeta := a.registry.Codecs().GetRelMeta(relName); relMeta != nil {
			if target.Dir && relMeta.EndNode != nil {
				// Outgoing relationship - target is the EndNode type
				rel.Target = relMeta.EndNode.NodeType.Elem().Name()
			} else if !target.Dir && relMeta.StartNode != nil {
				// Incoming relationship - target is the StartNode type
				rel.Target = relMeta.StartNode.NodeType.Elem().Name()
			} else {
				// Fallback to relationship struct name
				rel.Target = relName
			}
		} else {
			// No metadata found - use relationship struct name
			rel.Target = relName
		}
	} else {
		// Shorthand relationship - find the target node
		startNode := target.Rel.StartNode.RegisteredNode
		endNode := target.Rel.EndNode.RegisteredNode

		if startNode != nil && startNode.Name() != sourceNodeName {
			rel.Target = startNode.Name()
		} else if endNode != nil && endNode.Name() != sourceNodeName {
			rel.Target = endNode.Name()
		} else if startNode != nil {
			// Self-referential
			rel.Target = startNode.Name()
		}
	}

	return rel
}

// typeToString converts a reflect.Type to a string representation.
func (a *Adapter) typeToString(typ reflect.Type) string {
	switch typ.Kind() {
	case reflect.Ptr:
		return "*" + a.typeToString(typ.Elem())
	case reflect.Slice:
		return "[]" + a.typeToString(typ.Elem())
	case reflect.Array:
		return fmt.Sprintf("[%d]%s", typ.Len(), a.typeToString(typ.Elem()))
	case reflect.Map:
		return fmt.Sprintf("map[%s]%s", a.typeToString(typ.Key()), a.typeToString(typ.Elem()))
	default:
		// For named types from other packages, include the package
		if typ.PkgPath() != "" && !isBuiltinType(typ.Name()) {
			// Use short package name
			pkgPath := typ.PkgPath()
			parts := strings.Split(pkgPath, "/")
			pkgName := parts[len(parts)-1]
			return pkgName + "." + typ.Name()
		}
		return typ.Name()
	}
}

// isBuiltinType returns true if the type name is a Go builtin.
func isBuiltinType(name string) bool {
	switch name {
	case "bool", "string",
		"int", "int8", "int16", "int32", "int64",
		"uint", "uint8", "uint16", "uint32", "uint64", "uintptr",
		"byte", "rune",
		"float32", "float64",
		"complex64", "complex128",
		"error":
		return true
	}
	return false
}

// GetCodecRegistry returns the underlying codec registry for advanced use cases.
func (a *Adapter) GetCodecRegistry() *codec.CodecRegistry {
	return a.registry.Codecs()
}
