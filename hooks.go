package neogo

import "reflect"

// MarshalHook runs after a struct parameter is serialized to map[string]any
// but before the map is sent to Neo4j. It receives the parameter key name,
// the original struct value, and the serialized map for modification.
type MarshalHook func(key string, original reflect.Value, serialized map[string]any) error

// UnmarshalHook runs after values are unmarshalled from Neo4j results.
// `from` is the most specific raw source that produced the current bound value:
// the root hook may receive a full neo4j.Node or neo4j.Relationship, while
// nested struct fields and slice elements receive their corresponding child raw
// values (typically maps or indexed elements).
// `to` is the deserialized struct value.
type UnmarshalHook func(from any, to reflect.Value) error
