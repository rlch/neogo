package neorm

import (
	"errors"
	"fmt"
	"strings"

	"github.com/rlch/neogo"
	"github.com/rlch/neogo/db"
	"github.com/rlch/neogo/internal"
)

func NewTransaction() Tx {
	return &tx{}
}

type Tx interface {
	Where(conds ...any) Tx
	Query(query string, opts ...any) Tx
	Find(dest any, opts ...any) error
	Delete(value any, conds ...any) error
	Create(value any) error
	Save(value any) error
}

type (
	tx struct {
		registry   *internal.Registry
		driver     neogo.Driver
		err        error
		queries    []querySpec
		conditions []internal.ICondition
	}
	querySpec struct {
		selectors []*querySelector
		condition internal.ICondition
	}
	querySelector struct {
		// name is the name of the variable that is used in the query.
		name string
		// field is the name of the field should be loaded.
		field string
		// props represents the properties of the JSON payload that should be loaded into the struct.
		// If this is empty, all properties will be loaded. If this is nil, none will be loaded.
		props []string
	}
)

var (
	errInvalidValue         = func(value any) error { return fmt.Errorf("invalid value: %T", value) }
	errInvalidConditionArgs = errors.New("expected condition to be ICondition, <key> <op> <value> or <expr> <args>")
)

func (t *tx) AddError(err error) {
	if err == nil {
		return
	}
	if t.err != nil {
		t.err = errors.Join(t.err, err)
	} else {
		t.err = err
	}
}

func (t *tx) Where(conds ...any) Tx {
	condition, err := argsToCondition(conds...)
	if err != nil {
		t.AddError(err)
	}
	t.conditions = append(t.conditions, condition)
	return t
}

func (t *tx) Query(query string, opts ...any) Tx {
	queryParts, err := newQueryParser(query).parse()
	if err != nil {
		t.AddError(err)
		return t
	}
	qs := querySpec{
		selectors: queryParts,
	}
	condition, err := argsToCondition(opts...)
	if err != nil {
		t.AddError(err)
	}
	qs.condition = condition
	return t
}

func (t *tx) Find(dest any, args ...any) error {
	// if err := t.resolveQueries(dest); err != nil {
	// 	t.AddError(err)
	// }
	return nil
}

func (t *tx) Delete(value any, conds ...any) error {
	return nil
}

func (t *tx) Create(value any) error {
	return nil
}

func (t *tx) Save(src any) error {
	return nil
}

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

func (q querySpec) String() string {
	var buf strings.Builder
	for i, selector := range q.selectors {
		if selector.name != "" || selector.props != nil {
			buf.WriteString(selector.name)
			buf.WriteString(fmt.Sprintf("{%s}", strings.Join(selector.props, ",")))
			buf.WriteString(":")
		}
		buf.WriteString(selector.field)
		if i != len(q.selectors)-1 {
			buf.WriteString(" ")
		}
	}
	return buf.String()
}
