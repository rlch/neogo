package db

import (
	"github.com/rlch/neogo/client"
	"github.com/rlch/neogo/internal"
)

// SetPropValue sets a property to a value in a [SET] clause.
//
//	SET <identifier> = <value>
//
// [SET]: https://neo4j.com/docs/cypher-manual/current/clauses/set/
func SetPropValue(identifier client.PropertyIdentifier, value client.ValueIdentifier) internal.SetItem {
	return internal.SetItem{
		PropIdentifier: identifier,
		ValIdentifier:  value,
	}
}

// SetMerge merges a property to a value in a [SET] clause.
//
//	SET <identifier> += <properties>
//
// [SET]: https://neo4j.com/docs/cypher-manual/current/clauses/set/
func SetMerge(identifier client.PropertyIdentifier, properties any) internal.SetItem {
	return internal.SetItem{
		PropIdentifier: identifier,
		ValIdentifier:  properties,
		Merge:          true,
	}
}

// SetLabels sets labels in a [SET] clause.
//
//	SET <identifier>:<label>:...:<label>
//
// [SET]: https://neo4j.com/docs/cypher-manual/current/clauses/set/
func SetLabels(identifier client.PropertyIdentifier, labels ...string) internal.SetItem {
	return internal.SetItem{
		PropIdentifier: identifier,
		Labels:         labels,
	}
}

// RemoveProp removes a property in a [REMOVE] clause.
//
//	SET <identifier>.<prop>
//
// [REMOVE]: https://neo4j.com/docs/cypher-manual/current/clauses/remove/
func RemoveProp(identifier client.PropertyIdentifier) internal.RemoveItem {
	return internal.RemoveItem{
		PropIdentifier: identifier,
	}
}

// RemoveLabels removes labels in a [REMOVE] clause.
//
//	REMOVE <identifier>:<label>:...:<label>
//
// [REMOVE]: https://neo4j.com/docs/cypher-manual/current/clauses/remove/
func RemoveLabels(identifier client.PropertyIdentifier, labels ...string) internal.RemoveItem {
	return internal.RemoveItem{
		PropIdentifier: identifier,
		Labels:         labels,
	}
}

// OnCreate sets the actions to perform when a [MERGE] clause creates a node.
//
//	ON CREATE
//	 SET <...>
//	 ...
//
// [MERGE]: https://neo4j.com/docs/cypher-manual/current/clauses/merge/
func OnCreate(set ...internal.SetItem) internal.MergeOption {
	return &internal.Configurer{
		MergeOptions: func(mo *internal.Merge) {
			mo.OnCreate = append(mo.OnCreate, set...)
		},
	}
}

// OnMatch sets the actions to perform when a [MERGE] clause matches a node.
//
//	ON MATCH
//	 SET <...>
//	 ...
//
// [MERGE]: https://neo4j.com/docs/cypher-manual/current/clauses/merge/
func OnMatch(set ...internal.SetItem) internal.MergeOption {
	return &internal.Configurer{
		MergeOptions: func(mo *internal.Merge) {
			mo.OnMatch = append(mo.OnMatch, set...)
		},
	}
}
