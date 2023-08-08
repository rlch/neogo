package db

import (
	"github.com/rlch/neogo/client"
	"github.com/rlch/neogo/internal"
)

// With adds configuration to a projection item for use in a [WITH] clause.
//
// [WITH]: https://neo4j.com/docs/cypher-manual/current/clauses/where/#usage-with-with-clause
func With(identifier client.Identifier, opts ...internal.ProjectionBodyOption) *internal.ProjectionBody {
	m := &internal.ProjectionBody{}
	m.Identifier = identifier
	for _, opt := range opts {
		internal.ConfigureProjectionBody(m, opt)
	}
	return m
}

// Return adds configuration to a projection item for use in a [RETURN] clause.
//
// [RETURN]: https://neo4j.com/docs/cypher-manual/current/clauses/return/
func Return(identifier client.Identifier, opts ...internal.ProjectionBodyOption) *internal.ProjectionBody {
	return With(identifier, opts...)
}

// OrderBy adds an [ORDER BY] clause to a [With] or [Return] projection item.
// asc determines whether the ordering is ascending or descending.
//
//	ORDER BY <identifier> [ASC|DESC]
//
// [ORDER BY]: https://neo4j.com/docs/cypher-manual/current/clauses/order-by/
func OrderBy(identifier client.PropertyIdentifier, asc bool) internal.ProjectionBodyOption {
	return &internal.Configurer{
		ProjectionBody: func(m *internal.ProjectionBody) {
			if m.OrderBy == nil {
				m.OrderBy = map[any]bool{}
			}
			m.OrderBy[identifier] = asc
		},
	}
}

// Skip adds a [SKIP] clause to a [With] or [Return] projection item.
//
//	SKIP <expr>
//
// [SKIP]: https://neo4j.com/docs/cypher-manual/current/clauses/skip/
func Skip(expr string) internal.ProjectionBodyOption {
	return &internal.Configurer{
		ProjectionBody: func(m *internal.ProjectionBody) {
			m.Skip = Expr(expr)
		},
	}
}

// Limit adds a [LIMIT] clause to a [With] or [Return] projection item.
//
//	LIMIT <expr>
//
// [LIMIT]: https://neo4j.com/docs/cypher-manual/current/clauses/limit/
func Limit(expr string) internal.ProjectionBodyOption {
	return &internal.Configurer{
		ProjectionBody: func(m *internal.ProjectionBody) {
			m.Limit = Expr(expr)
		},
	}
}

// Distinct adds a [DISTINCT] clause to a [With] or [Return] projection item.
//
//	<clause> DISTINCT ...
//
// [DISTINCT]: https://neo4j.com/docs/cypher-manual/current/clauses/return/#query-return-distinct
var Distinct internal.ProjectionBodyOption = &internal.Configurer{
	ProjectionBody: func(m *internal.ProjectionBody) {
		m.Distinct = true
	},
}
