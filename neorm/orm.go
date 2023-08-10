package neorm

import "github.com/rlch/neogo/internal"

type Orm interface {
	Get(patterns internal.Patterns) error
	Create(patterns internal.Patterns) error
	List()
}
