package internal

import (
	"errors"
	"fmt"
	"reflect"
	"strings"
)

type (
	QuerySpec     []*QuerySelector
	QuerySelector struct {
		// Name is the Name of the variable that is used in the query.
		Name string
		// Field is the name of the Field should be loaded.
		Field string
		// Props represents the properties of the JSON payload that should be loaded into the struct.
		// If this is empty, all properties will be loaded. If this is nil, none will be loaded.
		Props []string
	}
	NodeSelection struct {
		QuerySelector
		Alloc   any
		Payload any
		Target  *NodeTarget
		Next    *RelationshipSelection
	}
	RelationshipSelection struct {
		QuerySelector
		Alloc   any
		Payload any
		Target  *RelationshipTarget
		Next    *NodeSelection
	}
)

func ResolveQuery(r *Registry, root any, query string) (head *NodeSelection, err error) {
	selectors, err := newQueryParser(query).parse()
	if err != nil {
		return nil, fmt.Errorf("failed to parse query %q: %w", query, err)
	}
	if root == nil {
		return nil, errors.New("no target node/relationship provided")
	}
	typ := reflect.TypeOf(root)
	isSlice := false
	for typ.Kind() == reflect.Slice {
		isSlice = true
		typ = typ.Elem()
	}
	target := r.Get(typ)
	if target == nil {
		return nil, errors.New("no target node/relationship provided")
	}
	var current *RegisteredNode
	switch target := target.(type) {
	case *RegisteredNode:
		current = target
	case *RegisteredAbstractNode:
		current = target.RegisteredNode
	case *RegisteredRelationship:
		return nil, errors.New("cannot use relationship as root")
	}

	if len(selectors) == 0 {
		return nil, errors.New("no selectors provided")
	}

	var (
		i               int
		allocationDepth = 0
		rootSelector    *QuerySelector
	)
	if rootSelector = selectors[0]; rootSelector.Field == "." {
		i++
	} else {
		rootSelector = &QuerySelector{
			Field: ".",
			Props: []string{},
		}
	}
	head = &NodeSelection{
		QuerySelector: *rootSelector,
		Alloc:         root,
		Target: &NodeTarget{
			RegisteredNode: current,
		},
	}
	if isSlice {
		allocationDepth = 1
	}
	if err := head.validateProps(); err != nil {
		return nil, err
	}

	var (
		prevNodeSel = head
		prevRelSel  *RelationshipSelection
		curV        = reflect.ValueOf(root)
	)
	for i < len(selectors) {
		selector := selectors[i]
		if selector.Field == "." {
			return nil, fmt.Errorf("ambiguous selector in query argument %d: %s", i, query)
		}
		var (
			curRelSel *RelationshipSelection
			nextField string
			nextNode  *NodeTarget
		)
		allocate := func(typ reflect.Type) any {
			alloc := reflect.New(typ)
			effDepth := allocationDepth
			for range effDepth {
				alloc = reflect.Zero(reflect.SliceOf(alloc.Type()))
			}
			return alloc.Interface()
		}
		rsTarget, ok := prevNodeSel.Target.Relationships[selector.Field]
		if !ok {
			return nil, fmt.Errorf("relationship %s not found in node %s", selector.Field, prevNodeSel.Target.Name())
		}
		curRelSel = &RelationshipSelection{
			QuerySelector: *selector,
			Target:        rsTarget,
		}
		if err := curRelSel.validateProps(); err != nil {
			return nil, err
		}
		nextNode = rsTarget.Target()
		if rsTarget.Dir {
			nextField = rsTarget.Rel.EndNode.Field
		} else {
			nextField = rsTarget.Rel.StartNode.Field
		}
		prevNodeSel.Next = curRelSel
		prevRelSel = curRelSel

		i++
		if i == len(selectors) {
			selector = &QuerySelector{Field: "."}
		} else {
			selector = selectors[i]
		}
		if selector.Field != "." && selector.Field != nextField {
			return nil, fmt.Errorf("field %s not found in relationship %s, expected %s or '.'", selector.Field, curRelSel.Target.Rel.Name(), nextField)
		}
		curNodeSel := &NodeSelection{
			QuerySelector: *selector,
			Target:        nextNode,
		}
		if err := curNodeSel.validateProps(); err != nil {
			return nil, err
		}

		// Create allocations
		// We check for the case where the depth of the result needs to be increased by 1.
		// This is where we're returning either the node or relationship in a *-to-many relationship.
		if rsTarget.Many && (curRelSel.Name != "" || curNodeSel.Name != "") {
			allocationDepth++
		}
		// If the relationship is shorthand, we don't need to allocate a value for it.
		rType := rsTarget.Rel.Type()
		if rType != nil {
			curRelSel.Alloc = allocate(rType)
		}
		curNodeSel.Alloc = allocate(nextNode.Type())

		attachPayload := func() {
			curV = UnwindValue(curV)
			if curV.Kind() != reflect.Struct || curV.IsZero() {
				return
			}
			relV := curV.FieldByName(curRelSel.Field)
			if rType != nil {
				if curRelSel.Target.Many {
					relV = UnwindValue(relV)
					if relV.Kind() != reflect.Struct {
						return
					}
					relV = relV.FieldByName("V")
					if !relV.IsValid() || !relV.CanInterface() {
						return
					}
				}
				curRelSel.Payload = relV.Interface()
			}
			relV = UnwindValue(relV)
			if relV.Kind() != reflect.Struct {
				return
			}
			curV = relV.FieldByName(curNodeSel.Target.Field)
			// Handle shorthand relationships
			if rType == nil && curRelSel.Target.Many {
				curV = UnwindValue(curV)
				if curV.Kind() != reflect.Struct {
					return
				}
				curV = relV.FieldByName("V")
				if !curV.IsValid() {
					return
				}
			}
			if !curV.CanInterface() {
				return
			}
			curNodeSel.Payload = curV.Interface()
			curV = UnwindValue(curV)
		}
		attachPayload()
		prevRelSel.Next = curNodeSel
		prevNodeSel = curNodeSel
		i++
	}
	return
}

func (n *NodeSelection) validateProps() error {
	ftp := n.Target.FieldsToProps()
	props := make(map[string]struct{}, len(ftp))
	for _, prop := range ftp {
		props[prop] = struct{}{}
	}
	for _, prop := range n.Props {
		if _, ok := props[prop]; !ok {
			return fmt.Errorf("property %s not found in node %s", prop, n.Target.Name())
		}
	}
	return nil
}

func (r *RelationshipSelection) validateProps() error {
	ftp := r.Target.Rel.FieldsToProps()
	props := make(map[string]struct{}, len(ftp))
	for _, prop := range ftp {
		props[prop] = struct{}{}
	}
	for _, prop := range r.Props {
		if _, ok := props[prop]; !ok {
			return fmt.Errorf("property %s not found in relationship %s", prop, r.Target.Rel.Name())
		}
	}
	return nil
}

func (q QuerySpec) String() string {
	var buf strings.Builder
	for i, selector := range q {
		if selector.Name != "" || selector.Props != nil {
			buf.WriteString(selector.Name)
			buf.WriteString(fmt.Sprintf("{%s}", strings.Join(selector.Props, ",")))
			buf.WriteString(":")
		}
		buf.WriteString(selector.Field)
		if i != len(q)-1 {
			buf.WriteString(" ")
		}
	}
	return buf.String()
}
