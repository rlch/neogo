package neorm

import "github.com/rlch/neogo"

type Person struct {
	neogo.Node `neo4j:"Person"`
	Name       string     `json:"name"`
	Cats       []*IsOwner `neo4j:"->" json:"-"`
	Friends    []*Person  `neo4j:"FRIENDS_WITH>" json:"-"`
}

type Cat struct {
	neogo.Node `neo4j:"Cat"`
	Name       string   `json:"name"`
	Owner      *IsOwner `neo4j:"<-" json:"-"`
}

type IsOwner struct {
	neogo.Relationship `neo4j:"IS_OWNER"`
	ID                 string  `json:"id"`
	Active             bool    `json:"active"`
	Owner              *Person `neo4j:"startNode" json:"-"`
	Cat                *Cat    `neo4j:"endNode" json:"-"`
}
