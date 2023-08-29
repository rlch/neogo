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

func (c *reconcilerClient) Use(scope client.Scope, graphExpr string) {
	c.Reconcile(scope, clauseUse, graphExpr)
}

func (c *reconcilerClient) OptionalMatch(scope client.Scope, patterns internal.Patterns) {
	c.Reconcile(scope, clauseOptionalMatch, patterns)
}

func (c *reconcilerClient) Match(scope client.Scope, patterns internal.Patterns) {
	c.Reconcile(scope, clauseMatch, patterns)
}

func (c *reconcilerClient) With(scope client.Scope, identifiers ...client.Identifier) {
	c.Reconcile(scope, clauseWith, toAnyList(identifiers...)...)
}

func (c *reconcilerClient) Unwind(scope client.Scope, expr client.Identifier, as string) {
	c.Reconcile(scope, clauseUnwind, expr, as)
}

func (c *reconcilerClient) Call(scope client.Scope, procedure string) {
	c.Reconcile(scope, clauseCall, procedure)
}

func (c *reconcilerClient) Show(scope client.Scope, command string) {
	c.Reconcile(scope, clauseShow, command)
}

func (c *reconcilerClient) Where(scope client.Scope, where *internal.Where) {
	c.Reconcile(scope, clauseWhere, where)
}

func (c *reconcilerClient) Yield(scope client.Scope, identifiers ...client.Identifier) {
	c.Reconcile(scope, clauseYield, toAnyList(identifiers...)...)
}

func (c *reconcilerClient) Return(scope client.Scope, identifiers ...client.Identifier) {
	c.Reconcile(scope, clauseReturn, toAnyList(identifiers...)...)
}

func (c *reconcilerClient) Create(scope client.Scope, pattern internal.Patterns) {
	c.Reconcile(scope, clauseCreate, pattern)
}

func (c *reconcilerClient) Merge(scope client.Scope, pattern internal.Pattern, opts *internal.Merge) {
	c.Reconcile(scope, clauseMerge, pattern, opts)
}

func (c *reconcilerClient) DetachDelete(scope client.Scope, identifiers ...client.Identifier) {
	c.Reconcile(scope, clauseDetachDelete, toAnyList(identifiers...)...)
}

func (c *reconcilerClient) Delete(scope client.Scope, identifiers ...client.Identifier) {
	c.Reconcile(scope, clauseDelete, toAnyList(identifiers...)...)
}

func (c *reconcilerClient) Set(scope client.Scope, items ...internal.SetItem) {
	c.Reconcile(scope, clauseSet, toAnyList(items...)...)
}

func (c *reconcilerClient) Remove(scope client.Scope, items ...*internal.RemoveItem) {
	c.Reconcile(scope, clauseRemove, toAnyList(items...)...)
}
