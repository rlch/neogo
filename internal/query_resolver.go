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
		// name is the name of the variable that is used in the query.
		name string
		// field is the name of the field should be loaded.
		field string
		// props represents the properties of the JSON payload that should be loaded into the struct.
		// If this is empty, all properties will be loaded. If this is nil, none will be loaded.
		props []string
	}
	NodeSelection struct {
		QuerySelector
		// alloc is the allocated value for this node. It is only populated when the selection is qualified.
		alloc  any
		target *NodeTarget
		next   *RelationshipSelection
	}
	RelationshipSelection struct {
		QuerySelector
		// alloc is the allocated value for this relationship. It is only populated when the selection is qualified.
		alloc  any
		target *RelationshipTarget
		next   *NodeSelection
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
	for typ.Kind() == reflect.Slice {
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
	if rootSelector = selectors[0]; rootSelector.field == "." {
		i++
	} else {
		rootSelector = &QuerySelector{
			field: ".",
			props: []string{},
		}
	}
	head = &NodeSelection{
		QuerySelector: *rootSelector,
		alloc:         root,
		target: &NodeTarget{
			RegisteredNode: current,
		},
	}
	if reflect.TypeOf(root).Kind() == reflect.Slice {
		allocationDepth = 1
	}
	if err := head.validateProps(); err != nil {
		return nil, err
	}

	var (
		prevNodeSel = head
		prevRelSel  *RelationshipSelection
	)
	for i < len(selectors) {
		selector := selectors[i]
		if selector.field == "." {
			return nil, fmt.Errorf("ambiguous selector in query argument %d: %s", i, query)
		}
		var (
			curRelSel *RelationshipSelection
			nextField string
			nextNode  *NodeTarget
		)
		allocate := func(typ reflect.Type) any {
			alloc := reflect.New(typ)
			for range allocationDepth {
				alloc = reflect.Zero(reflect.SliceOf(alloc.Type()))
			}
			return alloc.Interface()
		}
		for field, rsTarget := range prevNodeSel.target.Relationships {
			if field != selector.field {
				continue
			}
			if rsTarget.Many {
				allocationDepth++
			}
			var alloc any
			// If the relationship is  shorthand, we don't need to allocate a value for it.
			if rType := rsTarget.Rel.Type(); rType != nil {
				alloc = allocate(rsTarget.Rel.Type())
			}
			curRelSel = &RelationshipSelection{
				QuerySelector: *selector,
				target:        rsTarget,
				alloc:         alloc,
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
			break
		}
		if curRelSel == nil {
			return nil, fmt.Errorf("relationship %s not found in node %s", selector.field, prevNodeSel.target.Name())
		}
		prevNodeSel.next = curRelSel
		prevRelSel = curRelSel

		i++
		if i == len(selectors) {
			selector = &QuerySelector{field: "."}
		} else {
			selector = selectors[i]
		}
		if selector.field != "." && selector.field != nextField {
			return nil, fmt.Errorf("field %s not found in relationship %s, expected %s or '.'", selector.field, curRelSel.target.Rel.Name(), nextField)
		}
		alloc := allocate(nextNode.Type())
		curNodeSel := &NodeSelection{
			QuerySelector: *selector,
			target:        nextNode,
			alloc:         alloc,
		}
		if err := curNodeSel.validateProps(); err != nil {
			return nil, err
		}
		prevRelSel.next = curNodeSel
		prevNodeSel = curNodeSel
		i++
	}
	return
}

func (n *NodeSelection) validateProps() error {
	ftp := n.target.FieldsToProps()
	props := make(map[string]struct{}, len(ftp))
	for _, prop := range ftp {
		props[prop] = struct{}{}
	}
	for _, prop := range n.props {
		if _, ok := props[prop]; !ok {
			return fmt.Errorf("property %s not found in node %s", prop, n.target.Name())
		}
	}
	return nil
}

func (r *RelationshipSelection) validateProps() error {
	ftp := r.target.Rel.FieldsToProps()
	props := make(map[string]struct{}, len(ftp))
	for _, prop := range ftp {
		props[prop] = struct{}{}
	}
	for _, prop := range r.props {
		if _, ok := props[prop]; !ok {
			return fmt.Errorf("property %s not found in relationship %s", prop, r.target.Rel.Name())
		}
	}
	return nil
}

func (q QuerySpec) String() string {
	var buf strings.Builder
	for i, selector := range q {
		if selector.name != "" || selector.props != nil {
			buf.WriteString(selector.name)
			buf.WriteString(fmt.Sprintf("{%s}", strings.Join(selector.props, ",")))
			buf.WriteString(":")
		}
		buf.WriteString(selector.field)
		if i != len(q)-1 {
			buf.WriteString(" ")
		}
	}
	return buf.String()
}
