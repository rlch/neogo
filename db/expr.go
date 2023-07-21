package db

import (
	"strconv"

	"github.com/rlch/neo4j-gorm/internal"
)

func Expr(expr string) internal.Expr {
	return internal.Expr(expr)
}

func String(s string) internal.Expr {
	return internal.Expr(strconv.Quote(s))
}
