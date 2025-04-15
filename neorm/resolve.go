package neorm

import (
	"github.com/rlch/neogo/internal"
)

type (
	nodeSelection struct {
		querySelector
		reg  *internal.RegisteredNode
		next *relationshipSelection
	}
	relationshipSelection struct {
		querySelector
		reg  *internal.RegisteredRelationship
		next *nodeSelection
	}
)

func (t *tx) resolveQuery(root any, query querySpec) (head *nodeSelection, err error) {
	// target := t.registry.Get(reflect.TypeOf(root))
	// if target == nil {
	// 	return nil, errors.New("no target node/relationship provided")
	// }
	// var current *internal.RegisteredNode
	// switch target := target.(type) {
	// case *internal.RegisteredNode:
	// 	current = target
	// case *internal.RegisteredAbstractNode:
	// 	current = target.RegisteredNode
	// case *internal.RegisteredRelationship:
	// 	return nil, errors.New("cannot use relationship as root")
	// }
	//
	// if len(query.selectors) == 0 {
	// 	return nil, errors.New("no selectors provided")
	// }
	//
	// var (
	// 	i            int
	// 	rootSelector *querySelector
	// )
	// if rootSelector = query.selectors[0]; rootSelector.field == "." {
	// 	i++
	// } else {
	// 	rootSelector = &querySelector{
	// 		field: ".",
	// 		props: []string{},
	// 	}
	// }
	// head, err = t.resolveNodeSelector(current, rootSelector)
	// if err != nil {
	// 	return nil, err
	// }

	// var (
	// 	prevNodeSel *nodeSelection = head
	// 	prevRelSel  *relationshipSelection
	// )
	// for i < len(query.selectors) {
	// 	selector := query.selectors[i]
	// 	if selector.field == "." {
	// 		return nil, fmt.Errorf("ambiguous selector in query argument %d: %s", i, query.String())
	// 	}
	// 	var (
	// 		curRelSel *relationshipSelection
	// 		nextField string
	// 		nextNode  *internal.RegisteredNode
	// 	)
	// 	for field, rsTarget := range prevNodeSel.reg.Relationships {
	// 		if field == selector.field {
	// 			curRelSel, err = t.resolveRelationshipSelector(rsTarget, selector)
	// 			if err != nil {
	// 				return nil, err
	// 			}
	// 			nextNode = rsTarget.Target()
	// 			nextField = field
	// 			break
	// 		}
	// 	}
	// 	if curRelSel == nil {
	// 		return nil, fmt.Errorf("relationship %s not found in node %s", selector.field, prevNodeSel.reg.Name())
	// 	}
	// 	prevNodeSel.next = curRelSel
	// 	prevRelSel = curRelSel
	//
	// 	i++
	// 	if i == len(query.selectors) {
	// 		selector = &querySelector{field: "."}
	// 	} else {
	// 		selector = query.selectors[i]
	// 	}
	// 	if selector.field != "." && selector.field != nextField {
	// 		return nil, fmt.Errorf("field %s not found in relationship %s", nextField, curRelSel.reg.Name())
	// 	}
	// 	curNodeSel, err := t.resolveNodeSelector(nextNode, selector)
	// 	if err != nil {
	// 		return nil, err
	// 	}
	// 	prevRelSel.next = curNodeSel
	// 	prevNodeSel = curNodeSel
	// }
	return
}

func getPropsToFields(entity internal.RegisteredEntity) map[string]string {
	propsToFields := make(map[string]string)
	for field, prop := range entity.FieldsToProps() {
		propsToFields[prop] = field
	}
	return propsToFields
}
