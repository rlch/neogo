package neo4jgorm

type IRelationship interface {
	IsRelationship()
}

type Relationship struct{}

func (Relationship) IsRelationship() {}
