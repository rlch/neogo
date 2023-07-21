package internal

type Configurer struct {
	MatchOptions   func(*MatchOptions)
	Variable       func(*Variable)
	ProjectionBody func(*ProjectionBody)
	Where          func(*Where)
}

var _ interface {
	MatchOption
	VariableOption
	ProjectionBodyOption
	WhereOption
} = (*Configurer)(nil)

func (c *Configurer) ConfigureMatchOptions(o *MatchOptions) {
	c.MatchOptions(o)
}

func (c *Configurer) ConfigureVariable(v *Variable) {
	c.Variable(v)
}

func (c *Configurer) ConfigureProjectionBody(p *ProjectionBody) {
	c.ProjectionBody(p)
}

func (c *Configurer) ConfigureWhere(w *Where) {
	c.Where(w)
}

type (
	MatchOption interface {
		ConfigureMatchOptions(*MatchOptions)
	}
	MatchOptions struct {
		Optional bool
	}
)

type (
	VariableOption interface {
		ConfigureVariable(*Variable)
	}
	Variable struct {
		Entity any
		Bind   any
		Name   string
		// If both name and expr are provided, name is used as an alias
		Expr    Expr
		Where   *Where
		Omit    []string
		Select  []string
		Props   map[any]Expr
		Pattern Expr
	}
)

type (
	ProjectionBodyOption interface {
		ConfigureProjectionBody(*ProjectionBody)
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
		ConfigureWhere(*Where)
	}
	ICondition interface {
		ConfigureWhere(*Where)
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
		Path  Path
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

func (e Expr) ConfigureWhere(w *Where) {
	w.Expr = string(e)
}

func (c *Condition) ConfigureWhere(w *Where) {
	w.Conds = append(w.Conds, c)
}

func (c *Condition) Condition() *Condition {
	return c
}
