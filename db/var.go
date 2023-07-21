package db

import (
	"fmt"

	"github.com/rlch/neo4j-gorm/internal"
)

func Var(entity any, opts ...internal.VariableOption) *internal.Variable {
	v := &internal.Variable{}
	for _, opt := range opts {
		opt.ConfigureVariable(v)
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
		panic(fmt.Sprintf(
			`Cannot create variable from 2 expressions: Qual(%s, ...) = %+v)`, entity, v,
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

func Pattern(pattern internal.Expr) internal.VariableOption {
	return &internal.Configurer{
		Variable: func(v *internal.Variable) {
			v.Pattern = pattern
		},
	}
}

type Props map[any]internal.Expr

func (p Props) ConfigureVariable(v *internal.Variable) {
	v.Props = p
}
