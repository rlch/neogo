package internal

import (
	"fmt"
	"maps"
	"reflect"
	"strings"
)

type (
	Registry struct {
		AbstractNodes   []*RegisteredAbstractNode
		Nodes           []*RegisteredNode
		Relationships   []*RegisteredRelationship
		registeredTypes map[string]RegisteredEntity
	}
	RegisteredEntity interface {
		Name() string
		Type() reflect.Type
		FieldsToProps() map[string]string
	}
	RegisteredAbstractNode struct {
		*RegisteredNode
		Implementers []*RegisteredNode
	}
	RegisteredNode struct {
		name          string
		rType         reflect.Type
		fieldsToProps map[string]string

		Labels        []string
		Relationships map[string]*RelationshipTarget
	}
	RegisteredRelationship struct {
		name          string
		rType         reflect.Type
		fieldsToProps map[string]string

		Reltype   string
		StartNode NodeTarget
		EndNode   NodeTarget
	}
	NodeTarget struct {
		Field string
		*RegisteredNode
	}
	RelationshipTarget struct {
		// Many will be true if there is a (one/many)-to-many relationship between the source and target node.
		Many bool
		// true = ->, false = <-
		Dir bool
		Rel *RegisteredRelationship
	}
)

func (r *RegisteredNode) Name() string {
	return r.name
}

func (r *RegisteredNode) Type() reflect.Type {
	return r.rType
}

func (r *RegisteredNode) ReflectType() any {
	return r.rType
}

func (r *RegisteredNode) FieldsToProps() map[string]string {
	return r.fieldsToProps
}

func (r *RegisteredRelationship) Name() string {
	return r.name
}

func (r *RegisteredRelationship) Type() reflect.Type {
	return r.rType
}

func (r *RegisteredRelationship) FieldsToProps() map[string]string {
	return r.fieldsToProps
}

func (r RelationshipTarget) Target() *NodeTarget {
	if r.Dir {
		return &r.Rel.EndNode
	} else {
		return &r.Rel.StartNode
	}
}

var (
	rAbstract      = reflect.TypeOf((*IAbstract)(nil)).Elem()
	rINode         = reflect.TypeOf((*INode)(nil)).Elem()
	rIRelationship = reflect.TypeOf((*IRelationship)(nil)).Elem()
	rNode          = reflect.TypeOf(Node{})
)

func NewRegistry() *Registry {
	return &Registry{
		AbstractNodes:   []*RegisteredAbstractNode{},
		Nodes:           []*RegisteredNode{},
		Relationships:   []*RegisteredRelationship{},
		registeredTypes: make(map[string]RegisteredEntity),
	}
}

func (r *Registry) RegisterTypes(types ...any) {
	for _, t := range types {
		r.RegisterType(t)
	}
}

func (r *Registry) RegisterType(typ any) (registered any) {
	if abs, ok := typ.(IAbstract); ok {
		return r.RegisterAbstractNode(typ, abs)
	} else if v, ok := typ.(INode); ok {
		return r.RegisterNode(v)
	} else if v, ok := typ.(IRelationship); ok {
		return r.RegisterRelationship(v)
	}
	return
}

func (r *Registry) RegisterNode(v INode) *RegisteredNode {
	vv := UnwindValue(reflect.ValueOf(v))
	vvt := vv.Type()
	name := vvt.Name()
	if n, ok := r.registeredTypes[name]; ok {
		if reg, ok := n.(*RegisteredNode); ok {
			return reg
		} else {
			return n.(*RegisteredAbstractNode).RegisteredNode
		}
	}
	registered := &RegisteredNode{
		rType:         vvt,
		name:          name,
		Labels:        []string{},
		fieldsToProps: make(map[string]string),
		Relationships: make(map[string]*RelationshipTarget),
	}

	r.Nodes = append(r.Nodes, registered)
	r.registeredTypes[name] = registered
	// These are the labels to register after walking the node. We want to preference
	// labels from anonymous nodes as they're lower in the inheritance hierarchy.
	postpendLabels := []string{}
	err := WalkStruct(
		vv,
		func(i int, typ reflect.StructField, val reflect.Value) (bool, error) {
			jsonName, ok := extractJSONFieldName(typ)
			if ok {
				registered.fieldsToProps[typ.Name] = jsonName
			}
			// If a field with a neo4j tag is anonymous, we need to register it as a node.
			// Special case for neogo.Node given it shouldn't be registered.
			var shouldRecurse bool
			if typ.Type == rNode {
				shouldRecurse = true
			} else if typ.Anonymous {
				isNode := typ.Type.Implements(rINode)
				if isNode {
					nested := r.RegisterNode(reflect.New(typ.Type).Interface().(INode))
					registered.Labels = append(registered.Labels, nested.Labels...)
					maps.Copy(registered.fieldsToProps, nested.fieldsToProps)
					maps.Copy(registered.Relationships, nested.Relationships)
				} else {
					shouldRecurse = true
				}
			}
			neo4jTag, ok := typ.Tag.Lookup(Neo4jTag)
			if !ok || neo4jTag == "" {
				return shouldRecurse, nil
			}

			parts := strings.Split(neo4jTag, ",")
			if len(parts) == 0 {
				return false, fmt.Errorf("invalid tag format for field %s.%s: %s", vvt.Name(), typ.Name, neo4jTag)
			}
			registerRelationshipField := func(dir bool, shorthand string) error {
				relType := typ.Type
				isMany := false
				if relType.Kind() == reflect.Struct {
					relField, ok := relType.FieldByName("S")
					if !ok {
						return fmt.Errorf("expected Many[T] for field %s.%s, where T is some node or relationship", vvt.Name(), typ.Name)
					}
					relType = relField.Type.Elem()
					isMany = true
				}
				if relType.Kind() != reflect.Ptr {
					return fmt.Errorf("invalid relationship for field %s.%s. Got %s", vvt.Name(), typ.Name, relType)
				}
				var relReg *RegisteredRelationship
				if shorthand != "" {
					node, ok := reflect.New(relType.Elem()).Interface().(INode)
					if !ok {
						return fmt.Errorf("expected a pointer to a struct, implementing INode for field %s.%s", vvt.Name(), typ.Name)
					}
					target := r.RegisterNode(node)
					relReg = &RegisteredRelationship{Reltype: shorthand}
					if dir {
						relReg.StartNode = NodeTarget{RegisteredNode: target}
						relReg.EndNode = NodeTarget{RegisteredNode: registered}
					} else {
						relReg.StartNode = NodeTarget{RegisteredNode: registered}
						relReg.EndNode = NodeTarget{RegisteredNode: target}
					}
					r.registeredTypes[shorthand] = relReg
				} else {
					rel, ok := reflect.New(relType.Elem()).Interface().(IRelationship)
					if !ok {
						return fmt.Errorf("expected a pointer to a struct, implementing IRelationship for field %s.%s", vvt.Name(), typ.Name)
					}
					relReg = r.RegisterRelationship(rel)
				}
				registered.Relationships[typ.Name] = &RelationshipTarget{
					Dir:  dir,
					Rel:  relReg,
					Many: isMany,
				}
				return nil
			}
			ident := parts[0]
			switch ident {
			case "<-", "->":
				if err := registerRelationshipField(ident == "->", ""); err != nil {
					return false, err
				}
			case "":
				return false, fmt.Errorf("field has empty neo4j label / direction: %s.%s", vvt.Name(), typ.Name)
			default:
				// TODO: There should be labels that aren't bindable (created with neogo.Label)
				if typ.Anonymous {
					postpendLabels = append(postpendLabels, ident)
				} else {
					var err error
					if ident[0] == '<' {
						err = registerRelationshipField(false, ident[1:])
					} else if ident[len(ident)-1] == '>' {
						err = registerRelationshipField(true, ident[:len(ident)-1])
					}
					if err != nil {
						return false, err
					}
				}
			}

			return shouldRecurse, nil
		},
	)
	if err != nil {
		panic(err)
	}
	registered.Labels = append(registered.Labels, postpendLabels...)
	if len(registered.Labels) == 0 {
		panic(fmt.Errorf("node %s has no labels", name))
	}
	if len(registered.Relationships) == 0 {
		registered.Relationships = nil
	}
	return registered
}

func (r *Registry) RegisterAbstractNode(typ any, typAbs IAbstract) *RegisteredAbstractNode {
	vv := UnwindValue(reflect.ValueOf(typ))
	name := vv.Type().Name()
	// There's a chance that the abstract node is registered as a concrete node, in which case we re-register
	var node *RegisteredNode
	if n, ok := r.registeredTypes[name]; ok {
		if reg, ok := n.(*RegisteredNode); ok {
			node = reg
		} else {
			return n.(*RegisteredAbstractNode)
		}
	} else {
		node = r.RegisterNode(typ.(INode))
	}
	registered := &RegisteredAbstractNode{
		RegisteredNode: node,
	}
	impls := typAbs.Implementers()
	registered.Implementers = make([]*RegisteredNode, len(impls))
	for i, impl := range impls {
		registered.Implementers[i] = r.RegisterNode(impl)
	}
	r.AbstractNodes = append(r.AbstractNodes, registered)
	r.registeredTypes[name] = registered
	return registered
}

func (r *Registry) RegisterRelationship(v IRelationship) *RegisteredRelationship {
	vv := UnwindValue(reflect.ValueOf(v))
	vvt := vv.Type()
	name := vvt.Name()
	if v, ok := r.registeredTypes[name]; ok {
		return v.(*RegisteredRelationship)
	}
	registered := &RegisteredRelationship{
		name:          name,
		rType:         vvt,
		fieldsToProps: make(map[string]string),
	}
	r.Relationships = append(r.Relationships, registered)
	r.registeredTypes[name] = registered
	err := WalkStruct(
		vv,
		func(i int, typ reflect.StructField, val reflect.Value) (bool, error) {
			jsonName, ok := extractJSONFieldName(typ)
			if ok {
				registered.fieldsToProps[typ.Name] = jsonName
			}
			neo4jTag, ok := typ.Tag.Lookup(Neo4jTag)
			if !ok {
				return false, nil
			}
			parts := strings.Split(neo4jTag, ",")
			if len(parts) == 0 {
				return false, fmt.Errorf("invalid tag format for field %s.%s: %s", vvt.Name(), typ.Name, neo4jTag)
			}
			switch parts[0] {
			case "startNode", "endNode":
				isStart := parts[0] == "startNode"
				nodeType := typ.Type
				if nodeType.Kind() != reflect.Ptr {
					return false, fmt.Errorf("expected a pointer to a struct, implementing INode for field %s.%s", vvt.Name(), typ.Name)
				}
				node, ok := reflect.New(nodeType.Elem()).Interface().(INode)
				if !ok {
					return false, fmt.Errorf("expected a pointer to a struct, implementing INode for field %s.%s", vvt.Name(), typ.Name)
				}
				nodeReg := r.RegisterNode(node)
				if isStart {
					registered.StartNode = NodeTarget{
						Field:          typ.Name,
						RegisteredNode: nodeReg,
					}
				} else {
					registered.EndNode = NodeTarget{
						Field:          typ.Name,
						RegisteredNode: nodeReg,
					}
				}
			case "":
				return false, fmt.Errorf("field has empty neo4j label: %s.%s", vvt.Name(), typ.Name)
			default:
				if registered.Reltype != "" {
					return false, fmt.Errorf("relationship has multiple neo4j labels: %s.%s", vvt.Name(), typ.Name)
				}
				registered.Reltype = parts[0]
			}
			return false, nil
		},
	)
	if err != nil {
		panic(err)
	}
	if registered.Reltype == "" {
		panic(fmt.Errorf("relationship %s has no type", name))
	}
	if len(registered.fieldsToProps) == 0 {
		registered.fieldsToProps = nil
	}
	return registered
}

func (r *Registry) Get(typ reflect.Type) (entity RegisteredEntity) {
	if typ == nil {
		return nil
	}
	name := UnwindType(typ).Name()
	if v, ok := r.registeredTypes[name]; ok {
		return v
	}
	if typ.Implements(rAbstract) {
		return r.registeredTypes["Base"+name]
	}
	return nil
}

func (r *Registry) GetByName(name string) (entity RegisteredEntity) {
	return r.registeredTypes[name]
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
Bases:
	for _, base := range r.AbstractNodes {
		labels := base.Labels
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
	if inheritanceCounter == len(nodeLabels) {
		return abstractNode.RegisteredNode, nil
	}
Impls:
	for _, nextImpl := range abstractNode.Implementers {
		for _, label := range nextImpl.Labels {
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
