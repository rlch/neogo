package hooks

import (
	"github.com/rlch/neogo/client"
	"github.com/rlch/neogo/internal"
)

type Matcher interface {
	isStar() bool
	reconcile(data any) bool
	reset()
}

type matchers []Matcher

func (m matchers) isStar() bool {
	return len(m) == 1 && m[0] != nil && m[0].isStar()
}

var (
	_ Matcher = (*IdentifierMatcher)(nil)
	_ Matcher = (*StringMatcher)(nil)
	_ Matcher = (*PatternMatcher)(nil)
	_ Matcher = (*PatternsMatcher)(nil)
	_ Matcher = (*WhereMatcher)(nil)
	_ Matcher = (*MergeMatcher)(nil)
	_ Matcher = (*SetItemMatcher)(nil)
	_ Matcher = (*RemoveItemMatcher)(nil)
)

type baseMatcher struct{}

func (m *baseMatcher) isStar() bool { return false }

type iStarMatcher interface {
	Matcher
	setLen(int)
	list() []Matcher
}

type StarMatcher[M any, PM interface {
	Matcher
	*M
}] []PM

func (m *StarMatcher[M, PM]) isStar() bool { return true }

func (m *StarMatcher[M, PM]) reconcile(data any) bool {
	panic("iterate through each matcher to reconcile")
}

func (m *StarMatcher[M, PM]) setLen(i int) {
	*m = make([]PM, i)
}

func (m *StarMatcher[M, PM]) list() []Matcher {
	ms := make([]Matcher, len(*m))
	for i, m := range *m {
		ms[i] = m
	}
	return ms
}

func (m *StarMatcher[M, PM]) reset() {
	for i := range *m {
		(*m)[i].reset()
	}
	m.setLen(0)
}

// Star matches zero or more instances of the given matcher. It is useful when
// a hook can be applied to a variable number of arguments.
func Star[M any, PM interface {
	Matcher
	*M
}]() StarMatcher[M, PM] {
	return StarMatcher[M, PM](
		[]PM{},
	)
}

func toMatcherList[M Matcher](matchers ...M) []Matcher {
	ms := make([]Matcher, len(matchers))
	for i, m := range matchers {
		ms[i] = m
	}
	return ms
}

type IdentifierMatcher struct {
	*baseMatcher

	Identifier interface {
		client.Identifier |
			client.PropertyIdentifier |
			client.ValueIdentifier
	}
}

func (m *IdentifierMatcher) reconcile(data any) bool {
	m.Identifier = data
	return true
}

func (m *IdentifierMatcher) reset() {
	m.Identifier = nil
}

type StringMatcher struct {
	*baseMatcher

	String string
}

func (m *StringMatcher) reconcile(data any) bool {
	if s, ok := data.(string); ok {
		m.String = s
		return true
	}
	return false
}

func (m *StringMatcher) reset() {
	m.String = ""
}

type GraphPatternMatcher interface {
	Matcher
	isGraphPattern()
}

type PatternMatcher struct {
	*baseMatcher

	path *internal.CypherPath
}

func (m *PatternMatcher) reconcile(data any) bool {
	if p, ok := data.(*internal.CypherPath); ok {
		m.path = p
		return true
	}
	return false
}

func (m *PatternMatcher) reset() {
	m.path = nil
}

func (m *PatternMatcher) Head() *internal.NodePattern {
	return m.path.Pattern
}

func (m *PatternMatcher) isGraphPattern() {}

type PatternsMatcher struct {
	*baseMatcher

	pattern *internal.CypherPattern
}

func (m *PatternsMatcher) reconcile(data any) bool {
	if p, ok := data.(*internal.CypherPattern); ok {
		m.pattern = p
		return true
	}
	return false
}

func (m *PatternsMatcher) reset() {
	m.pattern = nil
}

func (m *PatternsMatcher) Heads() []*internal.NodePattern {
	return m.pattern.Patterns
}

func (m *PatternsMatcher) isGraphPattern() {}

type WhereMatcher struct {
	*baseMatcher

	Where *internal.Where
}

func (m *WhereMatcher) reconcile(data any) bool {
	if w, ok := data.(*internal.Where); ok {
		m.Where = w
		return true
	}
	return false
}

func (m *WhereMatcher) reset() {
	m.Where = nil
}

type MergeMatcher struct {
	*baseMatcher

	Merge *internal.Merge
}

func (m *MergeMatcher) reconcile(data any) bool {
	if w, ok := data.(*internal.Merge); ok {
		m.Merge = w
		return true
	}
	return false
}

func (m *MergeMatcher) reset() {
	m.Merge = nil
}

type SetItemMatcher struct {
	*baseMatcher

	SetItem *internal.SetItem
}

func (m *SetItemMatcher) reconcile(data any) bool {
	if s, ok := data.(*internal.SetItem); ok {
		m.SetItem = s
		return true
	}
	return false
}

func (m *SetItemMatcher) reset() {
	m.SetItem = nil
}

type RemoveItemMatcher struct {
	*baseMatcher

	RemoveItem *internal.RemoveItem
}

func (m *RemoveItemMatcher) reconcile(data any) bool {
	if r, ok := data.(*internal.RemoveItem); ok {
		m.RemoveItem = r
		return true
	}
	return false
}

func (m *RemoveItemMatcher) reset() {
	m.RemoveItem = nil
}
