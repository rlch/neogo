package db

import "github.com/rlch/neogo/internal"

// Param injects param into the parameters of a query. The parameter's name will
// be based off the type of param, or can be explictly set with [NamedParam].
func Param(param any) internal.Param {
	return internal.Param{
		Value: &param,
	}
}

// NamedParam injects param into the parameters of a query with the given name.
func NamedParam(param any, name string) internal.Param {
	return internal.Param{
		Value: &param,
		Name:  name,
	}
}
