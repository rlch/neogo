package db

import (
	"strconv"

	"github.com/rlch/neogo/internal"
)

// Expr returns a Cypher literal [expression].
//
// [expression]: https://neo4j.com/docs/cypher-manual/current/syntax/expressions/
func Expr(expr string, args ...any) internal.Expr {
	return internal.Expr{
		Value: expr,
		Args:  args,
	}
}

// String returns a Cypher [string literal expression], wrapped in double-quotes.
// This is a convenience function for:
//
//	Expr(strconv.Quote(s))
//
// [string literal expression]: https://neo4j.com/docs/cypher-manual/current/syntax/expressions/#cypher-expressions-string-literals
func String(s string) string {
	return strconv.Quote(s)
}
