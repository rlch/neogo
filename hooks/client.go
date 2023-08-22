package hooks

import (
	"github.com/rlch/neogo/client"
)

type (
	Hook struct {
		*hookState
		Name   string
		Before HookFn
		After  HookFn
	}
	HookFn func(scope client.Scope) error

	HookClient interface {
		State() *hookState
	}
	registrant struct {
		*hookState
	}
)

func (r *registrant) Before(do HookFn) *Hook {
	return &Hook{
		hookState: r.hookState,
		Before:    do,
	}
}

func (r *registrant) After(do HookFn) *Hook {
	return &Hook{
		hookState: r.hookState,
		After:     do,
	}
}

func New(createHook func(*Client) HookClient) *registrant {
	s := &hookState{}
	c := s.newClient()
	to := createHook(c)
	state := to.State()
	state.Restart()
	return &registrant{
		hookState: state,
	}
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
		*reader
		*updater
	}
	querier struct {
		*hookState
		*reader
		*runner
		*updater
	}
	reader struct {
		*hookState
	}
	yielder struct {
		*hookState
		*querier
	}
	updater struct {
		*hookState
	}
	runner struct {
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
		reader:    s.newReader(),
		updater:   s.newUpdater(),
	}
}

func (s *hookState) newQuerier() *querier {
	return &querier{
		hookState: s,
		reader:    s.newReader(),
		runner:    s.newRunner(),
		updater:   s.newUpdater(),
	}
}

func (s *hookState) newReader() *reader {
	return &reader{hookState: s}
}

func (s *hookState) newYielder() *yielder {
	return &yielder{hookState: s}
}

func (s *hookState) newUpdater() *updater {
	return &updater{hookState: s}
}

func (s *hookState) newRunner() *runner {
	return &runner{hookState: s}
}

func (c *Client) Use(graphExpr *StringMatcher) *querier {
	c.Extend(clauseUse, graphExpr)
	return c.newQuerier()
}

func (c *reader) OptionalMatch(patterns GraphPatternMatcher) *querier {
	c.Extend(clauseOptionalMatch, patterns)
	return c.newQuerier()
}

func (c *reader) Match(patterns GraphPatternMatcher) *querier {
	c.Extend(clauseMatch, patterns)
	return c.newQuerier()
}

func (c *reader) With(identifiers ...*IdentifierMatcher) *querier {
	c.Extend(clauseWith, toMatcherList(identifiers...)...)
	return c.newQuerier()
}

func (c *reader) Unwind(expr *IdentifierMatcher, as *StringMatcher) *querier {
	c.Extend(clauseUnwind, expr, as)
	return c.newQuerier()
}

func (c *reader) Call(procedure *StringMatcher) *yielder {
	c.Extend(clauseCall, procedure)
	return c.newYielder()
}

func (c *reader) Show(command *StringMatcher) *yielder {
	c.Extend(clauseShow, command)
	return c.newYielder()
}

func (c *querier) Where(opts ...*WhereMatcher) *querier {
	c.Extend(clauseWhere, toMatcherList(opts...)...)
	return c.newQuerier()
}

func (c *yielder) Yield(identifiers ...*IdentifierMatcher) *querier {
	c.Extend(clauseYield, toMatcherList(identifiers...)...)
	return c.newQuerier()
}

func (c *reader) Return(identifiers ...*IdentifierMatcher) *runner {
	c.Extend(clauseReturn, toMatcherList(identifiers...)...)
	return c.newRunner()
}

func (c *updater) Create(pattern GraphPatternMatcher) *querier {
	c.Extend(clauseCreate, pattern)
	return c.newQuerier()
}

func (c *updater) Merge(pattern GraphPatternMatcher, opts *MergeMatcher) *querier {
	c.Extend(clauseMerge, pattern, opts)
	return c.newQuerier()
}

func (c *updater) DetachDelete(identifiers ...*IdentifierMatcher) *querier {
	c.Extend(clauseDetachDelete, toMatcherList(identifiers...)...)
	return c.newQuerier()
}

func (c *updater) Delete(identifiers ...*IdentifierMatcher) *querier {
	c.Extend(clauseDelete, toMatcherList(identifiers...)...)
	return c.newQuerier()
}

func (c *updater) Set(items ...*SetItemMatcher) *querier {
	c.Extend(clauseSet, toMatcherList(items...)...)
	return c.newQuerier()
}

func (c *updater) Remove(items ...*RemoveItemMatcher) *querier {
	c.Extend(clauseRemove, toMatcherList(items...)...)
	return c.newQuerier()
}
