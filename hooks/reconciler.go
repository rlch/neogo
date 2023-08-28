package hooks

import (
	"github.com/rlch/neogo/client"
	"github.com/rlch/neogo/internal"
)

type reconcilerClient struct {
	*Registry
}

func toAnyList[T any](ms ...T) []any {
	as := make([]any, len(ms))
	for i, m := range ms {
		as[i] = m
	}
	return as
}

func (c *reconcilerClient) Use(graphExpr string) {
	c.Reconcile(c.scope, clauseUse, graphExpr)
}

func (c *reconcilerClient) OptionalMatch(patterns internal.Patterns) {
	c.Reconcile(clauseOptionalMatch, patterns)
}

func (c *reconcilerClient) Match(patterns internal.Patterns) {
	c.Reconcile(clauseMatch, patterns)
}

func (c *reconcilerClient) With(identifiers ...client.Identifier) {
	c.Reconcile(clauseWith, toAnyList(identifiers...)...)
}

func (c *reconcilerClient) Unwind(expr client.Identifier, as string) {
	c.Reconcile(clauseUnwind, expr, as)
}

func (c *reconcilerClient) Call(procedure string) {
	c.Reconcile(clauseCall, procedure)
}

func (c *reconcilerClient) Show(command string) {
	c.Reconcile(clauseShow, command)
}

func (c *reconcilerClient) Where(where *internal.Where) {
	c.Reconcile(clauseWhere, where)
}

func (c *reconcilerClient) Yield(identifiers ...client.Identifier) {
	c.Reconcile(clauseYield, toAnyList(identifiers...)...)
}

func (c *reconcilerClient) Return(identifiers ...client.Identifier) {
	c.Reconcile(clauseReturn, toAnyList(identifiers...)...)
}

func (c *reconcilerClient) Create(pattern internal.Patterns) {
	c.Reconcile(clauseCreate, pattern)
}

func (c *reconcilerClient) Merge(pattern internal.Pattern, opts *internal.Merge) {
	c.Reconcile(clauseMerge, pattern, opts)
}

func (c *reconcilerClient) DetachDelete(identifiers ...client.Identifier) {
	c.Reconcile(clauseDetachDelete, toAnyList(identifiers...)...)
}

func (c *reconcilerClient) Delete(identifiers ...client.Identifier) {
	c.Reconcile(clauseDelete, toAnyList(identifiers...)...)
}

func (c *reconcilerClient) Set(items ...internal.SetItem) {
	c.Reconcile(clauseSet, toAnyList(items...)...)
}

func (c *reconcilerClient) Remove(items ...*internal.RemoveItem) {
	c.Reconcile(clauseRemove, toAnyList(items...)...)
}
