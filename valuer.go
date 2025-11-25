package neogo

import (
	"github.com/neo4j/neo4j-go-driver/v5/neo4j"
	"github.com/rlch/neogo/internal"
)

// Valuer allows arbitrary types to be marshalled into and unmarshalled from
// Neo4J data types. This allows any type (as oppposed to stdlib types, [INode],
// [IAbstract], [IRelationship], and structs with neo4j tags) to be used with
// [neogo]. The valid Neo4J data types are defined by [neo4j.RecordValue].
//
// For example, here we define a custom type MyString that marshals to and
// from a string, one of the types in the [neo4j.RecordValue] union:
//
//	type MyString string
//
//	var _ Valuer[string] = (*MyString)(nil)
//
//	func (s MyString) Marshal() (*string, error) {
//		return func(s string) *string {
//			return &s
//		}(string(s)), nil
//	}
//
//	func (s *MyString) Unmarshal(v *string) error {
//		*s = MyString(*v)
//		return nil
//	}
type Valuer[V neo4j.RecordValue] internal.Valuer[V]
