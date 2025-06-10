package internal

import (
	"errors"
	"fmt"
	"reflect"
	"strings"
)

type (
	querySpec struct {
		selectors []*querySelector
		condition ICondition
	}
	querySelector struct {
		// name is the name of the variable that is used in the query.
		name string
		// field is the name of the field should be loaded.
		field string
		// props represents the properties of the JSON payload that should be loaded into the struct.
		// If this is empty, all properties will be loaded. If this is nil, none will be loaded.
		props []string
	}
	nodeSelection struct {
		querySelector
		// alloc is the allocated value for this node. It is only populated when the selection is qualified.
		alloc any
		reg   *RegisteredNode
		next  *relationshipSelection
	}
	relationshipSelection struct {
		querySelector
		// alloc is the allocated value for this relationship. It is only populated when the selection is qualified.
		alloc any
		reg   *RegisteredRelationship
		next  *nodeSelection
	}
)

// func ResolveQuery(rootIdentifier any, query string) (Pattern, error) {
// 	querySpec, err := newQueryParser(query).parse()
// 	if err != nil {
// 		return  o
// 	}
// }

func resolveQuery(root any, query querySpec) func(r *Registry) (head *nodeSelection, err error) {
	return func(r *Registry) (head *nodeSelection, err error) {
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

		if len(query.selectors) == 0 {
			return nil, errors.New("no selectors provided")
		}

		var (
			i               int
			allocationDepth = 0
			rootSelector    *querySelector
		)
		if rootSelector = query.selectors[0]; rootSelector.field == "." {
			i++
		} else {
			rootSelector = &querySelector{
				field: ".",
				props: []string{},
			}
		}
		head = &nodeSelection{
			querySelector: *rootSelector,
			alloc:         root,
			reg:           current,
		}
		if reflect.TypeOf(root).Kind() == reflect.Slice {
			allocationDepth = 1
		}
		if err := head.validateProps(); err != nil {
			return nil, err
		}

		var (
			prevNodeSel = head
			prevRelSel  *relationshipSelection
		)
		for i < len(query.selectors) {
			selector := query.selectors[i]
			if selector.field == "." {
				return nil, fmt.Errorf("ambiguous selector in query argument %d: %s", i, query.String())
			}
			var (
				curRelSel *relationshipSelection
				nextField string
				nextNode  *RegisteredNode
			)
			allocate := func(typ reflect.Type) any {
				alloc := reflect.New(typ)
				for range allocationDepth {
					alloc = reflect.Zero(reflect.SliceOf(alloc.Type()))
				}
				return alloc.Interface()
			}
			for field, rsTarget := range prevNodeSel.reg.Relationships {
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
				curRelSel = &relationshipSelection{
					querySelector: *selector,
					reg:           rsTarget.Rel,
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
				return nil, fmt.Errorf("relationship %s not found in node %s", selector.field, prevNodeSel.reg.Name())
			}
			prevNodeSel.next = curRelSel
			prevRelSel = curRelSel

			i++
			if i == len(query.selectors) {
				selector = &querySelector{field: "."}
			} else {
				selector = query.selectors[i]
			}
			if selector.field != "." && selector.field != nextField {
				return nil, fmt.Errorf("field %s not found in relationship %s, expected %s or '.'", selector.field, curRelSel.reg.Name(), nextField)
			}
			alloc := allocate(nextNode.Type())
			curNodeSel := &nodeSelection{
				querySelector: *selector,
				reg:           nextNode,
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
}

func (n *nodeSelection) validateProps() error {
	ftp := n.reg.FieldsToProps()
	props := make(map[string]struct{}, len(ftp))
	for _, prop := range ftp {
		props[prop] = struct{}{}
	}
	for _, prop := range n.props {
		if _, ok := props[prop]; !ok {
			return fmt.Errorf("property %s not found in node %s", prop, n.reg.Name())
		}
	}
	return nil
}

func (r *relationshipSelection) validateProps() error {
	ftp := r.reg.FieldsToProps()
	props := make(map[string]struct{}, len(ftp))
	for _, prop := range ftp {
		props[prop] = struct{}{}
	}
	for _, prop := range r.props {
		if _, ok := props[prop]; !ok {
			return fmt.Errorf("property %s not found in relationship %s", prop, r.reg.Name())
		}
	}
	return nil
}

func (q querySpec) String() string {
	var buf strings.Builder
	for i, selector := range q.selectors {
		if selector.name != "" || selector.props != nil {
			buf.WriteString(selector.name)
			buf.WriteString(fmt.Sprintf("{%s}", strings.Join(selector.props, ",")))
			buf.WriteString(":")
		}
		buf.WriteString(selector.field)
		if i != len(q.selectors)-1 {
			buf.WriteString(" ")
		}
	}
	return buf.String()
}
