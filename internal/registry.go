package internal

import (
	"fmt"
	"reflect"
	"strings"
)

type (
	Registry struct {
		AbstractNodes   []*RegisteredAbstractNode
		Nodes           []*RegisteredNode
		Relationships   []*RegisteredRelationship
		registeredTypes map[string]any
	}
	RegisteredAbstractNode struct {
		*RegisteredNode
		implementers []*RegisteredNode
	}
	RegisteredNode struct {
		name   string
		typ    any
		labels []string
	}
	RegisteredRelationship struct {
		name    string
		typ     any
		reltype string
	}
)

func (r *Registry) RegisterTypes(types ...any) {
	if r.registeredTypes == nil {
		r.registeredTypes = map[string]any{}
	}
	if r.AbstractNodes == nil {
		r.AbstractNodes = []*RegisteredAbstractNode{}
	}
	if r.Nodes == nil {
		r.Nodes = []*RegisteredNode{}
	}
	if r.Relationships == nil {
		r.Relationships = []*RegisteredRelationship{}
	}
	for _, t := range types {
		r.RegisterType(t)
	}
}

func (r *Registry) RegisterType(typ any) (registered any) {
	tName := r.getTypeName(typ)
	if r, ok := r.registeredTypes[tName]; ok {
		return r
	}
	r.registeredTypes[tName] = registered
	if abs, ok := typ.(IAbstract); ok {
		return r.RegisterAbstractNode(typ, abs, tName)
	}
	if v, ok := typ.(INode); ok {
		return r.RegisterNode(v, tName)
	}
	if v, ok := typ.(IRelationship); ok {
		fmt.Println("Registering relationship:", tName)
		return r.RegisterRelationship(v, tName)
	}
	return
}

func (r *Registry) RegisterNode(v INode, name string) *RegisteredNode {
	registered := &RegisteredNode{
		typ:    v,
		name:   name,
		labels: ExtractNodeLabels(v),
	}
	r.Nodes = append(r.Nodes, registered)
	r.registeredTypes[name] = registered
	return registered
}

func (r *Registry) RegisterAbstractNode(typ any, typAbs IAbstract, name string) *RegisteredAbstractNode {
	fmt.Println("Registering abstract node:", name)
	node := r.RegisterNode(typ.(INode), name)
	registered := &RegisteredAbstractNode{
		RegisteredNode: node,
	}
	registered.labels = ExtractNodeLabels(typ)
	impls := typAbs.Implementers()
	fmt.Println(impls)
	registered.implementers = make([]*RegisteredNode, len(impls))
	for i, impl := range impls {
		name := r.getTypeName(impl)
		registered.implementers[i] = r.RegisterNode(impl, name)
	}
	r.AbstractNodes = append(r.AbstractNodes, registered)
	r.registeredTypes[name] = registered
	return registered
}

func (r *Registry) RegisterRelationship(v IRelationship, name string) *RegisteredRelationship {
	registered := &RegisteredRelationship{typ: v, name: name}
	registered.reltype = ExtractRelationshipType(v)
	r.Relationships = append(r.Relationships, registered)
	r.registeredTypes[name] = registered
	return registered
}

func (r *Registry) Get(typ any) any {
	if typ == nil {
		return nil
	}
	tName := r.getTypeName(typ)
	fmt.Println("Get type name:", tName, r.registeredTypes)
	if v, ok := r.registeredTypes[tName]; ok {
		return v
	}
	if _, ok := typ.(IAbstract); ok {
		return r.registeredTypes["Base"+tName]
	}
	return nil
}

func (r *Registry) GetConcreteImplementation(nodeLabels []string) (*RegisteredNode, error) {
	var (
		abstractNode       *RegisteredAbstractNode
		closestImpl        *RegisteredNode
		inheritanceCounter int
		isNodeLabel        = make(map[string]struct{}, len(nodeLabels))
	)
	for _, label := range nodeLabels {
		isNodeLabel[label] = struct{}{}
	}
	// We find the abstract node (or exact implementation if registered) that has
	// a inheritance chain closest to the database node we're extracting from.
	if abstractNode == nil {
	Bases:
		for _, base := range r.AbstractNodes {
			labels := base.labels
			if len(labels) == 0 {
				continue
			}
			currentInheritanceCounter := 0
			for _, label := range labels {
				if _, ok := isNodeLabel[label]; !ok {
					continue Bases
				}
				currentInheritanceCounter++
			}
			if currentInheritanceCounter > inheritanceCounter {
				abstractNode = base
				inheritanceCounter = currentInheritanceCounter
			}
		}
		if abstractNode == nil {
			return nil, fmt.Errorf(
				"no abstract node found for labels: %s\nDid you forget to register the base node using neogo.WithTypes(...)?",
				strings.Join(nodeLabels, ", "),
			)
		}
	}
	if inheritanceCounter == len(nodeLabels) {
		return abstractNode.RegisteredNode, nil
	}
Impls:
	for _, nextImpl := range abstractNode.implementers {
		for _, label := range nextImpl.labels {
			if _, ok := isNodeLabel[label]; !ok {
				continue Impls
			}
		}
		closestImpl = nextImpl
		break
	}
	if closestImpl == nil {
		return nil, fmt.Errorf(
			"no concrete implementation found for labels: %s\nDid you forget to register the base node using neogo.WithTypes(...)?",
			strings.Join(nodeLabels, ", "),
		)
	}
	return closestImpl, nil
}

func (r *Registry) getTypeName(typ any) string {
	tt := reflect.TypeOf(typ)
	for tt.Kind() == reflect.Ptr {
		tt = tt.Elem()
	}
	return tt.Name()
}
