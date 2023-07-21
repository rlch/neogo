package db

import (
	"fmt"

	"github.com/goccy/go-json"

	"github.com/rlch/neogo/client"
	"github.com/rlch/neogo/internal"
)

// Var creates a [variable] from an identifier.
//
// [variable]: https://neo4j.com/docs/cypher-manual/current/syntax/variables/
func Var(identifier client.Identifier, opts ...internal.VariableOption) *internal.Variable {
	v := &internal.Variable{}
	for _, opt := range opts {
		internal.ConfigureVariable(v, opt)
	}
	switch e := identifier.(type) {
	case internal.Expr:
		v.Expr = e
	case string:
		v.Expr = Expr(e)
	default:
		v.Identifier = e
	}
	return v
}

// Qual qualifies a [variable] with a name/expression. It can be used when:
// - A [variable] is created from an identifier and we want to give it a name
// - A projection item is created (i.e. in a [WITH] or [RETURN] clause)
//
//	Qual(Person{}, "p") -> (p:Person)
//
// If identifier is already registered, is a string or [Expr], it becomes the expression of the [variable] and expr
// becomes the alias. If a name is also provided with [Name], we throw.
//
//	<identifier> AS <expr>
//
// [variable]: https://neo4j.com/docs/cypher-manual/current/syntax/variables/
// [WITH]: https://neo4j.com/docs/cypher-manual/current/clauses/with/
// [RETURN]: https://neo4j.com/docs/cypher-manual/current/clauses/return/
func Qual(identifier client.Identifier, expr string, opts ...internal.VariableOption) *internal.Variable {
	// Check if name is provided in opts, if so we make it an alias.
	v := Var(identifier, opts...)
	if v.Name != "" && v.Expr != "" {
		panic(fmt.Errorf(
			`cannot create variable from 2 expressions: Qual(%s, ...) = %+v)`, identifier, v,
		))
	}
	// identifier > expr > name
	if v.Expr != "" {
		v.Name = expr
	} else {
		v.Expr = Expr(expr)
	}
	return v
}

// Bind binds an existing identifier to a pointer.
// When referring to that [variable], the original identifier can no longer be
// used and is replaced by toPtr.
//
// [variable]: https://neo4j.com/docs/cypher-manual/current/syntax/variables/
func Bind(identifier client.Identifier, toPtr any) *internal.Variable {
	return &internal.Variable{
		Identifier: identifier,
		Bind:       toPtr,
	}
}

// Name qualifies a [variable] with a name.
//
// [variable]: https://neo4j.com/docs/cypher-manual/current/syntax/variables/
func Name(name string) internal.VariableOption {
	return &internal.Configurer{
		Variable: func(v *internal.Variable) {
			v.Name = name
		},
	}
}

// Label sets the [label expression] of a node or relationship.
//
// [label expression]: https://neo4j.com/docs/cypher-manual/current/syntax/expressions/#label-expressions
func Label(pattern internal.Expr) internal.VariableOption {
	return &internal.Configurer{
		Variable: func(v *internal.Variable) {
			v.Pattern = pattern
		},
	}
}

// VarLength sets the [variable-length expression] of a relationship.
//
// [variable-length expression]: https://neo4j.com/docs/cypher-manual/current/patterns/reference/#variable-length-relationships
func VarLength(varLengthExpr internal.Expr) internal.VariableOption {
	return &internal.Configurer{
		Variable: func(v *internal.Variable) {
			v.VarLength = varLengthExpr
		},
	}
}

// Select filters the properties of a node or relationship to be returned.
//
// NOTE: This API has not been implemented yet.
func Select(filter *json.FieldQuery) internal.VariableOption {
	return &internal.Configurer{
		Variable: func(v *internal.Variable) {
			v.Select = filter
		},
	}
}

// Props sets the properties of a node or relationship.
// - Keys behave as [pkg/github.com/rlch/neogo/client.PropertyIdentifier]'s
// - Values behave as [pkg/github.com/rlch/neogo/client.ValueIdentifier]'s
type Props = internal.Props
