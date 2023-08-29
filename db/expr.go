package db

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/rlch/neogo/internal"
)

// Expr returns a Cypher literal [expression].
//
// [expression]: https://neo4j.com/docs/cypher-manual/current/syntax/expressions/
func Expr(expr string) internal.Expr {
	return internal.Expr(expr)
}

// String returns a Cypher [string literal expression], wrapped in double-quotes.
// This is a convenience function for:
//
//	Expr(strconv.Quote(s))
//
// [string literal expression]: https://neo4j.com/docs/cypher-manual/current/syntax/expressions/#cypher-expressions-string-literals
func String(s string) internal.Expr {
	return internal.Expr(strconv.Quote(s))
}

func Map(m map[string]internal.Expr) internal.Expr {
	var b strings.Builder
	b.WriteRune('{')
	var i int
	n := len(m)
	for k, v := range m {
		fmt.Fprintf(&b, "%s: %s", k, v)
		i++
		if i < n {
			b.WriteRune(',')
		}
		if n > 1 {
			b.WriteString("\n" + internal.Indent)
		}
	}
	b.WriteRune('}')
	return internal.Expr(b.String())
}
