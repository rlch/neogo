package neogo

import (
	"context"

	_ "github.com/neo4j/neo4j-go-driver/v5/neo4j"
	_ "github.com/sanity-io/litter"

	"github.com/rlch/neogo/internal"
)

// Client is the interface for 
type Client interface {
	Reader
	Updater[Querier]
	Use(graphExpr string) Querier
	Union(unions ...func(c Client) runner) Querier
	UnionAll(unions ...func(c Client) runner) Querier
}

type Scope interface {
	Name(variable any) string
}

type Reader interface {
	OptionalMatch(pattern internal.Patterns) Querier
	Match(pattern internal.Patterns) Querier
	Return(matches ...any) runner

	// The WITH clause allows query parts to be chained together, piping the
	// results from one to be used as starting points or criteria in the next.
	With(variables ...any) Querier

	Call(procedure string) yielder

	Subquery(func(c Client) runner) Querier

	// Cypher allows you to inject a raw Cypher query into the query.
	// The function is passed a Scope, which can be used to obtain the information
	// about the querys current state.
	Cypher(query func(scope Scope) string) Querier

	// Unwind expands a list into a sequence of rows.
	//
	// expr can be an expression/string or a slice/array.
	// If expr is a slice/array, it will be injected into the query as a
	// parameter. Otherwise, it will be injected as a literal.
	//
	// as is the name of the variable to which the list elements will be bound. If
	// the expr is addressable, this name will be bound and can be reused if
	// variable remains within the scope of the query.
	Unwind(expr any, as string) Querier
}

type yielder interface {
	Querier
	Yield(variables ...any) Querier
}

type Querier interface {
	Reader
	runner
	Updater[Querier]

	Where(opts ...internal.WhereOption) Querier
}

type Updater[To any] interface {
	Create(patterns internal.Patterns) To
	Merge(pattern internal.Pattern, opts ...internal.MergeOption) To
	Delete(variables ...any) To
	Set(items ...internal.SetItem) To
	Remove(items ...internal.RemoveItem) To
	ForEach(entity, inList any, do func(c Updater[any])) To
}

type runner interface {
	Run(ctx context.Context) error
}
