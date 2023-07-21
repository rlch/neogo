package internal

func NewCypherClient() *cypherClient {
	cy := newCypher()
	return &cypherClient{
		cypherReader:  *newCypherReader(cy),
		cypherUpdater: *newCypherUpdater(cy),
	}
}

type (
	cypherPath struct {
		n *node
	}
	cypherPattern struct {
		ns []*node
	}
	cypherClient struct {
		cypherReader
		cypherUpdater
	}
	cypherQuerier struct {
		cypherReader
		cypherRunner
		cypherUpdater
		*cypher
	}
	cypherReader struct {
		*cypher
	}
	cypherUpdater struct {
		*cypher
	}
	cypherRunner struct {
		*cypher
	}
)

func newCypherQuerier(cy *cypher) *cypherQuerier {
	q := &cypherQuerier{
		cypher:        cy,
		cypherReader:  *newCypherReader(cy),
		cypherUpdater: *newCypherUpdater(cy),
		cypherRunner:  *newCypherRunner(cy),
	}
	return q
}

func newCypherReader(cy *cypher) *cypherReader {
	return &cypherReader{cypher: cy}
}

func newCypherUpdater(cy *cypher) *cypherUpdater {
	return &cypherUpdater{cypher: cy}
}

func newCypherRunner(cy *cypher) *cypherRunner {
	return &cypherRunner{cypher: cy}
}

var (
	_ Path     = (*cypherPath)(nil)
	_ Patterns = (*cypherPattern)(nil)
)

type (
	node struct {
		data any
		edge *relationship
		MatchOptions
	}
	relationship struct {
		data    any
		to      *node
		from    *node
		related *node
	}
	// An instance of a node/relationship in the cypher query
	member struct {
		// The entity that was registered
		entity any
		// Whether the entity was added to the scope by the query that returned this
		// member.
		isNew bool
		// The name of the variable in the cypher query
		name  string
		alias string
		// The name of the property in the cypher query
		props string

		variable *Variable

		// The where clause that this member is associated with.
		where *Where

		// The projection body that this member is associated with.
		projectionBody *ProjectionBody
	}
)

func (n *node) next() *node {
	if n.edge == nil {
		return n
	}
	if n.edge.from != nil {
		return n.edge.from
	} else if n.edge.to != nil {
		return n.edge.to
	} else if n.edge.related != nil {
		return n.edge.related
	} else {
		panic("edge has no target")
	}
}

func (n *node) tail() *node {
	tail := n
	if tail == nil {
		panic("head is nil")
	}
	for tail != nil && tail.edge != nil {
		tail = tail.next()
	}
	return tail
}

func (c *cypherClient) Node(match any) Path {
	return &cypherPath{n: &node{data: match}}
}

func (c *cypherClient) Paths(paths ...Path) Patterns {
	if len(paths) == 0 {
		panic("no paths")
	}
	ns := make([]*node, len(paths))
	for i, path := range paths {
		ns[i] = path.node()
	}
	return &cypherPattern{ns: ns}
}

func (c *cypherPath) Related(edgeMatch, nodeMatch any) Path {
	c.n.tail().edge = &relationship{
		data:    edgeMatch,
		related: &node{data: nodeMatch},
	}
	return c
}

func (c *cypherPath) From(edgeMatch, nodeMatch any) Path {
	c.n.tail().edge = &relationship{
		data: edgeMatch,
		from: &node{data: nodeMatch},
	}
	return c
}

func (c *cypherPath) To(edgeMatch, nodeMatch any) Path {
	c.n.tail().edge = &relationship{
		data: edgeMatch,
		to:   &node{data: nodeMatch},
	}
	return c
}

func (c *cypherPath) node() *node {
	return c.n
}

func (c *cypherPath) nodes() []*node {
	return []*node{c.n}
}

func (c *cypherPath) Condition() *Condition {
	return &Condition{Path: c}
}

func (c *cypherPath) ConfigureWhere(w *Where) {
	c.Condition().ConfigureWhere(w)
}

func (c *cypherPattern) nodes() []*node {
	return c.ns
}
