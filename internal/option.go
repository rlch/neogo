package internal

func ConfigureMerge(o *Merge, configurer MergeOption) {
	configurer.configureMerge(o)
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
	Merge          func(*Merge)
	Variable       func(*Variable)
	ProjectionBody func(*ProjectionBody)
	Where          func(*Where)
}

var _ interface {
	MergeOption
	VariableOption
	ProjectionBodyOption
	WhereOption
} = (*Configurer)(nil)

func (c *Configurer) configureMerge(o *Merge) {
	c.Merge(o)
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
	MergeOption interface {
		configureMerge(*Merge)
	}
	Merge struct {
		OnCreate []SetItem
		OnMatch  []SetItem
	}
)

type (
	VariableOption interface {
		configureVariable(*Variable)
	}
	Variable struct {
		Identifier any
		Bind       any
		Name       string
		// If both name and expr are provided, name is used as an alias
		Expression      string
		Where           *Where
		Props           Props
		PropsExpression string
		Pattern         string
		VarLength       string
	}
)

type (
	ProjectionBodyOption interface {
		configureProjectionBody(*ProjectionBody)
	}
	ProjectionBody struct {
		selectionSubClause

		Identifier any
		Distinct   bool
	}
	selectionSubClause struct {
		// Field name -> true if ascending
		OrderBy map[any]bool
		Skip    string
		Limit   string
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
		Identifier any
		Expr       *Expr
		Conds      []*Condition
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
	Expr struct {
		Value string
		Args  []any
	}
)

var (
	_ ICondition = Expr{}
	_ interface {
		WhereOption
		ICondition
	} = (*Condition)(nil)
)

func (e Expr) configureWhere(w *Where) {
	w.Expr = &e
}

func (e Expr) Condition() *Condition {
	return &Condition{Key: e}
}

func (c *Condition) configureWhere(w *Where) {
	w.Conds = append(w.Conds, c)
}

func (c *Condition) Condition() *Condition {
	return c
}

type Props map[any]any

func (p Props) configureVariable(v *Variable) {
	v.Props = p
}

type (
	SetItem struct {
		PropIdentifier any
		ValIdentifier  any
		Merge          bool
		Labels         []string
	}
	RemoveItem struct {
		PropIdentifier any
		Labels         []string
	}
)

type Param struct {
	Name  string
	Value *any
}
