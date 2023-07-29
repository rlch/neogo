package db

import (
	"fmt"

	"github.com/goccy/go-json"

	"github.com/rlch/neogo/internal"
)

func Var(entity any, opts ...internal.VariableOption) *internal.Variable {
	v := &internal.Variable{}
	for _, opt := range opts {
		internal.ConfigureVariable(v, opt)
	}
	switch e := entity.(type) {
	case internal.Expr:
		v.Expr = e
	case string:
		v.Expr = Expr(e)
	default:
		v.Entity = e
	}
	return v
}

// If entity is a string, it becomes the expression of the variable and expr
// becomes the alias. If a name is also provided, we throw.
func Qual(entity any, expr string, opts ...internal.VariableOption) *internal.Variable {
	// Check if name is provided in opts, if so we make it an alias.
	v := Var(entity, opts...)
	if v.Name != "" && v.Expr != "" {
		panic(fmt.Errorf(
			`cannot create variable from 2 expressions: Qual(%s, ...) = %+v)`, entity, v,
		))
	}
	// entity > expr > name
	if v.Expr != "" {
		v.Name = expr
	} else {
		v.Expr = Expr(expr)
	}
	return v
}

func Bind(entity any, toPtr any) *internal.Variable {
	return &internal.Variable{
		Entity: entity,
		Bind:   toPtr,
	}
}

func Name(name string) internal.VariableOption {
	return &internal.Configurer{
		Variable: func(v *internal.Variable) {
			v.Name = name
		},
	}
}

func Label(pattern internal.Expr) internal.VariableOption {
	return &internal.Configurer{
		Variable: func(v *internal.Variable) {
			v.Pattern = pattern
		},
	}
}

func Quantifier(quantifier internal.Expr) internal.VariableOption {
	return &internal.Configurer{
		Variable: func(v *internal.Variable) {
			v.Quantifier = quantifier
		},
	}
}

func Select(filter *json.FieldQuery) internal.VariableOption {
	return &internal.Configurer{
		Variable: func(v *internal.Variable) {
			v.Select = filter
		},
	}
}

type Props = internal.Props
