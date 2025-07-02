package internal

import (
	"reflect"
	"testing"

	"github.com/rlch/neogo/db"
	"github.com/stretchr/testify/require"
)

type (
	Topic struct {
		Node      `neo4j:"Topic"`
		Subtopics Many[*Subtopic] `neo4j:"CONTAINS>" json:"-"`
	}
	Subtopic struct {
		Node   `neo4j:"Subtopic"`
		Title  string
		Skills Many[*Skill] `neo4j:"CONTAINS>" json:"-"`
		Topics Many[*Topic] `neo4j:"<CONTAINS" json:"-"`
	}
	Skill struct {
		Node      `neo4j:"Skill"`
		Active    bool
		Subtopics Many[*Subtopic] `neo4j:"<CONTAINS" json:"-"`
	}
)

func main() {
	b := newCypherClient(nil)
	var topic Topic
	topic.
		Subtopics.Set(&Subtopic{Title: "asdf"}).
		Skills.Set(&Skill{Active: true})
	b.Create(db.Query(topic, ". Subtopics . Skills ."))

	// topic.Subtopics.V.Skills.V.Active = true
	// b = Find(
	// 	b,
	// 	&topic,
	// 	"t:. Subtopics s:Skills",
	// 	db.OrderBy("s.order"),
	// 	db.Where("t.active = ?", true),
	// )
	match := b.Match(
		db.Query(&topic, "t:. s:Subtopics v:Skills"),
	)
	b = CollectQuery(match)
	// skills := topic.Subtopics.V.Skills.S
	// MATCH (t:Topic)-[:CONTAINS]->(:Subtopic {title: "asdf"})-[:CONTAINS]->(s:Skill {active: true})
	// WITH t, s, collect(v) AS v
	// WITH t, collect(s) AS s, v
}

func Find(q *CypherClient, identifier any, queryString string) *CypherQuerier {
	return nil
}

type Owner struct {
	Node       `neo4j:"Owner"`
	Name       string         `json:"name"`
	Cats       Many[*IsOwner] `neo4j:"->" json:"-"`
	BestFriend *Owner         `neo4j:"BEST_FRIEND>" json:"-"`
	Friends    Many[*Owner]   `neo4j:"FRIENDS_WITH>" json:"-"`
}

type Cat struct {
	Node  `neo4j:"Cat"`
	Name  string   `json:"name"`
	Owner *IsOwner `neo4j:"<-" json:"-"`
}

type IsOwner struct {
	Relationship `neo4j:"IS_OWNER"`
	ID           string `json:"id"`
	Active       bool   `json:"active"`
	Owner        *Owner `neo4j:"startNode" json:"-"`
	Cat          *Cat   `neo4j:"endNode" json:"-"`
}

// qs returns a simple querySelector using the provided field, name and props.
func qs(field, name string, props ...string) QuerySelector {
	if len(props) == 1 && props[0] == "" {
		props = []string{}
	} else if len(props) == 0 {
		props = nil
	}
	return QuerySelector{
		Field: field,
		Name:  name,
		Props: props,
	}
}

func TestResolveQuery(t *testing.T) {
	r := NewRegistry()
	r.RegisterNode(&Owner{})

	var (
		regOwner      = r.Get(reflect.TypeOf(&Owner{})).(*RegisteredNode)
		regIsOwner    = r.Get(reflect.TypeOf(&IsOwner{})).(*RegisteredRelationship)
		regCat        = r.Get(reflect.TypeOf(&Cat{})).(*RegisteredNode)
		regBestFriend = r.GetByName("BEST_FRIEND").(*RegisteredRelationship)
	)
	ownerWithProps := &Owner{}
	ownerWithProps.
		Cats.Set(&IsOwner{Active: true, Cat: &Cat{Name: "Alice", Owner: &IsOwner{Owner: &Owner{}}}}).Cat.Owner.Owner.
		Cats.Set(&IsOwner{Cat: &Cat{Name: "Whiskers"}})

	for _, test := range []struct {
		name      string
		root      any
		query     string
		expect    *NodeSelection
		expectErr string
	}{
		{
			name:      "fails if no root provided",
			expectErr: "no target node/relationship provided",
		},
		{
			name:      "fails when relationship used as root",
			root:      &IsOwner{},
			expectErr: "cannot use relationship as root",
		},
		{
			name:      "fails when no selectors provided",
			root:      &Owner{},
			expectErr: "no selectors provided",
		},
		{
			name:      "fails when an invalid relationship is used in selector",
			root:      &Owner{},
			query:     ". Catz .",
			expectErr: "relationship Catz not found in node Owner",
		},
		{
			name:      "fails when an invalid node is used in selector",
			root:      &Owner{},
			query:     ". Cats Gat",
			expectErr: "field Gat not found in relationship IsOwner, expected Cat or '.'",
		},
		{
			name:      "fails when an invalid node prop is referenced",
			root:      &Owner{},
			query:     "{prop}:.",
			expectErr: "property prop not found in node Owner",
		},
		{
			name:      "fails when an invalid relationship prop is referenced",
			root:      &Owner{},
			query:     ". {prop}:Cats",
			expectErr: "property prop not found in relationship IsOwner",
		},
		{
			name:  "simple query with one selector",
			root:  &Owner{},
			query: ".",
			expect: &NodeSelection{
				Alloc:         &Owner{},
				QuerySelector: qs(".", ""),
				Target: &NodeTarget{
					RegisteredNode: regOwner,
				},
			},
		},
		{
			name:  "simple query with props and name",
			root:  &Owner{},
			query: "n{name}:.",
			expect: &NodeSelection{
				Alloc:         &Owner{},
				QuerySelector: qs(".", "n", "name"),
				Target: &NodeTarget{
					RegisteredNode: regOwner,
				},
			},
		},
		{
			name:  "relationship with explicit node selector",
			root:  &Owner{},
			query: ". o:Cats c{}:.",
			expect: &NodeSelection{
				Alloc:         &Owner{},
				QuerySelector: qs(".", ""),
				Target: &NodeTarget{
					RegisteredNode: regOwner,
				},
				Next: &RelationshipSelection{
					Alloc:         ([]*IsOwner)(nil),
					QuerySelector: qs("Cats", "o"),
					Target: &RelationshipTarget{
						Many: true,
						Dir:  true,
						Rel:  regIsOwner,
					},
					Next: &NodeSelection{
						Alloc:         ([]*Cat)(nil),
						QuerySelector: qs(".", "c", ""),
						Target: &NodeTarget{
							Field:          "Cat",
							RegisteredNode: regCat,
						},
					},
				},
			},
		},
		{
			name:  "relationship with implicit node selector",
			root:  &Owner{},
			query: ". Cats",
			expect: &NodeSelection{
				QuerySelector: qs(".", ""),
				Target: &NodeTarget{
					RegisteredNode: regOwner,
				},
				Alloc: &Owner{},
				Next: &RelationshipSelection{
					QuerySelector: qs("Cats", ""),
					Target: &RelationshipTarget{
						Many: true,
						Dir:  true,
						Rel:  regIsOwner,
					},
					Alloc: new(IsOwner),
					Next: &NodeSelection{
						QuerySelector: qs(".", ""),
						Target: &NodeTarget{
							Field:          "Cat",
							RegisteredNode: regCat,
						},
						Alloc: new(Cat),
					},
				},
			},
		},
		{
			name:  "shorthand relationship selection",
			root:  &Owner{},
			query: "BestFriend",
			expect: &NodeSelection{
				QuerySelector: qs(".", "", ""),
				Target: &NodeTarget{
					RegisteredNode: regOwner,
				},
				Alloc: &Owner{},
				Next: &RelationshipSelection{
					QuerySelector: qs("BestFriend", ""),
					Target: &RelationshipTarget{
						Dir:  true,
						Rel:  regBestFriend,
						Many: false,
					},
					Next: &NodeSelection{
						QuerySelector: qs(".", ""),
						Target: &NodeTarget{
							RegisteredNode: regOwner,
						},
						Alloc: &Owner{},
					},
				},
			},
		},
		{
			name:  "multiple relationships with implicit node selector",
			root:  &Owner{},
			query: "BestFriend . Cats c:Cat Owner",
			expect: &NodeSelection{
				QuerySelector: qs(".", "", ""),
				Target: &NodeTarget{
					RegisteredNode: regOwner,
				},
				Alloc: &Owner{},
				Next: &RelationshipSelection{
					QuerySelector: qs("BestFriend", ""),
					Target: &RelationshipTarget{
						Dir: true,
						Rel: regBestFriend,
					},
					Next: &NodeSelection{
						QuerySelector: qs(".", ""),
						Target: &NodeTarget{
							RegisteredNode: regOwner,
						},
						Alloc: &Owner{},
						Next: &RelationshipSelection{
							QuerySelector: qs("Cats", ""),
							Target: &RelationshipTarget{
								Many: true,
								Dir:  true,
								Rel:  regIsOwner,
							},
							Alloc: ([]*IsOwner)(nil),
							Next: &NodeSelection{
								QuerySelector: qs("Cat", "c"),
								Target: &NodeTarget{
									Field:          "Cat",
									RegisteredNode: regCat,
								},
								Alloc: ([]*Cat)(nil),
								Next: &RelationshipSelection{
									QuerySelector: qs("Owner", ""),
									Target: &RelationshipTarget{
										Dir: false,
										Rel: regIsOwner,
									},
									Alloc: ([]*IsOwner)(nil),
									Next: &NodeSelection{
										QuerySelector: qs(".", ""),
										Target: &NodeTarget{
											Field:          "Owner",
											RegisteredNode: regOwner,
										},
										Alloc: ([]*Owner)(nil),
									},
								},
							},
						},
					},
				},
			},
		},
		{
			name:  "> 1 layer of allocation depth",
			root:  []*Owner{},
			query: "c:Cats . Owner . Cats cc:.",
			expect: &NodeSelection{
				QuerySelector: qs(".", "", ""),
				Target: &NodeTarget{
					RegisteredNode: regOwner,
				},
				Alloc: []*Owner{},
				Next: &RelationshipSelection{
					QuerySelector: qs("Cats", "c"),
					Target: &RelationshipTarget{
						Many: true,
						Dir:  true,
						Rel:  regIsOwner,
					},
					Alloc: ([][]*IsOwner)(nil),
					Next: &NodeSelection{
						QuerySelector: qs(".", ""),
						Target: &NodeTarget{
							Field:          "Cat",
							RegisteredNode: regCat,
						},
						Alloc: ([][]*Cat)(nil),
						Next: &RelationshipSelection{
							QuerySelector: qs("Owner", ""),
							Target: &RelationshipTarget{
								Dir: false,
								Rel: regIsOwner,
							},
							Alloc: ([][]*IsOwner)(nil),
							Next: &NodeSelection{
								QuerySelector: qs(".", ""),
								Target: &NodeTarget{
									Field:          "Owner",
									RegisteredNode: regOwner,
								},
								Alloc: ([][]*Owner)(nil),
								Next: &RelationshipSelection{
									QuerySelector: qs("Cats", ""),
									Target: &RelationshipTarget{
										Many: true,
										Dir:  true,
										Rel:  regIsOwner,
									},
									Alloc: ([][][]*IsOwner)(nil),
									Next: &NodeSelection{
										QuerySelector: qs(".", "cc"),
										Target: &NodeTarget{
											Field:          "Cat",
											RegisteredNode: regCat,
										},
										Alloc: ([][][]*Cat)(nil),
									},
								},
							},
						},
					},
				},
			},
		},
		{
			name:  "payload set in relationships",
			root:  ownerWithProps,
			query: ". c:Cats . Owner . Cats cc:.",
			expect: &NodeSelection{
				QuerySelector: qs(".", ""),
				Target:        &NodeTarget{RegisteredNode: regOwner},
				Alloc:         ownerWithProps,
				Next: &RelationshipSelection{
					QuerySelector: qs("Cats", "c"),
					Target: &RelationshipTarget{
						Many: true,
						Dir:  true,
						Rel:  regIsOwner,
					},
					Alloc:   ([]*IsOwner)(nil),
					Payload: ownerWithProps.Cats.V,
					Next: &NodeSelection{
						QuerySelector: qs(".", ""),
						Target: &NodeTarget{
							Field:          "Cat",
							RegisteredNode: regCat,
						},
						Alloc:   ([]*Cat)(nil),
						Payload: ownerWithProps.Cats.V.Cat,
						Next: &RelationshipSelection{
							QuerySelector: qs("Owner", ""),
							Target: &RelationshipTarget{
								Dir: false,
								Rel: regIsOwner,
							},
							Alloc:   ([]*IsOwner)(nil),
							Payload: ownerWithProps.Cats.V.Cat.Owner,
							Next: &NodeSelection{
								QuerySelector: qs(".", ""),
								Target: &NodeTarget{
									Field:          "Owner",
									RegisteredNode: regOwner,
								},
								Alloc:   ([]*Owner)(nil),
								Payload: ownerWithProps.Cats.V.Cat.Owner.Owner,
								Next: &RelationshipSelection{
									QuerySelector: qs("Cats", ""),
									Target: &RelationshipTarget{
										Many: true,
										Dir:  true,
										Rel:  regIsOwner,
									},
									Alloc:   ([][]*IsOwner)(nil),
									Payload: ownerWithProps.Cats.V.Cat.Owner.Owner.Cats.V,
									Next: &NodeSelection{
										QuerySelector: qs(".", "cc"),
										Target: &NodeTarget{
											Field:          "Cat",
											RegisteredNode: regCat,
										},
										Payload: &Cat{Name: "Whiskers"},
										Alloc:   ([][]*Cat)(nil),
									},
								},
							},
						},
					},
				},
			},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			require := require.New(t)
			selection, err := ResolveQuery(
				r,
				test.root,
				test.query,
			)
			if test.expectErr != "" {
				require.ErrorContains(err, test.expectErr)
			} else if err != nil {
				require.NoError(err)
			}
			require.Equal(test.expect, selection)
		})
	}
}
