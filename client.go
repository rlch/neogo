package neogo

import (
	"context"

	_ "github.com/neo4j/neo4j-go-driver/v5/neo4j"
	_ "github.com/sanity-io/litter"

	"github.com/rlch/neogo/internal"
)

type Client interface {
	reader
	updater[querier]
	Use(graphExpr string) querier
	Union(unions ...func(c Client) runner) querier
	UnionAll(unions ...func(c Client) runner) querier
}

type Scope interface {
	Name(variable any) string
}

type reader interface {
	OptionalMatch(pattern internal.Patterns) querier
	Match(pattern internal.Patterns) querier
	Return(matches ...any) runner

	// The WITH clause allows query parts to be chained together, piping the
	// results from one to be used as starting points or criteria in the next.
	With(variables ...any) querier

	Call(procedure string) yielder

	Subquery(func(c Client) runner) querier

	// Cypher allows you to inject a raw Cypher query into the query.
	// The function is passed a Scope, which can be used to obtain the information
	// about the querys current state.
	Cypher(query func(scope Scope) string) querier

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

type yielder interface {
	querier
	Yield(variables ...any) querier
}

type querier interface {
	reader
	runner
	updater[querier]

	Where(opts ...internal.WhereOption) querier
}

type updater[To any] interface {
	Create(patterns internal.Patterns) To
	Merge(pattern internal.Pattern, opts ...internal.MergeOption) To
	Delete(variables ...any) To
	Set(items ...internal.SetItem) To
	Remove(items ...internal.RemoveItem) To
	ForEach(entity, inList any, do func(c updater[any])) To
}

type runner interface {
	Run(ctx context.Context) error
}
