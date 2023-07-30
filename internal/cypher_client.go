package internal

import (
	"reflect"
	"strings"
)

func NewCypherClient() *CypherClient {
	cy := newCypher()
	return &CypherClient{
		CypherReader:  *newCypherReader(cy, nil),
		CypherUpdater: *newCypherUpdater(cy),
	}
}

type (
	CypherPath struct {
		n *node
	}
	CypherPattern struct {
		ns []*node
	}
	CypherClient struct {
		CypherReader
		CypherUpdater[*CypherQuerier]
	}
	CypherQuerier struct {
		CypherReader
		CypherRunner
		CypherUpdater[*CypherQuerier]
		*cypher
	}
	CypherReader struct {
		Parent *scope
		*cypher
	}
	CypherUpdater[To any] struct {
		*cypher
		To func(*cypher) To
	}
	CypherRunner struct {
		*cypher
		isReturn bool
	}
)

func newCypherQuerier(cy *cypher) *CypherQuerier {
	q := &CypherQuerier{
		cypher:        cy,
		CypherReader:  *newCypherReader(cy, nil),
		CypherUpdater: *newCypherUpdater(cy),
		CypherRunner:  *newCypherRunner(cy, false),
	}
	return q
}

func newCypherReader(cy *cypher, parent *scope) *CypherReader {
	return &CypherReader{cypher: cy}
}

func newCypherUpdater(cy *cypher) *CypherUpdater[*CypherQuerier] {
	return &CypherUpdater[*CypherQuerier]{
		cypher: cy,
		To: func(c *cypher) *CypherQuerier {
			return newCypherQuerier(c)
		},
	}
}

func newCypherRunner(cy *cypher, isReturn bool) *CypherRunner {
	return &CypherRunner{cypher: cy, isReturn: isReturn}
}

func (c *CypherReader) Match(patterns Patterns, options ...MatchOption) *CypherQuerier {
	for _, pattern := range patterns.nodes() {
		for _, option := range options {
			option.configureMatchOptions(&pattern.MatchOptions)
		}
	}
	c.writeReadingClause(patterns.nodes())
	return newCypherQuerier(c.cypher)
}

func (c *CypherReader) Subquery(subquery func(c *CypherClient) *CypherRunner) *CypherQuerier {
	c.writeSubqueryClause(subquery)
	return newCypherQuerier(c.cypher)
}

func (c *CypherReader) With(variables ...any) *CypherQuerier {
	c.writeProjectionBodyClause("WITH", c.Parent, variables...)
	return newCypherQuerier(c.cypher)
}

func (c *CypherReader) Unwind(expr any, as string) *CypherQuerier {
	c.writeUnwindClause(expr, as)
	return newCypherQuerier(c.cypher)
}

func (c *CypherReader) Return(matches ...any) *CypherRunner {
	c.writeProjectionBodyClause("RETURN", nil, matches...)
	return newCypherRunner(c.cypher, true)
}

func (c *CypherQuerier) Where(opts ...WhereOption) *CypherQuerier {
	where := &Where{}
	for _, opt := range opts {
		opt.configureWhere(where)
	}
	c.writeWhereClause(where, false)
	return newCypherQuerier(c.cypher)
}

func (c *CypherUpdater[To]) Create(pattern Patterns) To {
	c.writeCreateClause(pattern.nodes())
	return c.To(c.cypher)
}

func (c *CypherUpdater[To]) Merge(pattern Pattern, opts ...MergeOption) To {
	c.writeMergeClause(pattern.node(), opts...)
	return c.To(c.cypher)
}

func (c *CypherUpdater[To]) DetachDelete(variables ...any) To {
	c.writeDeleteClause(true, variables...)
	return c.To(c.cypher)
}

func (c *CypherUpdater[To]) Delete(variables ...any) To {
	c.writeDeleteClause(false, variables...)
	return c.To(c.cypher)
}

func (c *CypherUpdater[To]) Set(items ...SetItem) To {
	c.writeSetClause(items...)
	return c.To(c.cypher)
}

func (c *CypherUpdater[To]) Remove(items ...RemoveItem) To {
	c.writeRemoveClause(items...)
	return c.To(c.cypher)
}

func (c *CypherUpdater[To]) ForEach(entity, elementsExpr any, do func(c *CypherUpdater[any])) To {
	c.writeForEachClause(entity, elementsExpr, do)
	return c.To(c.cypher)
}

func (c *CypherRunner) Compile() (*CompiledCypher, error) {
	out := c.String()
	out = strings.TrimRight(out, "\n")
	if !c.isReturn {
		c.bindings = map[string]reflect.Value{}
	}
	cy := &CompiledCypher{
		Cypher:     out,
		Parameters: c.parameters,
		Bindings:   c.bindings,
	}
	if c.err != nil {
		return nil, c.err
	}
	return cy, nil
}
