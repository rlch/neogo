package internal

import "errors"

type (
	Pattern interface {
		Patterns
		ICondition

		node() *node
		Related(edgeMatch, nodeMatch any) Pattern
		From(edgeMatch, nodeMatch any) Pattern
		To(edgeMatch, nodeMatch any) Pattern
	}

	Patterns interface {
		nodes() []*node
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
	node struct {
		pathName     string
		data         any
		relationship *relationship
	}
	relationship struct {
		data    any
		to      *node
		from    *node
		related *node
	}
)

func (n *node) next() *node {
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

func (n *node) tail() *node {
	tail := n
	if tail == nil {
		panic(errors.New("head is nil"))
	}
	for tail != nil && tail.relationship != nil {
		tail = tail.next()
	}
	return tail
}

func Node(match any) Pattern {
	return &CypherPath{n: &node{data: match}}
}

func NewPath(path Pattern, name string) Pattern {
	n := path.node()
	n.pathName = name
	return &CypherPath{n: path.node()}
}

func Paths(paths ...Pattern) Patterns {
	if len(paths) == 0 {
		panic(errors.New("no paths"))
	}
	ns := make([]*node, len(paths))
	for i, path := range paths {
		ns[i] = path.node()
	}
	return &CypherPattern{ns: ns}
}

func (c *CypherPath) Related(edgeMatch, nodeMatch any) Pattern {
	c.n.tail().relationship = &relationship{
		data:    edgeMatch,
		related: &node{data: nodeMatch},
	}
	return c
}

func (c *CypherPath) From(edgeMatch, nodeMatch any) Pattern {
	c.n.tail().relationship = &relationship{
		data: edgeMatch,
		from: &node{data: nodeMatch},
	}
	return c
}

func (c *CypherPath) To(edgeMatch, nodeMatch any) Pattern {
	c.n.tail().relationship = &relationship{
		data: edgeMatch,
		to:   &node{data: nodeMatch},
	}
	return c
}

func (c *CypherPath) node() *node {
	return c.n
}

func (c *CypherPath) nodes() []*node {
	return []*node{c.n}
}

func (c *CypherPath) Condition() *Condition {
	return &Condition{Path: c}
}

func (c *CypherPath) configureWhere(w *Where) {
	c.Condition().configureWhere(w)
}

func (c *CypherPattern) nodes() []*node {
	return c.ns
}
