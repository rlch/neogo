package db

import "github.com/rlch/neogo/internal"

var Optional internal.MatchOption = &internal.Configurer{
	MatchOptions: func(v *internal.MatchOptions) {
		v.Optional = true
	},
}
