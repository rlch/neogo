package codec

import (
	"fmt"
	"reflect"
)

// UnwindValue unwraps pointer values to get to the underlying value
func UnwindValue(ptrTo reflect.Value) reflect.Value {
	for ptrTo.Kind() == reflect.Ptr {
		ptrTo = ptrTo.Elem()
	}
	return ptrTo
}

// UnwindType unwraps pointer types to get to the underlying type
func UnwindType(ptrTo reflect.Type) reflect.Type {
	for ptrTo.Kind() == reflect.Ptr {
		ptrTo = ptrTo.Elem()
	}
	return ptrTo
}

// WalkStruct walks through struct fields with a visitor function
// This replaces the WalkStruct from helpers.go
func WalkStruct(
	v reflect.Value,
	visit func(
		i int,
		typ reflect.StructField,
		val reflect.Value,
	) (recurseInto bool, err error),
) (recurseInto error) {
	vt := v.Type()
	for vt.Kind() == reflect.Ptr {
		vt = vt.Elem()
		v = v.Elem()
	}
	if vt.Kind() != reflect.Struct || v.Kind() != reflect.Struct {
		return fmt.Errorf("expected struct, got %s", vt.Kind())
	}
	for i := range vt.NumField() {
		vtf := vt.Field(i)
		vf := v.Field(i)
		shouldRecurse, err := visit(i, vtf, vf)
		if err != nil {
			return err
		}
		if shouldRecurse && vf.Kind() == reflect.Struct {
			if err := WalkStruct(vf, visit); err != nil {
				return err
			}
			continue
		}
	}
	return nil
}

// GetTypeName returns the type name for a given value, unwrapping pointers
func (r *CodecRegistry) GetTypeName(v any) string {
	typ := reflect.TypeOf(v)
	return UnwindType(typ).Name()
}

// GetTypeByName returns a registered type by its name
func (r *CodecRegistry) GetTypeByName(name string) reflect.Type {
	return r.GetTypeByNameLookup(name)
}

// IsRegistered checks if a type is already registered in the codec
func (r *CodecRegistry) IsRegistered(v any) bool {
	typePtr := getTypePtrFromValue(v)
	_, exists := r.metadata[typePtr]
	return exists
}

// CreateInstance creates a new instance of the given type
func (r *CodecRegistry) CreateInstance(typ reflect.Type) (any, error) {
	if typ.Kind() == reflect.Ptr {
		elemType := typ.Elem()
		if elemType.Kind() != reflect.Struct {
			return nil, fmt.Errorf("expected pointer to struct, got pointer to %s", elemType.Kind())
		}
		return reflect.New(elemType).Interface(), nil
	} else if typ.Kind() == reflect.Struct {
		return reflect.New(typ).Interface(), nil
	}
	return nil, fmt.Errorf("unsupported type kind: %s", typ.Kind())
}

// Common interface types for checking implementations
var (
	rAbstract      = reflect.TypeOf((*IAbstract)(nil)).Elem()
	rINode         = reflect.TypeOf((*INode)(nil)).Elem()
	rIRelationship = reflect.TypeOf((*IRelationship)(nil)).Elem()
	rNode          = reflect.TypeOf(Node{})
)

// Interface definitions (these match what's in internal/entity.go)
type IAbstract interface {
	IsNode()
	GetID() string
	IsAbstract()
	Implementers() []IAbstract
}

type INode interface {
	IsNode()
	GetID() string
}

type IRelationship interface {
	IsRelationship()
}

// Node struct (this matches what's in internal/entity.go)
type Node struct {
	ID string `neo4j:"id"`
}

// ImplementsAbstract checks if a type implements IAbstract
func (r *CodecRegistry) ImplementsAbstract(typ reflect.Type) bool {
	return typ.Implements(rAbstract)
}

// ImplementsINode checks if a type implements INode
func (r *CodecRegistry) ImplementsINode(typ reflect.Type) bool {
	return typ.Implements(rINode)
}

// ImplementsIRelationship checks if a type implements IRelationship
func (r *CodecRegistry) ImplementsIRelationship(typ reflect.Type) bool {
	return typ.Implements(rIRelationship)
}

// IsBaseNodeType checks if a type is the base Node type
func (r *CodecRegistry) IsBaseNodeType(typ reflect.Type) bool {
	return typ == rNode
}
