package neorm

import (
	"errors"
	"fmt"

	"github.com/rlch/neogo/internal"
)

var ErrInvalidValue = func(value any) error { return fmt.Errorf("invalid value: %T", value) }

func NewTransaction() Transaction {
	return &tx{}
}

type Transaction interface {
	Preload(query string, args ...any) Transaction
	Find(dest any, args ...any) error
}

type (
	tx struct {
		err      error
		preloads []preload
	}
	preload struct {
		query     string
		condition internal.ICondition
	}
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

func (t *tx) Preload(query string, args ...any) Transaction {
	condition, err := argsToCondition(args...)
	if err != nil {
		t.AddError(err)
	}
	t.preloads = append(t.preloads, preload{
		query:     query,
		condition: condition,
	})
	return t
}

func (t *tx) Find(dest any, args ...any) error {
	// var rootPattern internal.Pattern
	// if _, ok := dest.(neogo.INode); ok {
	// 	rootPattern = internal.NewNode(dest)
	// } else if _, ok := dest.(neogo.IRelationship); ok {
	// 	rootPattern = internal.NewNode(nil).To(dest, nil)
	// } else {
	// 	return ErrInvalidValue(dest)
	// }
	// for _, selectQuery := range t.selects {
	// }
	return nil
}
