package db

import (
	"github.com/rlch/neogo/query"
	"github.com/rlch/neogo/internal"
)

type (
	// A pattern is an expression of a Neo4J [pattern], which may be started
	// using [Node], and extended using To(), From() and Related().
	//
	//  db.Node(Person{}).
	//   To(nil, Movie{}).
	//   Related(ActedIn{}, nil).
	//   From(Knows{}, Person{})
	//
	//  // MATCH (:Person)-->(:Movie)-[:ACTED_IN]-()<--(:Person)
	//
	// [pattern]: https://neo4j.com/docs/cypher-manual/current/patterns/
	Pattern = internal.Pattern
)

// Node creates a [node pattern].
//
// [node pattern]: https://neo4j.com/docs/cypher-manual/current/patterns/concepts/#node-patterns
func Node(identifier query.Identifier) Pattern {
	return internal.NewNode(identifier)
}

// Path creates a [path pattern], qualified by name.
//
//	db.Path(db.Node(Person{}).Related(nil, Person{}), "p")
//
//	// p = (:Person)-->(:Person)
//
// [path pattern]: https://neo4j.com/docs/cypher-manual/current/patterns/concepts/#path-patterns
func Path(path Pattern, name string) Pattern {
	return internal.NewPath(path, name)
}

// Patterns is used to create multiple [Pattern]'s to be used in a single query.
//
//	Match(
//	 db.Patterns(
//	 	db.Node(db.Qual(
//	 		&martin,
//	 		"martin",
//	 		db.Props{
//	 			"name": "'Martin Sheen'",
//	 		},
//	 	)),
//	 	db.Node(db.Qual(
//	 		&rob,
//	 		"rob",
//	 		db.Props{
//	 			"name": "'Rob Reiner'",
//	 		},
//	 	)),
//	 ),
//	)
//
//	// MATCH
//	//	(martin:Person {name: 'Martin Sheen'}),
//	//	(rob:Person {name: 'Rob Reiner'})
func Patterns(paths ...Pattern) internal.Patterns {
	return internal.Paths(paths...)
}
