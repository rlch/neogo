package internal

import (
	"fmt"
	"reflect"
	"strings"
)

const neo4jTag = "neo4j"

func ExtractNodeLabels(node any) []string {
	if node == nil {
		return nil
	}
	if _, ok := node.(INode); !ok {
		v := reflect.ValueOf(node)
		for v.Kind() == reflect.Ptr {
			v = v.Elem()
		}
		if v.Kind() == reflect.Slice || v.Kind() == reflect.Array {
			return ExtractNodeLabels(reflect.Zero(v.Type().Elem()).Interface())
		}
		return nil
	}
	tags, err := extractNeo4JName(node)
	if err != nil {
		return nil
	}
	return tags
}

func ExtractRelationshipType(relationship any) string {
	if relationship == nil {
		return ""
	}
	if _, ok := relationship.(IRelationship); !ok {
		v := reflect.ValueOf(relationship)
		for v.Kind() == reflect.Ptr {
			v = v.Elem()
		}
		if v.Kind() == reflect.Slice || v.Kind() == reflect.Array {
			return ExtractRelationshipType(reflect.Zero(v.Type().Elem()).Interface())
		}
		return ""
	}
	tags, err := extractNeo4JName(relationship)
	if err != nil {
		return ""
	}
	if len(tags) > 1 {
		panic("relationships with multiple types are not supported in Neo4J")
	} else if len(tags) == 0 {
		return ""
	}
	return tags[0]
}

func extractNeo4JName(instance any, fields ...string) ([]string, error) {
	val := reflect.TypeOf(instance)
	for val.Kind() == reflect.Ptr {
		val = val.Elem()
	}
	if val.Kind() != reflect.Struct {
		return nil, fmt.Errorf("the type of %T is not a struct", instance)
	}
	tags := []string{}
	extractTagFromMatch := func(match *reflect.StructField) {
		if match == nil {
			return
		}
		label, ok := match.Tag.Lookup(neo4jTag)
		if !ok {
			return
		}
		tags = append(tags, strings.Split(label, ",")[0])
	}
	if len(fields) > 0 {
		for _, field := range fields {
			f, ok := val.FieldByName(field)
			if ok {
				extractTagFromMatch(&f)
			}
		}
	} else {
		queue := []reflect.Type{val}
		for len(queue) > 0 {
			v := queue[0]
			queue = queue[1:]
			for i := 0; i < v.NumField(); i++ {
				f := v.Field(i)
				if f.Anonymous && f.Type.Kind() == reflect.Struct {
					queue = append(queue, f.Type)
				}
				if f.Type.Kind() == reflect.Struct {
					extractTagFromMatch(&f)
				}
			}
		}
	}
	for i, j := 0, len(tags)-1; i < j; i, j = i+1, j-1 {
		tags[i], tags[j] = tags[j], tags[i]
	}
	return tags, nil
}

func extractJSONFieldName(field reflect.StructField) (string, bool) {
	jsTag, ok := field.Tag.Lookup("json")
	if !ok {
		return "", false
	}
	return strings.Split(jsTag, ",")[0], true
}
