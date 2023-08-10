package hooks

import (
	"context"

	"github.com/rlch/neogo/client"
	"github.com/rlch/neogo/db"
	"github.com/rlch/neogo/internal"
)

func main() {
	created := new(PatternsMatcher)
	matched := new(PatternsMatcher)
	hook := New[client.Querier](func(q client.Client) client.Querier {
		return q.Match(matched).Create(created)
	})
	hook.Before(func(scope client.Scope) error {
		head := created.Head()
		for head != nil {
			name := scope.Name(head.Identifier)
			scope.Binding(name).Interface().(internal.IDSetter).GenerateID()
			head = head.Next()
		}
		return nil
	})
	hook.After(func(scope client.Scope, c client.Querier) (client.Querier, error) {
		return c.Set(db.SetPropValue("n.id", "random-id")), nil
	})

	var c client.Client
	c.
		Match(db.Node("m")).
		Create(db.Node("n")).
		Set(db.SetPropValue("n.id", "random-id")).
		Return("n")
}

type PatternsMatcher struct {
	internal.Patterns

	path    *internal.CypherPath
	pattern *internal.CypherPattern
}

func (p *PatternsMatcher) Head() *internal.NodePattern {
	if p.path != nil {
		return p.path.Pattern
	} else {
		return p.pattern.Patterns[0]
	}
}

func (p *PatternsMatcher) Heads() []*internal.NodePattern {
	if p.path != nil {
		return []*internal.NodePattern{p.path.Pattern}
	} else {
		return p.pattern.Patterns
	}
}

type Hook[C any] interface {
	Before(func(scope client.Scope) error)
	After(func(scope client.Scope, c C) (C, error))
}

func New[C any](createHook func(client.Client) C) Hook[C] {
	s := &hookState{}
	c := s.newClient(internal.NewCypherClient())
	createHook(c)
	return nil
}

type clause int

const (
	clauseStart = iota
	clauseUse
	clauseUnion
	clauseUnionAll
	clauseOptionalMatch
	clauseMatch
	clauseReturn
	clauseWith
	clauseCall
	clauseShow
	clauseSubquery
	clauseCypher
	clauseUnwind
	clauseYield
	clauseWhere
	clauseCreate
	clauseMerge
	clauseDelete
	clauseDetachDelete
	clauseSet
	clauseRemove
	clauseForEach
)

type (
	hookState struct {
		clause clause
		next   *hookState
	}

	clientImpl struct {
		*hookState
		cy *internal.CypherClient
		client.Reader
		client.Updater[client.Querier]
	}
	querierImpl struct {
		*hookState
		cy *internal.CypherQuerier
		client.Reader
		client.Runner
		client.Updater[client.Querier]
	}
	readerImpl struct {
		*hookState
		cy *internal.CypherReader
	}
	yielderImpl struct {
		*hookState
		cy *internal.CypherYielder
		client.Querier
	}
	updaterImpl[To, ToCypher any] struct {
		*hookState
		cy *internal.CypherUpdater[ToCypher]
		to func(ToCypher, clause) To
	}
	runnerImpl struct {
		*hookState
		cy *internal.CypherRunner
	}
)

func (s *hookState) newClient(cy *internal.CypherClient, cc clause) *clientImpl {
	return &clientImpl{
		hookState: s,
		cy:        cy,
		Reader:    s.newReader(cy.CypherReader, cc),
		Updater: newUpdater[client.Querier, *internal.CypherQuerier](
			s,
			cy.CypherUpdater,
			func(c *internal.CypherQuerier, cc clause) client.Querier {
				return s.newQuerier(c, cc)
			},
		),
	}
}

func (s *hookState) newQuerier(cy *internal.CypherQuerier, cc clause) *querierImpl {
	return &querierImpl{
		hookState: s,
		cy:        cy,
		Reader:    s.newReader(cy.CypherReader, cc),
		Runner:    s.newRunner(cy.CypherRunner, cc),
		Updater: newUpdater[client.Querier, *internal.CypherQuerier](
			s,
			cy.CypherUpdater,
			func(c *internal.CypherQuerier, cc clause) client.Querier {
				return s.newQuerier(c, cc)
			},
		),
	}
}

func (s *hookState) newReader(cy *internal.CypherReader, cc clause) *readerImpl {
	return &readerImpl{
		hookState: s,
		cy:        cy,
	}
}

func (s *hookState) newYielder(cy *internal.CypherYielder, cc clause) *yielderImpl {
	return &yielderImpl{
		hookState: s,
		cy:        cy,
	}
}

func newUpdater[To, ToCypher any](
	s *hookState,
	cy *internal.CypherUpdater[ToCypher],
	to func(ToCypher, clause) To,
) *updaterImpl[To, ToCypher] {
	return &updaterImpl[To, ToCypher]{
		hookState: s,
		cy:        cy,
		to:        to,
	}
}

func (s *hookState) newRunner(cy *internal.CypherRunner, cc clause) *runnerImpl {
	return &runnerImpl{hookState: s, cy: cy}
}

func (c *clientImpl) Use(graphExpr string) client.Querier {
	return c.newQuerier(c.cy.Use(graphExpr), clauseUse)
}

func (c *clientImpl) Union(unions ...func(c client.Client) client.Runner) client.Querier {
	inUnions := make([]func(c *internal.CypherClient) *internal.CypherRunner, len(unions))
	for i, union := range unions {
		inUnions[i] = func(cc *internal.CypherClient) *internal.CypherRunner {
			return union(c.newClient(cc)).(*runnerImpl).cy
		}
	}
	return c.newQuerier(c.cy.Union(inUnions...))
}

func (c *clientImpl) UnionAll(unions ...func(c client.Client) client.Runner) client.Querier {
	inUnions := make([]func(c *internal.CypherClient) *internal.CypherRunner, len(unions))
	for i, union := range unions {
		inUnions[i] = func(cc *internal.CypherClient) *internal.CypherRunner {
			return union(c.newClient(cc)).(*runnerImpl).cy
		}
	}
	return c.newQuerier(c.cy.UnionAll(inUnions...))
}

func (c *readerImpl) OptionalMatch(patterns internal.Patterns) client.Querier {
	return c.newQuerier(c.cy.OptionalMatch(patterns))
}

func (c *readerImpl) Match(patterns internal.Patterns) client.Querier {
	return c.newQuerier(c.cy.Match(patterns))
}

func (c *readerImpl) Subquery(subquery func(c client.Client) client.Runner) client.Querier {
	inSubquery := func(cc *internal.CypherClient) *internal.CypherRunner {
		return subquery(c.newClient(cc)).(*runnerImpl).cy
	}
	return c.newQuerier(c.cy.Subquery(inSubquery))
}

func (c *readerImpl) With(identifiers ...any) client.Querier {
	return c.newQuerier(c.cy.With(identifiers...))
}

func (c *readerImpl) Unwind(expr any, as string) client.Querier {
	return c.newQuerier(c.cy.Unwind(expr, as))
}

func (c *readerImpl) Call(procedure string) client.Yielder {
	return c.newYielder(c.cy.Call(procedure))
}

func (c *readerImpl) Show(command string) client.Yielder {
	return c.newYielder(c.cy.Show(command))
}

func (c *readerImpl) Return(identifiers ...any) client.Runner {
	return c.newRunner(c.cy.Return(identifiers...))
}

func (c *readerImpl) Cypher(query func(s client.Scope) string) client.Querier {
	q := c.cy.Cypher(func(scope *internal.Scope) string {
		return query(scope)
	})
	return c.newQuerier(q)
}

func (c *querierImpl) Where(opts ...internal.WhereOption) client.Querier {
	return c.newQuerier(c.cy.Where(opts...))
}

func (c *updaterImpl[To, ToCypher]) Create(pattern internal.Patterns) To {
	return c.to(c.cy.Create(pattern))
}

func (c *updaterImpl[To, ToCypher]) Merge(pattern internal.Pattern, opts ...internal.MergeOption) To {
	return c.to(c.cy.Merge(pattern, opts...))
}

func (c *updaterImpl[To, ToCypher]) DetachDelete(identifiers ...any) To {
	return c.to(c.cy.DetachDelete(identifiers...))
}

func (c *updaterImpl[To, ToCypher]) Delete(identifiers ...any) To {
	return c.to(c.cy.Delete(identifiers...))
}

func (c *updaterImpl[To, ToCypher]) Set(items ...internal.SetItem) To {
	return c.to(c.cy.Set(items...))
}

func (c *updaterImpl[To, ToCypher]) Remove(items ...internal.RemoveItem) To {
	return c.to(c.cy.Remove(items...))
}

func (c *updaterImpl[To, ToCypher]) ForEach(identifier, elementsExpr any, do func(c client.Updater[any])) To {
	return c.to(c.cy.ForEach(identifier, elementsExpr, func(c *internal.CypherUpdater[any]) {
	}))
}

func (c *yielderImpl) Yield(identifiers ...any) client.Querier {
	return c.newQuerier(c.cy.Yield(identifiers...))
}

func (c *runnerImpl) Run(ctx context.Context) (err error) {
	return nil
}
