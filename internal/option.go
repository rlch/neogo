package internal

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
	MergeOptions   func(*MergeOptions)
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
		Identifier any
		Bind       any
		Name       string
		// If both name and expr are provided, name is used as an alias
		Expr      Expr
		Where     *Where
		Props     Props
		Pattern   Expr
		VarLength Expr
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
		condition() *Condition
	}
	Where struct {
		Identifier any
		Expr       string
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
	Expr string
)

var (
	_ ICondition = (Expr)("")
	_ interface {
		WhereOption
		ICondition
	} = (*Condition)(nil)
)

func ToCondition(i ICondition) *Condition {
	return i.condition()
}

func (e Expr) configureWhere(w *Where) {
	w.Expr = string(e)
}

func (e Expr) condition() *Condition {
	return &Condition{Key: e}
}

func (c *Condition) configureWhere(w *Where) {
	w.Conds = append(w.Conds, c)
}

func (c *Condition) condition() *Condition {
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
