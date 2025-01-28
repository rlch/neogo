package internal

import (
	"fmt"
	"reflect"
	"strings"
)

const neo4jTag = "neo4j"

func ExtractNodeLabels(i any) []string {
	labels := extractNodeLabels(i)
	if labels == nil {
		return nil
	}
	out := make([]string, len(labels))
	for i, l := range labels {
		out[i] = l.name
	}
	return out
}

func ExtractConcreteNodeLabels(i any) []string {
	labels := extractNodeLabels(i)
	if labels == nil {
		return nil
	}
	out := []string{}
	for _, l := range labels {
		if l.concrete {
			out = append(out, l.name)
		}
	}
	return out
}

func extractNodeLabels(i any) []neo4jName {
	if i == nil {
		return nil
	}
	if _, ok := i.(INode); !ok {
		v := reflect.ValueOf(i)
		for v.Kind() == reflect.Ptr {
			v = v.Elem()
			if n, ok := v.Interface().(INode); ok {
				return extractNodeLabels(n)
			}
		}
		if v.Kind() == reflect.Slice || v.Kind() == reflect.Array {
			return extractNodeLabels(reflect.Zero(v.Type().Elem()).Interface())
		}
		return nil
	}
	tags, err := extractNeo4JName(i)
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
	return tags[0].name
}

type neo4jName struct {
	name     string
	concrete bool
}

func extractNeo4JName(instance any, fields ...string) ([]neo4jName, error) {
	val := reflect.TypeOf(instance)
	for val.Kind() == reflect.Ptr {
		val = val.Elem()
	}
	if val.Kind() != reflect.Struct {
		return nil, fmt.Errorf("the type of %T is not a struct", instance)
	}
	tags := []neo4jName{}
	extractTagFromMatch := func(match *reflect.StructField) {
		if match == nil {
			return
		}
		label, ok := match.Tag.Lookup(neo4jTag)
		if !ok {
			return
		}
		name := strings.Split(label, ",")[0]
		concrete := match.Type.Name() != "Label"
		tags = append(tags, neo4jName{
			name:     name,
			concrete: concrete,
		})
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
