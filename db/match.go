package db

import "github.com/rlch/neo4j-gorm/internal"

var Optional internal.MatchOption = &internal.Configurer{
	MatchOptions: func(v *internal.MatchOptions) {
		v.Optional = true
	},
}
