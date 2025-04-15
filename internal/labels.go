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
		inner = r.Get(outer).Type()
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
	return reg.Labels
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

func extractJSONFieldName(field reflect.StructField) (string, bool) {
	jsTag, ok := field.Tag.Lookup("json")
	if !ok {
		return "", false
	}
	tag := strings.Split(jsTag, ",")[0]
	if tag == "" || tag == "-" {
		return "", false
	}
	return tag, true
}
