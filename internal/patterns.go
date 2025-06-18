package internal

import (
	"errors"
	"reflect"
)

type (
	Pattern interface {
		Patterns
		ICondition

		createNodePattern(*Registry) *nodePatternPart

		setRelationship(
			dir *bool,
			relationshipMatch, nodeMatch any,
		) Pattern
		Related(relationshipMatch, nodeMatch any) Pattern
		From(relationshipMatch, nodeMatch any) Pattern
		To(relationshipMatch, nodeMatch any) Pattern
	}

	Patterns interface {
		nodes(*Registry) []*nodePatternPart
	}
)

var (
	_ interface {
		Pattern
		// A path can be used in a WHERE condition
		WhereOption
	} = (*CypherPattern)(nil)
	_ Patterns = (*CypherPatterns)(nil)
)

type (
	CypherPattern struct {
		resolver func(*Registry) *nodePatternPart
	}
	CypherPatterns struct {
		resolver func(*Registry) []*nodePatternPart
	}
	nodePatternPart struct {
		pathName string
		// Defined for the head node of the query.
		selection    *NodeSelection
		data         any
		relationship *rsPatternPart
	}
	rsPatternPart struct {
		data    any
		to      *nodePatternPart
		from    *nodePatternPart
		related *nodePatternPart
	}
)

func NewNode(match any) Pattern {
	return &CypherPattern{
		resolver: func(_ *Registry) *nodePatternPart {
			return &nodePatternPart{data: match}
		},
	}
}

func NewPath(path Pattern, name string) Pattern {
	return &CypherPattern{
		resolver: func(r *Registry) *nodePatternPart {
			n := path.createNodePattern(r)
			n.pathName = name
			return n
		},
	}
}

func NewQueryPattern(rootIdentifier any, query string) Pattern {
	return &CypherPattern{
		resolver: func(r *Registry) *nodePatternPart {
			nodeMatch := func(sel *NodeSelection) (node any) {
				node = sel.Alloc
				props := identifierToProps(reflect.ValueOf(sel.Payload), sel.Name)
				if sel.Name != "" {
					node = Variable{
						Identifier: node,
						Name:       sel.Name,
						Props:      props,
					}
				}
				return
			}
			rsMatch := func(sel *RelationshipSelection) (rs any) {
				rs = sel.Alloc
				props := identifierToProps(reflect.ValueOf(sel.Payload), sel.Name)
				if sel.Name != "" {
					rs = Variable{
						Identifier: rs,
						Name:       sel.Name,
						Props:      props,
					}
				}
				return
			}
			headSel, err := ResolveQuery(r, rootIdentifier, query)
			var pattern Pattern = &CypherPattern{
				resolver: func(_ *Registry) *nodePatternPart {
					return &nodePatternPart{
						data:      nodeMatch(headSel),
						selection: headSel,
					}
				},
			}
			if err != nil {
				panic(err)
			}
			nextSel := headSel
			for nextSel != nil {
				relSel := nextSel.Next
				if relSel == nil {
					break
				}
				nextSel = relSel.Next
				if relSel.Target.Dir {
					pattern = pattern.To(rsMatch(relSel), nodeMatch(nextSel))
				} else {
					pattern = pattern.From(rsMatch(relSel), nodeMatch(nextSel))
				}
			}
			p := pattern.createNodePattern(r)
			return p
		},
	}
}

func Paths(paths ...Pattern) Patterns {
	if len(paths) == 0 {
		panic(errors.New("no paths"))
	}
	return &CypherPatterns{
		resolver: func(r *Registry) []*nodePatternPart {
			ns := make([]*nodePatternPart, len(paths))
			for i, path := range paths {
				ns[i] = path.createNodePattern(r)
			}
			return ns
		},
	}
}

func (n *nodePatternPart) next() *nodePatternPart {
	if n.relationship == nil {
		return n
	}
	if n.relationship.from != nil {
		return n.relationship.from
	} else if n.relationship.to != nil {
		return n.relationship.to
	} else if n.relationship.related != nil {
		return n.relationship.related
	} else {
		panic(errors.New("relationship has no target"))
	}
}

func (n *nodePatternPart) tail() *nodePatternPart {
	tail := n
	if tail == nil {
		panic(errors.New("head is nil"))
	}
	for tail != nil && tail.relationship != nil {
		tail = tail.next()
	}

	return tail
}

func (c *CypherPattern) Related(relationshipMatch, nodeMatch any) Pattern {
	return c.setRelationship(nil, relationshipMatch, nodeMatch)
}

func (c *CypherPattern) From(relationshipMatch, nodeMatch any) Pattern {
	falseVal := false
	return c.setRelationship(&falseVal, relationshipMatch, nodeMatch)
}

func (c *CypherPattern) To(relationshipMatch, nodeMatch any) Pattern {
	trueVal := true
	return c.setRelationship(&trueVal, relationshipMatch, nodeMatch)
}

func (c *CypherPattern) setRelationship(
	dir *bool,
	relationshipMatch, nodeMatch any,
) Pattern {
	prevResolver := c.resolver
	c.resolver = func(r *Registry) *nodePatternPart {
		prev := prevResolver(r)
		rel := &rsPatternPart{data: relationshipMatch}
		next := &nodePatternPart{data: nodeMatch}
		if dir == nil {
			rel.related = next
		} else if *dir {
			rel.to = next
		} else {
			rel.from = next
		}
		prev.tail().relationship = rel
		return prev
	}
	return c
}

func (c *CypherPattern) createNodePattern(r *Registry) *nodePatternPart {
	return c.resolver(r)
}

func (c *CypherPattern) nodes(r *Registry) []*nodePatternPart {
	return []*nodePatternPart{c.resolver(r)}
}

func (c *CypherPattern) Condition() *Condition {
	return &Condition{Path: c}
}

func (c *CypherPattern) configureWhere(w *Where) {
	c.Condition().configureWhere(w)
}

func (c *CypherPatterns) nodes(r *Registry) []*nodePatternPart {
	return c.resolver(r)
}
