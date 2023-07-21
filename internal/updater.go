package internal

func (c *cypherUpdater) Subquery(subquery runner) *cypherQuerier {
	return nil
}

func (c *cypherUpdater) Create(pattern Patterns) *cypherQuerier {
	c.writeUpdatingClause("CREATE", pattern.nodes())
	return newCypherQuerier(c.cypher)
}

func (c *cypherUpdater) Merge(pattern Patterns) *cypherQuerier {
	c.writeUpdatingClause("MERGE", pattern.nodes())
	return newCypherQuerier(c.cypher)
}

func (c *cypherUpdater) Update(payload any) *cypherQuerier {
	return nil
}

func (c *cypherUpdater) Delete(paths ...Path) *cypherQuerier {
	return nil
}
