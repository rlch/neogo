package tests

import (
	"reflect"
	"testing"

	"github.com/rlch/neogo/db"
	"github.com/rlch/neogo/internal"
)

func TestMerge(t *testing.T) {
	t.Run("Merge nodes", func(t *testing.T) {
		t.Run("Merge single node with a label", func(t *testing.T) {
			var (
				robert any
				labels []string
			)
			c := internal.NewCypherClient(r)
			cy, err := c.
				Merge(db.Node(db.Qual(&robert, "robert", db.Label("Critic")))).
				Return(&robert, db.Qual(&labels, "labels(robert)")).
				Compile()

			Check(t, cy, err, internal.CompiledCypher{
				Cypher: `
					MERGE (robert:Critic)
					RETURN robert, labels(robert)
					`,
				Bindings: map[string]reflect.Value{
					"robert":         reflect.ValueOf(&robert),
					"labels(robert)": reflect.ValueOf(&labels),
				},
			})
		})

		t.Run("Merge single node with properties", func(t *testing.T) {
			var charlie any
			c := internal.NewCypherClient(r)
			cy, err := c.
				Merge(db.Node(db.Qual(&charlie, "charlie", db.Props{
					"name": "'Charlie Sheen'",
					"age":  "10",
				}))).
				Return(&charlie).
				Compile()

			Check(t, cy, err, internal.CompiledCypher{
				Cypher: `
					MERGE (charlie {age: 10, name: 'Charlie Sheen'})
					RETURN charlie
					`,
				Bindings: map[string]reflect.Value{
					"charlie": reflect.ValueOf(&charlie),
				},
			})
		})

		t.Run("Merge single node specifying both label and property", func(t *testing.T) {
			var michael Person
			c := internal.NewCypherClient(r)
			cy, err := c.
				Merge(db.Node(db.Qual(&michael, "michael", db.Props{
					"name": "'Michael Douglas'",
				}))).
				Return(&michael.Name, &michael.BornIn).
				Compile()

			Check(t, cy, err, internal.CompiledCypher{
				Cypher: `
					MERGE (michael:Person {name: 'Michael Douglas'})
					RETURN michael.name, michael.bornIn
					`,
				Bindings: map[string]reflect.Value{
					"michael.name":   reflect.ValueOf(&michael.Name),
					"michael.bornIn": reflect.ValueOf(&michael.BornIn),
				},
			})
		})

		t.Run("Merge single node derived from an existing node property", func(t *testing.T) {
			var (
				person   Person
				location Location
			)
			c := internal.NewCypherClient(r)
			cy, err := c.
				Match(db.Node(db.Qual(&person, "person"))).
				Merge(db.Node(db.Qual(&location, "location", db.Props{
					"name": "person.bornIn",
				}))).
				Return(&person.Name, &person.BornIn, &location).
				Compile()

			Check(t, cy, err, internal.CompiledCypher{
				Cypher: `
					MATCH (person:Person)
					MERGE (location:Location {name: person.bornIn})
					RETURN person.name, person.bornIn, location
					`,
				Bindings: map[string]reflect.Value{
					"person.name":   reflect.ValueOf(&person.Name),
					"person.bornIn": reflect.ValueOf(&person.BornIn),
					"location":      reflect.ValueOf(&location),
				},
			})
		})
	})

	t.Run("Use ON CREATE and ON MATCH", func(t *testing.T) {
		t.Run("Merge with ON CREATE", func(t *testing.T) {
			var keanu Person
			c := internal.NewCypherClient(r)
			cy, err := c.
				Merge(
					db.Node(
						db.Qual(&keanu, "keanu", db.Props{
							"bornIn":        "'Beirut'",
							"chauffeurName": "'Eric Brown'",
							"name":          "'Keanu Reeves'",
						}),
					),
					db.OnCreate(db.SetPropValue(&keanu.Created, "timestamp()")),
				).
				Return(&keanu.Name, &keanu.Created).
				Compile()

			Check(t, cy, err, internal.CompiledCypher{
				Cypher: `
					MERGE (keanu:Person {bornIn: 'Beirut', chauffeurName: 'Eric Brown', name: 'Keanu Reeves'})
					ON CREATE
					  SET keanu.created = timestamp()
					RETURN keanu.name, keanu.created
					`,
				Bindings: map[string]reflect.Value{
					"keanu.name":    reflect.ValueOf(&keanu.Name),
					"keanu.created": reflect.ValueOf(&keanu.Created),
				},
			})
		})

		t.Run("Merge with ON MATCH", func(t *testing.T) {
			var person Person
			c := internal.NewCypherClient(r)
			cy, err := c.
				Merge(
					db.Node(db.Qual(&person, "person")),
					db.OnMatch(db.SetPropValue(&person.Found, true)),
				).
				Return(&person.Name, &person.Found).
				Compile()

			Check(t, cy, err, internal.CompiledCypher{
				Cypher: `
					MERGE (person:Person)
					ON MATCH
					  SET person.found = true
					RETURN person.name, person.found
					`,
				Bindings: map[string]reflect.Value{
					"person.name":  reflect.ValueOf(&person.Name),
					"person.found": reflect.ValueOf(&person.Found),
				},
			})
		})

		t.Run("Merge with ON CREATE and ON MATCH", func(t *testing.T) {
			var keanu Person
			c := internal.NewCypherClient(r)
			cy, err := c.
				Merge(
					db.Node(db.Qual(&keanu, "keanu", db.Props{
						"name": "'Keanu Reeves'",
					})),
					db.OnCreate(db.SetPropValue(&keanu.Created, "timestamp()")),
					db.OnMatch(db.SetPropValue(&keanu.LastSeen, "timestamp()")),
				).
				Return(&keanu.Name, &keanu.Created, &keanu.LastSeen).
				Compile()

			Check(t, cy, err, internal.CompiledCypher{
				Cypher: `
					MERGE (keanu:Person {name: 'Keanu Reeves'})
					ON CREATE
					  SET keanu.created = timestamp()
					ON MATCH
					  SET keanu.lastSeen = timestamp()
					RETURN keanu.name, keanu.created, keanu.lastSeen
					`,
				Bindings: map[string]reflect.Value{
					"keanu.name":     reflect.ValueOf(&keanu.Name),
					"keanu.created":  reflect.ValueOf(&keanu.Created),
					"keanu.lastSeen": reflect.ValueOf(&keanu.LastSeen),
				},
			})
		})

		t.Run("Merge with ON MATCH setting multiple properties", func(t *testing.T) {
			var person Person
			c := internal.NewCypherClient(r)
			cy, err := c.
				Merge(
					db.Node(db.Qual(&person, "person")),
					db.OnMatch(
						db.SetPropValue(&person.Found, true),
						db.SetPropValue(&person.LastSeen, "timestamp()"),
					),
				).
				Return(&person.Name, &person.Found, &person.LastSeen).
				Compile()

			Check(t, cy, err, internal.CompiledCypher{
				Cypher: `
					MERGE (person:Person)
					ON MATCH
					  SET
					    person.found = true,
					    person.lastSeen = timestamp()
					RETURN person.name, person.found, person.lastSeen
					`,
				Bindings: map[string]reflect.Value{
					"person.name":     reflect.ValueOf(&person.Name),
					"person.found":    reflect.ValueOf(&person.Found),
					"person.lastSeen": reflect.ValueOf(&person.LastSeen),
				},
			})
		})
	})

	t.Run("Merge relationships", func(t *testing.T) {
		t.Run("Merge on a relationship", func(t *testing.T) {
			var (
				charlie    Person
				wallStreet Movie
				typeR      string
			)
			c := internal.NewCypherClient(r)
			cy, err := c.
				Match(
					db.Patterns(
						db.Node(db.Qual(&charlie, "charlie", db.Props{
							"name": "'Charlie Sheen'",
						})),
						db.Node(db.Qual(&wallStreet, "wallStreet", db.Props{
							"title": "'Wall Street'",
						})),
					),
				).
				Merge(db.Node(&charlie).To(db.Qual(ActedIn{}, "r"), &wallStreet)).
				Return(&charlie.Name, db.Qual(&typeR, "type(r)"), &wallStreet.Title).
				Compile()

			Check(t, cy, err, internal.CompiledCypher{
				Cypher: `
					MATCH
					  (charlie:Person {name: 'Charlie Sheen'}),
					  (wallStreet:Movie {title: 'Wall Street'})
					MERGE (charlie)-[r:ACTED_IN]->(wallStreet)
					RETURN charlie.name, type(r), wallStreet.title
					`,
				Bindings: map[string]reflect.Value{
					"charlie.name":     reflect.ValueOf(&charlie.Name),
					"type(r)":          reflect.ValueOf(&typeR),
					"wallStreet.title": reflect.ValueOf(&wallStreet.Title),
				},
			})
		})

		t.Run("Merge on multiple relationships", func(t *testing.T) {
			var (
				oliver Person
				reiner Person
				movie  Movie
			)
			c := internal.NewCypherClient(r)
			cy, err := c.
				Match(
					db.Patterns(
						db.Node(db.Qual(&oliver, "oliver", db.Props{
							"name": "'Oliver Stone'",
						})),
						db.Node(db.Qual(&reiner, "reiner", db.Props{
							"name": "'Rob Reiner'",
						})),
					),
				).
				Merge(
					db.Node(&oliver).
						To(Directed{}, &movie).
						From(Directed{}, &reiner),
				).
				Return(&movie).
				Compile()

			Check(t, cy, err, internal.CompiledCypher{
				Cypher: `
					MATCH
					  (oliver:Person {name: 'Oliver Stone'}),
					  (reiner:Person {name: 'Rob Reiner'})
					MERGE (oliver)-[:DIRECTED]->(movie:Movie)<-[:DIRECTED]-(reiner)
					RETURN movie
					`,
				Bindings: map[string]reflect.Value{
					"movie": reflect.ValueOf(&movie),
				},
			})
		})

		t.Run("Merge on an undirected relationship", func(t *testing.T) {
			var (
				charlie Person
				oliver  Person
				knows   Knows
			)
			c := internal.NewCypherClient(r)
			cy, err := c.
				Match(
					db.Patterns(
						db.Node(db.Qual(&charlie, "charlie", db.Props{
							"name": "'Charlie Sheen'",
						})),
						db.Node(db.Qual(&oliver, "oliver", db.Props{
							"name": "'Oliver Stone'",
						})),
					),
				).
				Merge(
					db.Node(&charlie).
						Related(db.Qual(&knows, "r"), &oliver),
				).
				Return(&knows).
				Compile()

			Check(t, cy, err, internal.CompiledCypher{
				Cypher: `
					MATCH
					  (charlie:Person {name: 'Charlie Sheen'}),
					  (oliver:Person {name: 'Oliver Stone'})
					MERGE (charlie)-[r:KNOWS]-(oliver)
					RETURN r
					`,
				Bindings: map[string]reflect.Value{
					"r": reflect.ValueOf(&knows),
				},
			})
		})

		t.Run("Merge on a relationship between two existing nodes", func(t *testing.T) {
			var (
				person   Person
				location Location
			)
			c := internal.NewCypherClient(r)
			cy, err := c.
				Match(db.Node(db.Qual(&person, "person"))).
				Merge(
					db.Node(db.Qual(&location, "location", db.Props{
						"name": "person.bornIn",
					})),
				).
				Merge(
					db.Node(&person).To(db.Qual(BornIn{}, "r"), &location),
				).
				Return(&person.Name, &person.BornIn, &location).
				Compile()

			Check(t, cy, err, internal.CompiledCypher{
				Cypher: `
					MATCH (person:Person)
					MERGE (location:Location {name: person.bornIn})
					MERGE (person)-[r:BORN_IN]->(location)
					RETURN person.name, person.bornIn, location
					`,
				Bindings: map[string]reflect.Value{
					"person.name":   reflect.ValueOf(&person.Name),
					"person.bornIn": reflect.ValueOf(&person.BornIn),
					"location":      reflect.ValueOf(&location),
				},
			})
		})

		t.Run("Merge on a relationship between an existing node and a merged node derived from a node property", func(t *testing.T) {
			type Chaffeur struct {
				internal.Node `neo4j:"Chauffeur"`
			}
			type HasChauffeur struct {
				internal.Relationship `neo4j:"HAS_CHAUFFEUR"`
			}
			var (
				person       Person
				hasChauffeur HasChauffeur
				chauffeur    Chaffeur
			)
			c := internal.NewCypherClient(r)
			cy, err := c.
				Match(db.Node(db.Qual(&person, "person"))).
				Merge(
					db.Node(&person).To(
						db.Qual(&hasChauffeur, "r"),
						db.Qual(&chauffeur, "chauffeur", db.Props{
							"name": &person.ChauffeurName,
						}),
					),
				).
				Return(&person.Name, &person.ChauffeurName, &chauffeur).
				Compile()

			Check(t, cy, err, internal.CompiledCypher{
				Cypher: `
					MATCH (person:Person)
					MERGE (person)-[r:HAS_CHAUFFEUR]->(chauffeur:Chauffeur {name: person.chauffeurName})
					RETURN person.name, person.chauffeurName, chauffeur
					`,
				Bindings: map[string]reflect.Value{
					"person.name":          reflect.ValueOf(&person.Name),
					"person.chauffeurName": reflect.ValueOf(&person.ChauffeurName),
					"chauffeur":            reflect.ValueOf(&chauffeur),
				},
			})
		})
	})

	t.Run("Using node property uniqueness constraints with MERGE", func(t *testing.T) {
		// TODO:
	})

	t.Run("Using relationship property uniqueness constraints with MERGE", func(t *testing.T) {
		// TODO:
	})
}
