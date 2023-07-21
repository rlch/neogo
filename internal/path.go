package internal

import "errors"

type (
	Pattern interface {
		Patterns
		ICondition

		nodePattern() *nodePattern
		Related(edgeMatch, nodeMatch any) Pattern
		From(edgeMatch, nodeMatch any) Pattern
		To(edgeMatch, nodeMatch any) Pattern
	}

	Patterns interface {
		nodes() []*nodePattern
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
	nodePattern struct {
		pathName     string
		data         any
		relationship *relationshipPattern
	}
	relationshipPattern struct {
		data    any
		to      *nodePattern
		from    *nodePattern
		related *nodePattern
	}
)

func (n *nodePattern) next() *nodePattern {
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
		panic(errors.New("edge has no target"))
	}
}

func (n *nodePattern) tail() *nodePattern {
	tail := n
	if tail == nil {
		panic(errors.New("head is nil"))
	}
	for tail != nil && tail.relationship != nil {
		tail = tail.next()
	}
	return tail
}

func NewNode(match any) Pattern {
	return &CypherPath{n: &nodePattern{data: match}}
}

func NewPath(path Pattern, name string) Pattern {
	n := path.nodePattern()
	n.pathName = name
	return &CypherPath{n: path.nodePattern()}
}

func Paths(paths ...Pattern) Patterns {
	if len(paths) == 0 {
		panic(errors.New("no paths"))
	}
	ns := make([]*nodePattern, len(paths))
	for i, path := range paths {
		ns[i] = path.nodePattern()
	}
	return &CypherPattern{ns: ns}
}

func (c *CypherPath) Related(edgeMatch, nodeMatch any) Pattern {
	c.n.tail().relationship = &relationshipPattern{
		data:    edgeMatch,
		related: &nodePattern{data: nodeMatch},
	}
	return c
}

func (c *CypherPath) From(edgeMatch, nodeMatch any) Pattern {
	c.n.tail().relationship = &relationshipPattern{
		data: edgeMatch,
		from: &nodePattern{data: nodeMatch},
	}
	return c
}

func (c *CypherPath) To(edgeMatch, nodeMatch any) Pattern {
	c.n.tail().relationship = &relationshipPattern{
		data: edgeMatch,
		to:   &nodePattern{data: nodeMatch},
	}
	return c
}

func (c *CypherPath) nodePattern() *nodePattern {
	return c.n
}

func (c *CypherPath) nodes() []*nodePattern {
	return []*nodePattern{c.n}
}

func (c *CypherPath) Condition() *Condition {
	return &Condition{Path: c}
}

func (c *CypherPath) configureWhere(w *Where) {
	c.Condition().configureWhere(w)
}

func (c *CypherPattern) nodes() []*nodePattern {
	return c.ns
}
