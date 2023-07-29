package neogo

type IRelationship interface {
	IsRelationship()
}

type Relationship struct{}

func (Relationship) IsRelationship() {}
