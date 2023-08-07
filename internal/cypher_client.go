package internal

import (
	"fmt"
	"reflect"
	"strings"
)

func NewCypherClient() *CypherClient {
	cy := newCypher()
	return newCypherClient(cy)
}

func newCypherClient(cy *cypher) *CypherClient {
	return &CypherClient{
		cypher:        cy,
		CypherReader:  newCypherReader(cy, nil),
		CypherUpdater: newCypherUpdater(cy),
	}
}

type (
	CypherPath struct {
		n *nodePattern
	}
	CypherPattern struct {
		ns []*nodePattern
	}
	CypherClient struct {
		*cypher
		*CypherReader
		*CypherUpdater[*CypherQuerier]
	}
	CypherQuerier struct {
		*cypher
		*CypherReader
		*CypherRunner
		*CypherUpdater[*CypherQuerier]
	}
	CypherReader struct {
		*cypher
		Parent *Scope
	}
	CypherYielder struct {
		*cypher
		*CypherQuerier
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
		cypher:       cy,
		CypherReader: newCypherReader(cy, nil), CypherUpdater: newCypherUpdater(cy), CypherRunner: newCypherRunner(cy, false),
	}
	return q
}

func newCypherReader(cy *cypher, parent *Scope) *CypherReader {
	return &CypherReader{cypher: cy}
}

func newCypherUpdater(cy *cypher) *CypherUpdater[*CypherQuerier] {
	return &CypherUpdater[*CypherQuerier]{
		cypher: cy,
		To: func(c *cypher) *CypherQuerier {
			// We know if this is executed, the query has some update clause.
			c.isWrite = true
			return newCypherQuerier(c)
		},
	}
}

func newCypherYielder(cy *cypher) *CypherYielder {
	return &CypherYielder{
		cypher:        cy,
		CypherQuerier: newCypherQuerier(cy),
	}
}

func newCypherRunner(cy *cypher, isReturn bool) *CypherRunner {
	return &CypherRunner{cypher: cy, isReturn: isReturn}
}

func (c *CypherClient) Use(graphExpr string) *CypherQuerier {
	c.writeUseClause(graphExpr)
	return newCypherQuerier(c.cypher)
}

func (c *CypherClient) Union(unions ...func(c *CypherClient) *CypherRunner) *CypherQuerier {
	c.writeUnionClause(unions, false, c.Parent)
	q := newCypherQuerier(c.cypher)
	q.isReturn = true
	return q
}

func (c *CypherClient) UnionAll(unions ...func(c *CypherClient) *CypherRunner) *CypherQuerier {
	c.writeUnionClause(unions, true, c.Parent)
	q := newCypherQuerier(c.cypher)
	q.isReturn = true
	return q
}

func (c *CypherReader) OptionalMatch(patterns Patterns) *CypherQuerier {
	c.writeReadingClause(patterns.nodes(), true)
	return newCypherQuerier(c.cypher)
}

func (c *CypherReader) Match(patterns Patterns) *CypherQuerier {
	c.writeReadingClause(patterns.nodes(), false)
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

func (c *CypherReader) Call(procedure string) *CypherYielder {
	c.writeCallClause(procedure)
	return newCypherYielder(c.cypher)
}

func (c *CypherReader) Show(command string) *CypherYielder {
	c.writeShowClause(command)
	return newCypherYielder(c.cypher)
}

func (c *CypherReader) Return(matches ...any) *CypherRunner {
	c.writeProjectionBodyClause("RETURN", nil, matches...)
	return newCypherRunner(c.cypher, true)
}

func (c *CypherReader) Cypher(query func(scope *Scope) string) *CypherQuerier {
	q := query(c.Scope)
	c.WriteString(q + "\n")
	return newCypherQuerier(c.cypher)
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
	c.writeMergeClause(pattern.nodePattern(), opts...)
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

func (c *CypherYielder) Yield(variables ...any) *CypherQuerier {
	c.writeYieldClause(variables...)
	return newCypherQuerier(c.cypher)
}

func (c *CypherRunner) CompileWithParams(params map[string]any) (*CompiledCypher, error) {
	for k, v := range params {
		c.parameters[k] = v
	}
	return c.Compile()
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
		IsWrite:    c.isWrite,
	}
	if c.err != nil {
		return nil, c.err
	}
	return cy, nil
}

func (c *CypherRunner) Print() {
	fmt.Println(c.String())
	c.Reset()
}
