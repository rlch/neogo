package db

import "github.com/rlch/neogo/internal"

// Node creates a node pattern, which may be used in MATCH, CREATE and MERGE
// clauses.
func Node(entity any) internal.Pattern {
	return internal.NewNode(entity)
}

func Path(path internal.Pattern, name string) internal.Pattern {
	return internal.NewPath(path, name)
}

func Patterns(paths ...internal.Pattern) internal.Patterns {
	return internal.Paths(paths...)
}
