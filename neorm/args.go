package neorm

import (
	"errors"
	"fmt"

	"github.com/rlch/neogo/db"
	"github.com/rlch/neogo/internal"
)

var errInvalidConditionArgs = errors.New("expected condition to be ICondition, <key> <op> <value> or <expr> <args> ...")

func argsToCondition(args ...any) (internal.ICondition, error) {
	if len(args) == 0 {
		return nil, nil
	}
	if len(args) == 1 {
		if cond, ok := args[0].(internal.ICondition); ok {
			return cond, nil
		} else if !ok {
			return nil, errInvalidConditionArgs
		}
	}
	if _, ok := args[0].(internal.ICondition); ok {
		conds := make([]internal.ICondition, len(args))
		for i, arg := range args {
			if cond, ok := arg.(internal.ICondition); ok {
				conds[i] = cond
			} else {
				return nil, fmt.Errorf("expected all args to be ICondition, but arg %d is %T", i, arg)
			}
		}
		return db.And(conds...), nil
	}
	tryParseCond := func() internal.ICondition {
		key := args[0]
		op, ok := args[1].(string)
		if !ok {
			return nil
		}
		value := args[2]
		return db.Cond(key, op, value)
	}
	if len(args) == 3 {
		if cond := tryParseCond(); cond != nil {
			return cond, nil
		}
	}
	query, ok := args[0].(string)
	if !ok {
		return nil, errInvalidConditionArgs
	}
	return db.Expr(query, args[1:]...), nil
}
