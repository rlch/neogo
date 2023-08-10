package db

import (
	"github.com/rlch/neogo/client"
	"github.com/rlch/neogo/internal"
)

// Where creates an inline [WHERE] clause.
// Can be used for [nodes + relationship patterns] and [WITH] clauses.
//
// [WHERE]: https://neo4j.com/docs/cypher-manual/current/clauses/where/
// [nodes + relationship patterns]: https://neo4j.com/docs/cypher-manual/current/patterns/reference
// [WITH]: https://neo4j.com/docs/cypher-manual/current/clauses/where/#usage-with-with-clause
func Where(opts ...internal.WhereOption) interface {
	internal.VariableOption
	internal.ProjectionBodyOption
	internal.WhereOption
} {
	return &internal.Configurer{
		Where: func(w *internal.Where) {
			for _, opt := range opts {
				internal.ConfigureWhere(w, opt)
			}
		},
		Variable: func(v *internal.Variable) {
			v.Where = &internal.Where{}
			for _, opt := range opts {
				internal.ConfigureWhere(v.Where, opt)
			}
		},
		ProjectionBody: func(pb *internal.ProjectionBody) {
			pb.Where = &internal.Where{}
			for _, opt := range opts {
				internal.ConfigureWhere(pb.Where, opt)
			}
		},
	}
}

// Cond creates a condition for use in a [WHERE] clause.
//
//	WHERE <key> <op> <value>
//
// [WHERE]: https://neo4j.com/docs/cypher-manual/current/clauses/where/
func Cond(
	key client.PropertyIdentifier,
	op string,
	value client.ValueIdentifier,
) internal.ICondition {
	return &internal.Condition{
		Key:   key,
		Op:    op,
		Value: value,
	}
}

// Or creates an OR condition for use in a [WHERE] clause.
//
//	WHERE <cond> OR <cond> ... OR <cond>
//
// [WHERE]: https://neo4j.com/docs/cypher-manual/current/clauses/where/
func Or(conds ...internal.ICondition) internal.ICondition {
	ors := make([]*internal.Condition, len(conds))
	for i, cond := range conds {
		ors[i] = internal.ToCondition(cond)
	}
	return &internal.Condition{
		Or: ors,
	}
}

// And creates an AND condition for use in a [WHERE] clause.
//
//	WHERE <cond> AND <cond> ... AND <cond>
//
// [WHERE]: https://neo4j.com/docs/cypher-manual/current/clauses/where/
func And(conds ...internal.ICondition) internal.ICondition {
	ands := make([]*internal.Condition, len(conds))
	for i, cond := range conds {
		ands[i] = internal.ToCondition(cond)
	}
	return &internal.Condition{
		And: ands,
	}
}

// Xor creates an XOR condition for use in a [WHERE] clause.
//
//	WHERE <cond> XOR <cond> ... XOR <cond>
//
// [WHERE]: https://neo4j.com/docs/cypher-manual/current/clauses/where/
func Xor(conds ...internal.ICondition) internal.ICondition {
	xors := make([]*internal.Condition, len(conds))
	for i, cond := range conds {
		xors[i] = internal.ToCondition(cond)
	}
	return &internal.Condition{
		Xor: xors,
	}
}

// Not creates a NOT condition for use in a [WHERE] clause.
//
//	WHERE NOT <cond>
//
// [WHERE]: https://neo4j.com/docs/cypher-manual/current/clauses/where/
func Not(cond internal.ICondition) internal.ICondition {
	c := internal.ToCondition(cond)
	c.Not = true
	return c
}
