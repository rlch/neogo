package hooks

import (
	"fmt"
	"strings"
)

type ll struct {
	data  int
	next  *ll
	start *ll // Only present for tail
}

func (l *ll) String() string {
	parts := []string{}
	for l != nil {
		parts = append(parts, fmt.Sprintf("%d", l.data))
		l = l.next
	}
	return "[" + strings.Join(parts, ", ") + "]"
}

func fromList(list []int) *ll {
	head := &ll{}
	tail := head
	for i := len(list) - 1; i >= 0; i-- {
		head.data = list[i]
		if i != 0 {
			head = &ll{next: head}
		}
	}
	tail.start = head
	return head
}

func main() {
	hooks := []*ll{
		fromList([]int{1, 2, 3}),
		fromList([]int{1, 2, 3, 4}),
		fromList([]int{2, 3, 4, 5}),
		fromList([]int{3, 4, 5}),
		fromList([]int{3, 5}),
		fromList([]int{2, 4}),
		fromList([]int{4}),
	}

	heads := make(map[int][]*ll)
	for _, h := range hooks {
		heads[h.data] = append(heads[h.data], h)
	}
	removeFromHead := func(h *ll) {
		for i, hh := range heads[h.data] {
			if h == hh {
				heads[h.data] = append(heads[h.data][:i], heads[h.data][i+1:]...)
				return
			}
		}
	}

	matchedHooks := []*ll{}
	seqs := make(map[int][]*ll)
	for _, n := range []int{
		1, 2, 3, 4, 5, 3, 4, 5,
	} {
		newSeqs := make(map[int][]*ll)
		for _, cont := range seqs[n] {
			if cont.next == nil {
				matchedHooks = append(matchedHooks, cont.start)
				removeFromHead(cont.start)
				continue
			}
			newSeqs[cont.next.data] = append(newSeqs[cont.next.data], cont.next)
		}
		if matches, ok := heads[n]; ok {
			for _, match := range matches {
				if match.next == nil {
					matchedHooks = append(matchedHooks, match.start)
					removeFromHead(match.start)
					continue
				}
				newSeqs[match.next.data] = append(newSeqs[match.next.data], match.next)
			}
		}
		seqs = newSeqs
	}

	for _, match := range matchedHooks {
		fmt.Println(match)
	}
}
