package internal

import "github.com/goccy/go-json"

func ConfigureMatchOptions(o *MatchOptions, configurer MatchOption) {
	configurer.configureMatchOptions(o)
}

func ConfigureMergeOptions(o *MergeOptions, configurer MergeOption) {
	configurer.configureMergeOptions(o)
}

func ConfigureVariable(v *Variable, configurer VariableOption) {
	configurer.configureVariable(v)
}

func ConfigureProjectionBody(p *ProjectionBody, configurer ProjectionBodyOption) {
	configurer.configureProjectionBody(p)
}

func ConfigureWhere(w *Where, configurer WhereOption) {
	configurer.configureWhere(w)
}

type Configurer struct {
	MatchOptions   func(*MatchOptions)
	MergeOptions   func(*MergeOptions)
	Variable       func(*Variable)
	ProjectionBody func(*ProjectionBody)
	Where          func(*Where)
}

var _ interface {
	MatchOption
	MergeOption
	VariableOption
	ProjectionBodyOption
	WhereOption
} = (*Configurer)(nil)

func (c *Configurer) configureMatchOptions(o *MatchOptions) {
	c.MatchOptions(o)
}

func (c *Configurer) configureMergeOptions(o *MergeOptions) {
	c.MergeOptions(o)
}

func (c *Configurer) configureVariable(v *Variable) {
	c.Variable(v)
}

func (c *Configurer) configureProjectionBody(p *ProjectionBody) {
	c.ProjectionBody(p)
}

func (c *Configurer) configureWhere(w *Where) {
	c.Where(w)
}

type (
	MatchOption interface {
		configureMatchOptions(*MatchOptions)
	}
	MatchOptions struct {
		Optional bool
	}
	MergeOption interface {
		configureMergeOptions(*MergeOptions)
	}
	MergeOptions struct {
		OnCreate []SetItem
		OnMatch  []SetItem
	}
)

type (
	VariableOption interface {
		configureVariable(*Variable)
	}
	Variable struct {
		Entity any
		Bind   any
		Name   string
		// If both name and expr are provided, name is used as an alias
		Expr       Expr
		Where      *Where
		Select     *json.FieldQuery
		Props      map[any]Expr
		Pattern    Expr
		Quantifier Expr
	}
)

type (
	ProjectionBodyOption interface {
		configureProjectionBody(*ProjectionBody)
	}
	ProjectionBody struct {
		selectionSubClause

		Entity     any
		Pagination PaginationOptions
		Distinct   bool
	}
	PaginationOptions struct {
		First  int
		Last   int
		After  string
		Before string
		SortBy string
		Desc   bool
	}
	selectionSubClause struct {
		// Field name -> true if ascending
		OrderBy map[string]bool
		Skip    Expr
		Limit   Expr
		Where   *Where
	}
)

func (s *ProjectionBody) hasProjectionClauses() bool {
	return len(s.OrderBy) > 0 || s.Limit != "" || s.Skip != "" || s.Where != nil
}

type (
	WhereOption interface {
		configureWhere(*Where)
	}
	ICondition interface {
		WhereOption
		Condition() *Condition
	}
	Where struct {
		Entity any
		Expr   string
		Conds  []*Condition
	}
	Condition struct {
		Xor   []*Condition
		Or    []*Condition
		And   []*Condition
		Path  Pattern
		Key   any
		Op    string
		Value any
		Not   bool
	}
	Expr string
)

var (
	_ WhereOption = (Expr)("")
	_ interface {
		WhereOption
		ICondition
	} = (*Condition)(nil)
)

func (e Expr) configureWhere(w *Where) {
	w.Expr = string(e)
}

func (c *Condition) configureWhere(w *Where) {
	w.Conds = append(w.Conds, c)
}

func (c *Condition) Condition() *Condition {
	return c
}

type Props map[any]Expr

func (p Props) configureVariable(v *Variable) {
	v.Props = p
}

type (
	SetItem struct {
		Entity any
		Value  any
		Merge  bool
		Labels []string
	}
	RemoveItem struct {
		Entity any
		Labels []string
	}
)

type Param struct {
	Name  string
	Value *any
}
