// Package builder provides client interfaces for constructing and executing Cypher queries.
package builder

import (
	"context"
	"strings"

	"github.com/neo4j/neo4j-go-driver/v5/neo4j"
	"github.com/rlch/neogo/internal"
)

type (
	// Identifier is an important concept in neogo. It represents a reference to
	// some entity in the database (node, relationship, property, variables, etc).
	//
	// It can be:
	//   - nil for referencing nothing
	//   - string or [pkg/github.com/rlch/neogo/db.Expr] for referencing an entity by name or expression
	//   - pointer to a zero value
	//   - non-string, non-zero value which will be injected as a parameter
	//   - pointer to a field of a registered struct identifier
	//
	// If the identifier has been registered in the current state of the query, the
	// corresponding name will be injected into the query.
	//
	// Depending on the context in which the identifier is used, the behaviour of
	// certain types can change. See [PropertyIdentifier] and [ValueIdentifier].
	Identifier = any

	// PropertyIdentifier is a type of [Identifier], which considers strings as
	// property accessors as opposed to literals, when used in a pattern WHERE
	// clause or ORDER BY clause. Literals may still be used by wrapping then in
	// [pkg/github.com/rlch/neogo/db.Expr]
	//
	// It is important to note that [PropertyIdentifier]'s cannot register
	// identifiers, only refer to existing ones.
	PropertyIdentifier = Identifier

	// ValueIdentifier is a type of [Identifier] and a super-type of
	// [PropertyIdentifier]. It handles strings in the
	//
	// It is important to note that ValueIdentifiers cannot register
	// identifiers, only refer to existing ones.
	ValueIdentifier = Identifier
)

type (
	// Scope provides information about the current state of the query.
	Scope interface {
		// Name returns the name of previously registered identifier.
		Name(identifier Identifier) string
		// Error returns the error that occurred during the query.
		Error() error
		// AddError adds an error to the query.
		AddError(err error)
	}

	Expression interface {
		Compile(s Scope, b *strings.Builder)
	}
)

// Builder is the interface for constructing a Cypher query.
//
// It can be instantiated using the [pkg/github.com/rlch/neogo.New] function.
type Builder interface {
	Reader
	Updater[Querier]

	// Use writes a USE clause to the query, specifying the graph to be used.
	//
	//  USE <graphExpr>
	Use(graphExpr string) Querier

	// Union writes a UNION clause to the query, combining the results of each
	// subquery.
	//
	//  <query>
	//  UNION
	//  <query>
	//  ...
	Union(unions ...func(c Builder) Runner) Querier

	// Union writes a UNION ALL clause to the query, combining the results of each
	// subquery.
	//
	//  <query>
	//  UNION ALL
	//  <query>
	//  ...
	UnionAll(unions ...func(c Builder) Runner) Querier
}

// Reader is the interface for reading data from the database.
type Reader interface {
	// OptionalMatch writes an OPTIONAL MATCH clause to the query.
	//
	//  OPTIONAL MATCH <pattern>
	OptionalMatch(pattern internal.Patterns) Querier

	// Match writes a MATCH clause to the query.
	//
	//  MATCH <pattern>
	Match(pattern internal.Patterns) Querier

	// Return writes a RETURN clause to the query.
	//
	//  RETURN <identifier>, ... ,<identifier>
	Return(identifiers ...Identifier) Runner

	// With writes a WITH clause to the query.
	//
	//  WITH <identifier>, ... ,<identifier>
	With(identifiers ...Identifier) Querier

	// Call writes a CALL clause to the query.
	//
	//  CALL <procedure>
	Call(procedure string) Yielder

	// Show writes a SHOW clause to the query.
	//
	//  SHOW <command>
	Show(command string) Yielder

	Subquery(func(c Builder) Runner) Querier

	// Cypher allows you to inject a raw Cypher query into the query.
	Cypher(query string) Querier

	// Eval allows you to inject an expression into the query.
	//
	// The expression is passed a Scope, which can be used to obtain the information
	// about the querys current state.
	Eval(expression Expression) Querier

	// Unwind writes an UNWIND clause to the query.
	//
	// as is the name of the variable to which the list elements will be bound.
	//
	//  UNWIND <identifier> AS <as>
	Unwind(identifier Identifier, as string) Querier
}

// Yielder is the interface for yielding or reading data from the database.
type Yielder interface {
	Querier

	// Yield writes a YIELD clause to the query.
	//
	//  YIELD <identifier>, ... ,<identifier>
	Yield(identifiers ...Identifier) Querier
}

// Querier is the interface for constructing a Cypher query.
type Querier interface {
	Reader
	Runner
	Updater[Querier]

	// Where writes a WHERE clause to the query.
	Where(args ...any) Querier
}

// Updater is the interface for updating data in the database.
type Updater[To any] interface {
	// Create writes a CREATE clause to the query.
	//
	//  CREATE <pattern>
	Create(patterns internal.Patterns) To

	// Merge writes a MERGE clause to the query.
	//
	//  MERGE <pattern>
	Merge(pattern internal.Pattern, opts ...internal.MergeOption) To

	// Delete writes a DELETE clause to the query.
	//
	//  DELETE <identifier>, ... ,<identifier>
	Delete(identifiers ...PropertyIdentifier) To

	// DetachDelete writes a DETACH DELETE clause to the query.
	//
	//  DETACH DELETE <identifier>, ... ,<identifier>
	DetachDelete(identifiers ...PropertyIdentifier) To

	// Set writes a SET clause to the query
	Set(items ...internal.SetItem) To

	// Remove writes a REMOVE clause to the query
	Remove(items ...internal.RemoveItem) To

	// Foreach writes a FOREACH clause to the query
	//
	// The subquery will contain the identifier in its scope.
	//
	// FOREACH (<identifier> IN <valueIdentifier> | <query>)
	ForEach(identifier Identifier, inValue ValueIdentifier, do func(c Updater[any])) To
}

// Runner allows the query to be executed.
type Runner interface {
	Print() Runner
	DebugPrint() Runner

	// Run executes the query, populating all the values bound within the query if
	// their identifiers exist in the returning scope.
	Run(ctx context.Context) error

	// RunWithParams is the same as Run, but injects the provided parameters into the
	// query.
	RunWithParams(ctx context.Context, params map[string]any) error

	// RunSummary is the same as Run, and returns a summary of the result.
	RunSummary(ctx context.Context) (ResultSummary, error)

	// RunSummaryWithParams is the same as RunWithParams, and returns a summary of the result.
	RunSummaryWithParams(ctx context.Context, params map[string]any) (ResultSummary, error)

	// Stream executes the query and returns an abstraction over a
	// [pkg/github.com/neo4j/neo4j-go-driver/v5/neo4j.ResultWithContext], which
	// allows records to be consumed one-by-one as a linked list, instead of all
	// at once like Run. This is useful for large or undefined results that may
	// not necessarily fit in memory.
	Stream(ctx context.Context, sink func(r Result) error) error

	// StreamWithParams is the same as Stream, but injects the provided parameters
	StreamWithParams(ctx context.Context, params map[string]any, sink func(r Result) error) error
}

type (
	Result interface {
		// Peek returns true only if there is a record after the current one to be processed without advancing the record
		// stream
		Peek(ctx context.Context) bool

		// Next returns true only if there is a record to be processed.
		Next(ctx context.Context) bool

		// Err returns the latest error that caused this Next to return false.
		Err() error

		// Read reads the values of the current record into the values bound within
		// the query.
		Read() error
	}
	ResultSummary = neo4j.ResultSummary
)
