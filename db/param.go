package db

import "github.com/rlch/neogo/internal"

func Param(param any) internal.Param {
	return internal.Param{
		Value: &param,
	}
}

func NamedParam(name string, param any) internal.Param {
	return internal.Param{
		Name:  name,
		Value: &param,
	}
}
