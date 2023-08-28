package hooks

import "github.com/rlch/neogo/client"

type Registry struct {
	*reconcilerClient
	hooks        map[string]*Hook
	hookStartMap map[ClauseType][]*Hook
	current      map[ClauseType][]*hookSequence
}

func NewRegistry() *Registry {
	r := &Registry{
		hooks:        make(map[string]*Hook),
		hookStartMap: make(map[ClauseType][]*Hook),
		current:      make(map[ClauseType][]*hookSequence),
	}
	r.reconcilerClient = &reconcilerClient{r}
	return r
}

func (r *Registry) UseHooks(hooks ...*Hook) {
	for _, h := range hooks {
		r.hooks[h.Name] = h
		r.hookStartMap[h.clause] = append(r.hookStartMap[h.clause], h)
	}
}

func (r *Registry) RemoveHooks(names ...string) {
	for _, name := range names {
		delete(r.hooks, name)
	}
}

type hookSequence struct {
	*Hook
	clauseParams [][]any
}

func (r *Registry) Reconcile(scope client.Scope, clause ClauseType, params ...any) {
	nextSeqs := make(map[ClauseType][]*hookSequence)
	matched := []*hookSequence{}
	for _, hook := range r.current[clause] {
		ms := hook.matcherList
		if ms.isStar() || len(ms) == len(params) {
			hook.clauseParams = append(hook.clauseParams, params)
		} else {
			continue
		}
		if hook.next == nil {
			matched = append(matched, hook)
		} else {
			hook.hookNode = hook.next
			nextSeqs[hook.clause] = append(nextSeqs[hook.clause], hook)
		}
	}
	for _, newHook := range r.hookStartMap[clause] {
		ms := newHook.matcherList
		seq := &hookSequence{Hook: newHook}
		if ms.isStar() || len(ms) == len(params) {
			seq.clauseParams = [][]any{params}
		} else {
			continue
		}
		if newHook.next == nil {
			matched = append(matched, seq)
		} else {
			seq.hookNode = seq.next
			nextSeqs[seq.clause] = append(nextSeqs[seq.clause], seq)
		}
	}
Matches:
	for _, match := range matched {
		match.Restart()
		i := 0
		n := match.State()
		defer match.Restart()
		for n.hookNode != nil {
			nParams := len(match.clauseParams[i])
			if n.matcherList.isStar() {
				star := n.matcherList[0].(iStarMatcher)
				star.setLen(nParams)
				for j, matcher := range star.list() {
					ok := matcher.reconcile(match.clauseParams[i][j])
					defer matcher.reset()
					if !ok {
						continue Matches
					}
				}
			} else {
				for j, matcher := range n.matcherList {
					if matcher == nil {
						continue
					}
					matcher.reconcile(match.clauseParams[i][j])
					defer matcher.reset()
				}
			}
			i++
			n.hookNode = n.next
		}
		if match.After != nil {
			match.After(scope)
		}
	}
	r.current = nextSeqs
}
