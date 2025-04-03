package neorm

import "github.com/rlch/neogo"

type Person struct {
	neogo.Node `neo4j:"Person"`
	Name       string    `json:"name"`
	Cats       []IsOwner `neo4j:"->" json:"-"`
	Friends    []Friend  `neo4j:"->" json:"-"`
}

type Cat struct {
	neogo.Node `neo4j:"Cat"`
	Name       string  `json:"name"`
	Owner      *Person `neo4j:"<-" json:"-"`
}

type IsOwner struct {
	neogo.Relationship `neo4j:"IS_OWNER"`
	Active             bool    `json:"active"`
	Owner              *Person `neo4j:"head" json:"-"`
	Cat                *Cat    `neo4j:"tail" json:"-"`
}

type Friend struct {
	neogo.Relationship `neo4j:"FRIEND"`
	Head               *Person `neo4j:"head" json:"-"`
	Tail               *Person `neo4j:"tail" json:"-"`
}
