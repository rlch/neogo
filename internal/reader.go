package internal

import (
	"fmt"
)

func (c *cypherReader) Match(patterns Patterns, options ...MatchOption) *cypherQuerier {
	for _, pattern := range patterns.nodes() {
		for _, option := range options {
			option.ConfigureMatchOptions(&pattern.MatchOptions)
		}
	}
	c.writeReadingClause(patterns.nodes())
	return newCypherQuerier(c.cypher)
}

func (c *cypherReader) With(variables ...any) *cypherQuerier {
	c.writeProjectionBodyClause("WITH", variables...)
	return newCypherQuerier(c.cypher)
}

func (c *cypherReader) Unwind(expr any, as string) *cypherQuerier {
	c.catch(func() {
		c.WriteString("UNWIND ")
		m := c.register(expr, nil)
		fmt.Fprintf(c, "%s AS %s", m.name, as)
		// Replace name with alias
		m.alias = as
		c.replaceBinding(m)
		c.newline()
	})
	return newCypherQuerier(c.cypher)
}
