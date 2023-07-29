package db

import "github.com/rlch/neogo/internal"

func Node(match any) internal.Pattern {
	return internal.Node(match)
}

func Path(path internal.Pattern, name string) internal.Pattern {
	return internal.NewPath(path, name)
}

func Patterns(paths ...internal.Pattern) internal.Patterns {
	return internal.Paths(paths...)
}
