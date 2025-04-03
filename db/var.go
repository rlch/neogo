package db

import (
	"fmt"

	"github.com/rlch/neogo/internal"
	"github.com/rlch/neogo/query"
)

// Var creates a [variable] from an identifier.
//
// [variable]: https://neo4j.com/docs/cypher-manual/current/syntax/variables/
func Var(identifier query.Identifier, opts ...internal.VariableOption) *internal.Variable {
	v := &internal.Variable{}
	for _, opt := range opts {
		internal.ConfigureVariable(v, opt)
	}
	v.Identifier = identifier
	return v
}

// Qual qualifies a [variable] with a name/expression. It can be used when:
// - A [variable] is created from an identifier and we want to give it a name
// - A projection item is created (i.e. in a [WITH] or [RETURN] clause)
//
//	Qual(Person{}, "p") -> (p:Person)
//
// If identifier is already registered, or is a string, it becomes the expression of the [variable] and expr
// becomes the alias. If a name is also provided with [Name], we throw.
//
//	<identifier> AS <expr>
//
// [variable]: https://neo4j.com/docs/cypher-manual/current/syntax/variables/
// [WITH]: https://neo4j.com/docs/cypher-manual/current/clauses/with/
// [RETURN]: https://neo4j.com/docs/cypher-manual/current/clauses/return/
func Qual(identifier query.Identifier, expr string, opts ...internal.VariableOption) *internal.Variable {
	// Check if name is provided in opts, if so we make it an alias.
	v := Var(identifier, opts...)
	if v.Name != "" && v.Expression != "" {
		panic(fmt.Errorf(
			`cannot create variable from 2 expressions: Qual(%s, ...) = %+v)`, identifier, v,
		))
	}
	// identifier > expr > name
	if v.Expression != "" {
		v.Name = expr
	} else {
		v.Expression = expr
	}
	return v
}

// Bind binds an existing identifier to a pointer.
// When referring to that [variable], the original identifier can no longer be
// used and is replaced by toPtr.
//
// [variable]: https://neo4j.com/docs/cypher-manual/current/syntax/variables/
func Bind(identifier query.Identifier, toPtr any) *internal.Variable {
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
func Label(pattern string) internal.VariableOption {
	return &internal.Configurer{
		Variable: func(v *internal.Variable) {
			v.Pattern = pattern
		},
	}
}

// VarLength sets the [variable-length expression] of a relationship.
//
// [variable-length expression]: https://neo4j.com/docs/cypher-manual/current/patterns/reference/#variable-length-relationships
func VarLength(varLengthExpr string) internal.VariableOption {
	return &internal.Configurer{
		Variable: func(v *internal.Variable) {
			v.VarLength = varLengthExpr
		},
	}
}

// Props sets the properties of a node or relationship.
// - Keys behave as [pkg/github.com/rlch/neogo/query.PropertyIdentifier]'s
// - Values behave as [pkg/github.com/rlch/neogo/query.ValueIdentifier]'s
type Props = internal.Props

// PropsExpr sets the properties of a node or relationship to the provided
// expression.
func PropsExpr(propsExpr string) internal.VariableOption {
	return &internal.Configurer{
		Variable: func(v *internal.Variable) {
			v.PropsExpression = propsExpr
		},
	}
}
