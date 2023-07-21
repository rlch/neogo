package internal

func (c *cypherQuerier) Where(opts ...WhereOption) *cypherQuerier {
	where := &Where {}
	for _, opt := range opts {
		opt.ConfigureWhere(where)
	}
	c.writeWhereClause(where, false)
	return newCypherQuerier(c.cypher)
}

func (c *cypherQuerier) Find(matches ...any) *cypherRunner {
	c.writeProjectionBodyClause("RETURN", matches...)
	return newCypherRunner(c.cypher)
}
