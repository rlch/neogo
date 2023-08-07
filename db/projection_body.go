package db

import "github.com/rlch/neogo/internal"

func With(identifier any, opts ...internal.ProjectionBodyOption) *internal.ProjectionBody {
	m := &internal.ProjectionBody{}
	m.Identifier = identifier
	for _, opt := range opts {
		internal.ConfigureProjectionBody(m, opt)
	}
	return m
}

func Return(identifier any, opts ...internal.ProjectionBodyOption) *internal.ProjectionBody {
	return With(identifier, opts...)
}

func OrderBy(field any, asc bool) internal.ProjectionBodyOption {
	return &internal.Configurer{
		ProjectionBody: func(m *internal.ProjectionBody) {
			if m.OrderBy == nil {
				m.OrderBy = map[any]bool{}
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
