package expr

import (
	"strings"

	"github.com/rlch/neogo"
	"github.com/rlch/neogo/internal"
	"github.com/rlch/neogo/query"
)

func New(expr func(scope query.Scope, b *strings.Builder)) neogo.Expression {
	return &expression{expr}
}

type expression struct {
	expr func(scope query.Scope, b *strings.Builder)
}

func (e *expression) Compile(s query.Scope, b *strings.Builder) {
	e.expr(s, b)
}

func empty() *Client {
	return &Client{buffer: internal.NewCypherClient()}
}

func newClient(cy *internal.CypherClient) *Client {
	return &Client{
		buffer:  cy,
		Reader:  newReader(cy.CypherReader),
		Updater: newUpdater(cy.CypherUpdater, newQuerier),
	}
}

func newRunner(cy *internal.CypherRunner) *runner {
	return &runner{cy}
}

func newQuerier(cy *internal.CypherQuerier) *Querier {
	return &Querier{
		buffer: cy,
		Reader: newReader(cy.CypherReader),
		Updater: newUpdater(
			cy.CypherUpdater,
			newQuerier,
		),
		runner: newRunner(cy.CypherRunner),
	}
}

func newReader(cy *internal.CypherReader) *Reader {
	return &Reader{
		buffer: cy,
		runner: newRunner(cy.CypherRunner),
	}
}

func newYielder(cy *internal.CypherYielder) *Yielder {
	return &Yielder{
		buffer:  cy,
		Querier: newQuerier(cy.CypherQuerier),
		runner:  newRunner(cy.CypherRunner),
	}
}

func newUpdater[To any, ToCypher any](
	cy *internal.CypherUpdater[ToCypher],
	to func(ToCypher) To,
) *Updater[To, ToCypher] {
	return &Updater[To, ToCypher]{
		buffer: cy,
		To:     to,
	}
}

type (
	Client struct {
		buffer *internal.CypherClient
		*Reader
		*Updater[*Querier, *internal.CypherQuerier]
	}
	runner struct{ buffer *internal.CypherRunner }
	Runner interface {
		neogo.Expression
		Print()
		getBuffer() *internal.CypherRunner
	}
	Querier struct {
		buffer *internal.CypherQuerier
		*Reader
		*Updater[*Querier, *internal.CypherQuerier]
		*runner
	}
	Reader struct {
		buffer *internal.CypherReader
		*runner
	}
	Yielder struct {
		buffer *internal.CypherYielder
		*Querier
		*runner
	}
	Updater[To any, ToCypher any] struct {
		buffer *internal.CypherUpdater[ToCypher]
		To     func(ToCypher) To
		*runner
	}
)

var (
	_ query.Expression = (*Querier)(nil)
	_ query.Expression = (*Reader)(nil)
	_ query.Expression = (*Yielder)(nil)
	_ query.Expression = (*Updater[any, any])(nil)
)

func Use(graphExpr string) *Querier {
	e := empty()
	q := e.buffer.Use(graphExpr)
	return newQuerier(q)
}

func (e *Client) Use(graphExpr string) *Querier {
	q := e.buffer.Use(graphExpr)
	return newQuerier(q)
}

func Union(unions ...func(*Client) Runner) *Querier {
	e := empty()
	return e.Union(unions...)
}

func (e *Client) Union(unions ...func(*Client) Runner) *Querier {
	in := make([]func(c *internal.CypherClient) *internal.CypherRunner, len(unions))
	for i, u := range unions {
		union := u
		in[i] = func(c *internal.CypherClient) *internal.CypherRunner {
			return union(newClient(c)).getBuffer()
		}
	}
	q := e.buffer.Union(in...)
	return newQuerier(q)
}

func UnionAll(unions ...func(*Client) Runner) *Querier {
	e := empty()
	return e.UnionAll(unions...)
}

func (e *Client) UnionAll(unions ...func(*Client) Runner) *Querier {
	in := make([]func(c *internal.CypherClient) *internal.CypherRunner, len(unions))
	for i, u := range unions {
		union := u
		in[i] = func(c *internal.CypherClient) *internal.CypherRunner {
			return union(newClient(c)).getBuffer()
		}
	}
	q := e.buffer.UnionAll(in...)
	return newQuerier(q)
}

func OptionalMatch(pattern internal.Patterns) *Querier {
	e := empty()
	q := e.buffer.OptionalMatch(pattern)
	return newQuerier(q)
}

func (e *Reader) OptionalMatch(pattern internal.Patterns) *Querier {
	q := e.buffer.OptionalMatch(pattern)
	return newQuerier(q)
}

func Match(pattern internal.Patterns) *Querier {
	e := empty()
	q := e.buffer.Match(pattern)
	return newQuerier(q)
}

func (e *Reader) Match(pattern internal.Patterns) *Querier {
	q := e.buffer.OptionalMatch(pattern)
	return newQuerier(q)
}

func Return(identifiers ...query.Identifier) *runner {
	e := empty()
	q := e.buffer.Return(identifiers...)
	return newRunner(q)
}

func (e *Reader) Return(identifiers ...query.Identifier) *runner {
	q := e.buffer.Return(identifiers...)
	return newRunner(q)
}

func With(identifiers ...query.Identifier) *Querier {
	e := empty()
	q := e.buffer.With(identifiers...)
	return newQuerier(q)
}

func (e *Reader) With(identifiers ...query.Identifier) *Querier {
	q := e.buffer.With(identifiers...)
	return newQuerier(q)
}

func Call(procedure string) *Yielder {
	e := empty()
	q := e.buffer.Call(procedure)
	return newYielder(q)
}

func (e *Reader) Call(procedure string) *Yielder {
	q := e.buffer.Call(procedure)
	return newYielder(q)
}

func Show(command string) *Yielder {
	e := empty()
	q := e.buffer.Show(command)
	return newYielder(q)
}

func (e *Reader) Show(command string) *Yielder {
	q := e.buffer.Show(command)
	return newYielder(q)
}

func Subquery(subquery func(c *Client) Runner) *Querier {
	e := empty()
	inSubquery := func(cc *internal.CypherClient) *internal.CypherRunner {
		runner := subquery(newClient(cc))
		return runner.getBuffer()
	}
	q := e.buffer.Subquery(inSubquery)
	return newQuerier(q)
}

func (e *Reader) Subquery(subquery func(c *Client) Runner) *Querier {
	inSubquery := func(cc *internal.CypherClient) *internal.CypherRunner {
		runner := subquery(newClient(cc))
		return runner.getBuffer()
	}
	q := e.buffer.Subquery(inSubquery)
	return newQuerier(q)
}

func Cypher(query string) *Querier {
	e := empty()
	q := e.buffer.Cypher(query)
	return newQuerier(q)
}

func (e *Reader) Cypher(query string) *Querier {
	q := e.buffer.Cypher(query)
	return newQuerier(q)
}

func Unwind(identifier query.Identifier, as string) *Querier {
	e := empty()
	q := e.buffer.Unwind(identifier, as)
	return newQuerier(q)
}

func (e *Reader) Unwind(identifier query.Identifier, as string) *Querier {
	q := e.buffer.Unwind(identifier, as)
	return newQuerier(q)
}

func Yield(identifiers ...query.Identifier) *Querier {
	e := empty().buffer.Call("")
	e.Reset()
	q := e.Yield(identifiers...)
	return newQuerier(q)
}

func (e *Yielder) Yield(identifiers ...query.Identifier) *Querier {
	q := e.buffer.Yield(identifiers...)
	return newQuerier(q)
}

func Where(opts ...internal.WhereOption) *Querier {
	e := empty().buffer.With("")
	e.Reset()
	q := e.Where(opts...)
	return newQuerier(q)
}

func (e *Querier) Where(opts ...internal.WhereOption) *Querier {
	q := e.buffer.Where(opts...)
	return newQuerier(q)
}

func Create(pattern internal.Patterns) *Querier {
	e := empty()
	q := e.buffer.Create(pattern)
	return newQuerier(q)
}

func (e *Updater[To, CypherTo]) Create(pattern internal.Patterns) To {
	q := e.buffer.Create(pattern)
	return e.To(q)
}

func Merge(pattern internal.Pattern, opts ...internal.MergeOption) *Querier {
	e := empty()
	q := e.buffer.Merge(pattern, opts...)
	return newQuerier(q)
}

func (e *Updater[To, CypherTo]) Merge(pattern internal.Pattern, opts ...internal.MergeOption) To {
	q := e.buffer.Merge(pattern, opts...)
	return e.To(q)
}

func Delete(identifiers ...query.PropertyIdentifier) *Querier {
	e := empty()
	q := e.buffer.Delete(identifiers...)
	return newQuerier(q)
}

func (e *Updater[To, CypherTo]) Delete(identifiers ...query.Identifier) To {
	q := e.buffer.Delete(identifiers...)
	return e.To(q)
}

func DetachDelete(identifiers ...query.PropertyIdentifier) *Querier {
	e := empty()
	q := e.buffer.DetachDelete(identifiers...)
	return newQuerier(q)
}

func (e *Updater[To, CypherTo]) DetachDelete(identifiers ...query.PropertyIdentifier) To {
	q := e.buffer.DetachDelete(identifiers...)
	return e.To(q)
}

func Set(items ...internal.SetItem) *Querier {
	e := empty()
	q := e.buffer.Set(items...)
	return newQuerier(q)
}

func (e *Updater[To, CypherTo]) Set(items ...internal.SetItem) To {
	q := e.buffer.Set(items...)
	return e.To(q)
}

func Remove(items ...internal.RemoveItem) *Querier {
	e := empty()
	q := e.buffer.Remove(items...)
	return newQuerier(q)
}

func (e *Updater[To, CypherTo]) Remove(items ...internal.RemoveItem) To {
	q := e.buffer.Remove(items...)
	return e.To(q)
}

func ForEach(identifier query.Identifier, inValue query.ValueIdentifier, do func(c *Updater[any, any])) *Querier {
	e := empty()
	q := e.buffer.ForEach(identifier, inValue, func(c *internal.CypherUpdater[any]) {
		u := newUpdater(c, func(tc any) any { return nil })
		do(u)
	})
	return newQuerier(q)
}

func (e *Updater[To, CypherTo]) ForEach(identifier query.Identifier, inValue query.ValueIdentifier, do func(c *Updater[any, any])) To {
	q := e.buffer.ForEach(identifier, inValue, func(c *internal.CypherUpdater[any]) {
		u := newUpdater(c, func(tc any) any { return nil })
		do(u)
	})

	return e.To(q)
}

func (c *runner) Compile(s query.Scope, b *strings.Builder) {
	cy, err := c.buffer.Compile()
	scope := s.(*internal.Scope)
	scope.MergeChildScope(c.buffer.Scope)
	scope.AddError(err)
	b.WriteString(cy.Cypher)
}

func (c *runner) Print() {
	c.buffer.Print()
}

func (c *runner) getBuffer() *internal.CypherRunner {
	return c.buffer
}
