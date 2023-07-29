package db

import "github.com/rlch/neogo/internal"

// Can be used for nodes + relationship patterns, WITH clauses and WHERE
// clauses.
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

func Cond(key any, op string, value any) internal.ICondition {
	return &internal.Condition{
		Key:   key,
		Op:    op,
		Value: value,
	}
}

func Or(conds ...internal.ICondition) internal.ICondition {
	ors := make([]*internal.Condition, len(conds))
	for i, cond := range conds {
		ors[i] = cond.Condition()
	}
	return &internal.Condition{
		Or: ors,
	}
}

func And(conds ...internal.ICondition) internal.ICondition {
	ands := make([]*internal.Condition, len(conds))
	for i, cond := range conds {
		ands[i] = cond.Condition()
	}
	return &internal.Condition{
		And: ands,
	}
}

func Xor(conds ...internal.ICondition) internal.ICondition {
	xors := make([]*internal.Condition, len(conds))
	for i, cond := range conds {
		xors[i] = cond.Condition()
	}
	return &internal.Condition{
		Xor: xors,
	}
}

func Not(cond internal.ICondition) internal.ICondition {
	c := cond.Condition()
	c.Not = true
	return c
}
