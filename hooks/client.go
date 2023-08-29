package hooks

import (
	"github.com/rlch/neogo/client"
)

type (
	Hook struct {
		*hookState
		Name   string
		Mutate func(scope client.Scope) error
	}

	HookClient interface {
		State() *hookState
	}
	registrant struct {
		*hookState
	}
)

func (r *registrant) Mutate(
	mutateFn func(scope client.Scope) error,
) *Hook {
	return &Hook{
		hookState: r.hookState,
		Mutate:    mutateFn,
	}
}

func New(createHook func(*Client) HookClient) *registrant {
	s := &hookState{}
	c := s.newClient()
	to := createHook(c)
	state := to.State()
	state.Restart()
	return &registrant{hookState: state}
}

type ClauseType int

const (
	clauseUse ClauseType = iota
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
	hookNode struct {
		clause      ClauseType
		matcherList matchers
		prev        *hookNode
		next        *hookNode
	}
	hookState struct {
		*hookNode
	}

	Client struct {
		*hookState
		*Reader
		*Updater
	}
	Querier struct {
		*hookState
		*Reader
		*Runner
		*Updater
	}
	Reader struct {
		*hookState
	}
	Yielder struct {
		*hookState
		*Querier
	}
	Updater struct {
		*hookState
	}
	Runner struct {
		*hookState
	}
)

func (h *hookState) Extend(c ClauseType, matchers ...Matcher) {
	next := &hookNode{}
	if h.hookNode == nil {
		h.hookNode = next
	} else {
		h.next = next
		next.prev = h.hookNode
		h.hookNode = next
	}
	next.clause = c
	next.matcherList = matchers
}

func (h *hookState) Restart() {
	for h.hookNode != nil && h.prev != nil {
		h.hookNode = h.prev
	}
}

func (h *hookState) State() *hookState {
	return h
}

func (s *hookState) newClient() *Client {
	return &Client{
		hookState: s,
		Reader:    s.newReader(),
		Updater:   s.newUpdater(),
	}
}

func (s *hookState) newQuerier() *Querier {
	return &Querier{
		hookState: s,
		Reader:    s.newReader(),
		Runner:    s.newRunner(),
		Updater:   s.newUpdater(),
	}
}

func (s *hookState) newReader() *Reader {
	return &Reader{hookState: s}
}

func (s *hookState) newYielder() *Yielder {
	return &Yielder{hookState: s}
}

func (s *hookState) newUpdater() *Updater {
	return &Updater{hookState: s}
}

func (s *hookState) newRunner() *Runner {
	return &Runner{hookState: s}
}

func (c *Client) Use(graphExpr *StringMatcher) *Querier {
	c.Extend(clauseUse, graphExpr)
	return c.newQuerier()
}

func (c *Reader) OptionalMatch(patterns GraphPatternMatcher) *Querier {
	c.Extend(clauseOptionalMatch, patterns)
	return c.newQuerier()
}

func (c *Reader) Match(patterns GraphPatternMatcher) *Querier {
	c.Extend(clauseMatch, patterns)
	return c.newQuerier()
}

func (c *Reader) With(identifiers ...*IdentifierMatcher) *Querier {
	c.Extend(clauseWith, toMatcherList(identifiers...)...)
	return c.newQuerier()
}

func (c *Reader) Unwind(expr *IdentifierMatcher, as *StringMatcher) *Querier {
	c.Extend(clauseUnwind, expr, as)
	return c.newQuerier()
}

func (c *Reader) Call(procedure *StringMatcher) *Yielder {
	c.Extend(clauseCall, procedure)
	return c.newYielder()
}

func (c *Reader) Show(command *StringMatcher) *Yielder {
	c.Extend(clauseShow, command)
	return c.newYielder()
}

func (c *Querier) Where(opts ...*WhereMatcher) *Querier {
	c.Extend(clauseWhere, toMatcherList(opts...)...)
	return c.newQuerier()
}

func (c *Yielder) Yield(identifiers ...*IdentifierMatcher) *Querier {
	c.Extend(clauseYield, toMatcherList(identifiers...)...)
	return c.newQuerier()
}

func (c *Reader) Return(identifiers ...*IdentifierMatcher) *Runner {
	c.Extend(clauseReturn, toMatcherList(identifiers...)...)
	return c.newRunner()
}

func (c *Updater) Create(pattern GraphPatternMatcher) *Querier {
	c.Extend(clauseCreate, pattern)
	return c.newQuerier()
}

func (c *Updater) Merge(pattern GraphPatternMatcher, opts *MergeMatcher) *Querier {
	c.Extend(clauseMerge, pattern, opts)
	return c.newQuerier()
}

func (c *Updater) DetachDelete(identifiers ...*IdentifierMatcher) *Querier {
	c.Extend(clauseDetachDelete, toMatcherList(identifiers...)...)
	return c.newQuerier()
}

func (c *Updater) Delete(identifiers ...*IdentifierMatcher) *Querier {
	c.Extend(clauseDelete, toMatcherList(identifiers...)...)
	return c.newQuerier()
}

func (c *Updater) Set(items ...*SetItemMatcher) *Querier {
	c.Extend(clauseSet, toMatcherList(items...)...)
	return c.newQuerier()
}

func (c *Updater) Remove(items ...*RemoveItemMatcher) *Querier {
	c.Extend(clauseRemove, toMatcherList(items...)...)
	return c.newQuerier()
}
