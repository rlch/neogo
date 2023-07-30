package internal

import (
	"context"
)

type Client interface {
	reader
	updater[querier]
}

type Pattern interface {
	Patterns
	ICondition

	node() *node
	Related(edgeMatch, nodeMatch any) Pattern
	From(edgeMatch, nodeMatch any) Pattern
	To(edgeMatch, nodeMatch any) Pattern
}

type Patterns interface {
	nodes() []*node
}

type Scope interface {
	Name(variable any) string
}

type reader interface {
	Match(pattern Patterns, options ...MatchOption) querier
	Return(matches ...any) runner

	// The WITH clause allows query parts to be chained together, piping the
	// results from one to be used as starting points or criteria in the next.
	With(variables ...any) querier

	Subquery(func(c Client) runner) querier

	// Cypher allows you to inject a raw Cypher query into the query.
	// The function is passed a Scope, which can be used to obtain the information
	// about the querys current state.
	Cypher(func(scope Scope) string) querier

	// Unwind expands a list into a sequence of rows.
	//
	// expr can be an expression/string or a slice/array.
	// If expr is a slice/array, it will be injected into the query as a
	// parameter. Otherwise, it will be injected as a literal.
	//
	// as is the name of the variable to which the list elements will be bound. If
	// the expr is addressable, this name will be bound and can be reused if
	// variable remains within the scope of the query.
	Unwind(expr any, as string) querier
}

type querier interface {
	reader
	runner
	updater[querier]

	Where(opts ...WhereOption) querier
}

type updater[To any] interface {
	Create(patterns Patterns) To
	Merge(payload any) To
	Delete(variables ...any) To
	Set(items ...SetItem) To
	Remove(items ...RemoveItem) To
	ForEach(entity, inList any, do func(c updater[any])) To
}

type runner interface {
	Run(ctx context.Context) error
}
