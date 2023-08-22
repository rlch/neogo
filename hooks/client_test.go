package hooks

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/rlch/neogo/client"
)

func TestNew(t *testing.T) {
	t.Run("creates hook", func(t *testing.T) {
		mp := &PatternsMatcher{}
		cp := &PatternsMatcher{}
		r := New(func(c *Client) HookClient {
			return c.
				Match(mp).
				Create(cp)
		}).After(func(scope client.Scope) error {
			return nil
		})
		r.Restart()
		assert.Equal(t, clauseMatch, r.clause)
		assert.NotEmpty(t, r.matcherList)
		assert.Equal(t, mp, r.matcherList[0])

		assert.NotNil(t, r.next)
		r.hookNode = r.next

		assert.Equal(t, clauseCreate, r.clause)
		assert.NotEmpty(t, r.matcherList)
		assert.Equal(t, mp, r.matcherList[0])
		assert.Nil(t, r.next)
	})
}
