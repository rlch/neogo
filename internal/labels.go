package internal

import (
	"reflect"
	"strings"
)

const Neo4jTag = "neo4j"

func (r *Registry) instatiateInnerType(outer reflect.Type, to reflect.Type) any {
	satisfied := false
	for outer.Kind() == reflect.Ptr || outer.Kind() == reflect.Slice {
		if outer.Implements(to) {
			satisfied = true
			break
		}
		outer = outer.Elem()
	}
	if !satisfied && !outer.Implements(to) {
		return nil
	}
	var inner any
	if outer.Kind() == reflect.Ptr {
		inner = reflect.New(outer.Elem()).Interface()
	} else {
		inner = reflect.Zero(outer).Interface()
	}
	if inner == nil {
		if registered := r.Get(outer); registered != nil {
			inner = reflect.New(registered.Type()).Interface()
		}
	}
	return inner
}

func (r *Registry) ExtractNodeLabels(node any) []string {
	if node == nil {
		return nil
	}
	var (
		iNode INode
		ok    bool
	)
	if iNode, ok = node.(INode); !ok {
		n := r.instatiateInnerType(reflect.TypeOf(node), rINode)
		if n == nil {
			return nil
		}
		iNode = n.(INode)
	}
	reg := r.RegisterNode(iNode)
	if reg == nil {
		return nil
	}
	return reg.Labels()
}

func (r *Registry) ExtractRelationshipType(rel any) string {
	if rel == nil {
		return ""
	}
	var (
		iRel IRelationship
		ok   bool
	)
	if iRel, ok = rel.(IRelationship); !ok {
		r := r.instatiateInnerType(reflect.TypeOf(rel), rIRelationship)
		if r == nil {
			return ""
		}
		iRel = r.(IRelationship)
	}
	n := r.RegisterRelationship(iRel)
	if n == nil {
		return ""
	}
	return n.Reltype
}

// extractNeo4jFieldName extracts the Neo4j property name from a struct field tag.
// Returns the property name and true if the field should be mapped to a Neo4j property.
func extractNeo4jFieldName(field reflect.StructField) (string, bool) {
	// Anonymous (embedded) fields are for labels/inheritance, not properties
	if field.Anonymous {
		return "", false
	}

	// Use neo4j tag for property names
	if neo4jTag, ok := field.Tag.Lookup("neo4j"); ok && neo4jTag != "" && neo4jTag != "-" {
		tag := strings.Split(neo4jTag, ",")[0]
		// Skip relationship direction markers and other special values
		if tag != "->" && tag != "<-" && tag != "startNode" && tag != "endNode" {
			return tag, true
		}
	}

	return "", false
}
