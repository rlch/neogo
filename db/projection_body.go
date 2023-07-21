package db

import "github.com/rlch/neo4j-gorm/internal"

func With(entity any, opts ...internal.ProjectionBodyOption) *internal.ProjectionBody {
	m := &internal.ProjectionBody{}
	m.Entity = entity
	for _, opt := range opts {
		opt.ConfigureProjectionBody(m)
	}
	return m
}

func Return(entity any, opts ...internal.ProjectionBodyOption) *internal.ProjectionBody {
	return With(entity, opts...)
}

func OrderBy(field string, asc bool) internal.ProjectionBodyOption {
	return &internal.Configurer{
		ProjectionBody: func(m *internal.ProjectionBody) {
			if m.OrderBy == nil {
				m.OrderBy = map[string]bool{}
			}
			m.OrderBy[field] = asc
		},
	}
}

func Skip(expr string) internal.ProjectionBodyOption {
	return &internal.Configurer{
		ProjectionBody: func(m *internal.ProjectionBody) {
			m.Skip = Expr(expr)
		},
	}
}

func Limit(expr string) internal.ProjectionBodyOption {
	return &internal.Configurer{
		ProjectionBody: func(m *internal.ProjectionBody) {
			m.Limit = Expr(expr)
		},
	}
}

var Distinct internal.ProjectionBodyOption = &internal.Configurer{
	ProjectionBody: func(m *internal.ProjectionBody) {
		m.Distinct = true
	},
}

func Paginate(options PaginationOptions) internal.ProjectionBodyOption {
	return &internal.Configurer{
		ProjectionBody: func(pb *internal.ProjectionBody) {
			o := internal.PaginationOptions{}
			options.apply(&o)
			pb.Pagination = o
		},
	}
}

type (
	PaginationOptions interface {
		apply(*internal.PaginationOptions)
	}
	ForwardOptions struct {
		First  int
		After  string
		SortBy string
		Desc   bool
	}
	BackwardOptions struct {
		Last   int
		Before string
		SortBy string
		Desc   bool
	}
)

func (o ForwardOptions) apply(opts *internal.PaginationOptions) {
	opts.First = o.First
	opts.After = o.After
	opts.SortBy = o.SortBy
	opts.Desc = o.Desc
}

func (o BackwardOptions) apply(opts *internal.PaginationOptions) {
	opts.Last = o.Last
	opts.Before = o.Before
	opts.SortBy = o.SortBy
	opts.Desc = o.Desc
}
