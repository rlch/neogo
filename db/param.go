package db

import "github.com/rlch/neogo/internal"

// Param injects param into the [parameters] of a query. The parameter's name will
// be based off the type of param, or can be explicitly set with [NamedParam].
//
// [parameters]: https://neo4j.com/docs/cypher-manual/current/syntax/parameters/
func Param(param any) internal.Param {
	return internal.Param{
		Value: &param,
	}
}

// NamedParam injects param into the [parameters] of a query, qualified by name.
//
// [parameters]: https://neo4j.com/docs/cypher-manual/current/syntax/parameters/
func NamedParam(param any, name string) internal.Param {
	return internal.Param{
		Value: &param,
		Name:  name,
	}
}
