package internal

import "errors"

type (
	Pattern interface {
		Patterns
		ICondition

		nodePattern() *NodePattern
		Related(relationship, node any) Pattern
		From(relationship, node any) Pattern
		To(relationship, node any) Pattern
	}

	Patterns interface {
		nodes() []*NodePattern
	}
)

var (
	_ interface {
		Pattern
		// A path can be used in a WHERE condition
		WhereOption
	} = (*CypherPath)(nil)
	_ Patterns = (*CypherPattern)(nil)
)

type (
	CypherPath struct {
		Pattern *NodePattern
	}
	CypherPattern struct {
		Patterns []*NodePattern
	}

	NodePattern struct {
		pathName     string
		Identifier   any
		Relationship *RelationshipPattern
	}
	RelationshipPattern struct {
		Identifier any
		To         *NodePattern
		From       *NodePattern
		Related    *NodePattern
	}
)

func PatternHead(p Pattern) *NodePattern {
	return p.nodes()[0]
}

func PatternsHeads(ps Patterns) []*NodePattern {
	return ps.nodes()
}

func (n *NodePattern) Next() *NodePattern {
	if n.Relationship == nil {
		return n
	}
	if n.Relationship.From != nil {
		return n.Relationship.From
	} else if n.Relationship.To != nil {
		return n.Relationship.To
	} else if n.Relationship.Related != nil {
		return n.Relationship.Related
	} else {
		panic(errors.New("edge has no target"))
	}
}

func (n *NodePattern) Tail() *NodePattern {
	tail := n
	if tail == nil {
		panic(errors.New("head is nil"))
	}
	for tail != nil && tail.Relationship != nil {
		tail = tail.Next()
	}
	return tail
}

func NewNode(match any) Pattern {
	return &CypherPath{Pattern: &NodePattern{Identifier: match}}
}

func NewPath(path Pattern, name string) Pattern {
	n := path.nodePattern()
	n.pathName = name
	return &CypherPath{Pattern: path.nodePattern()}
}

func Paths(paths ...Pattern) Patterns {
	if len(paths) == 0 {
		panic(errors.New("no paths"))
	}
	ns := make([]*NodePattern, len(paths))
	for i, path := range paths {
		ns[i] = path.nodePattern()
	}
	return &CypherPattern{Patterns: ns}
}

func (c *CypherPath) Related(edgeMatch, nodeMatch any) Pattern {
	c.Pattern.Tail().Relationship = &RelationshipPattern{
		Identifier: edgeMatch,
		Related:    &NodePattern{Identifier: nodeMatch},
	}
	return c
}

func (c *CypherPath) From(edgeMatch, nodeMatch any) Pattern {
	c.Pattern.Tail().Relationship = &RelationshipPattern{
		Identifier: edgeMatch,
		From:       &NodePattern{Identifier: nodeMatch},
	}
	return c
}

func (c *CypherPath) To(edgeMatch, nodeMatch any) Pattern {
	c.Pattern.Tail().Relationship = &RelationshipPattern{
		Identifier: edgeMatch,
		To:         &NodePattern{Identifier: nodeMatch},
	}
	return c
}

func (c *CypherPath) nodePattern() *NodePattern {
	return c.Pattern
}

func (c *CypherPath) nodes() []*NodePattern {
	return []*NodePattern{c.Pattern}
}

func (c *CypherPath) condition() *Condition {
	return &Condition{Path: c}
}

func (c *CypherPath) configureWhere(w *Where) {
	c.condition().configureWhere(w)
}

func (c *CypherPattern) nodes() []*NodePattern {
	return c.Patterns
}
