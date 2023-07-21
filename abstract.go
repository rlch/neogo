package neo4jgorm

type IAbstract interface {
	INode
	IsAbstract()
	Implementers() []IAbstract
}

type Abstract struct{}

func (*Abstract) IsAbstract() {}
