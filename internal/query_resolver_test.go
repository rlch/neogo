package internal

import (
	"reflect"
	"testing"

	"github.com/stretchr/testify/require"
)

type Owner struct {
	Node       `neo4j:"Owner"`
	Name       string     `json:"name"`
	Cats       []*IsOwner `neo4j:"->" json:"-"`
	BestFriend *Owner     `neo4j:"BEST_FRIEND>" json:"-"`
	Friends    []*Owner   `neo4j:"FRIENDS_WITH>" json:"-"`
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
		field: field,
		name:  name,
		props: props,
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
				alloc:         &Owner{},
				QuerySelector: qs(".", ""),
				target: &NodeTarget{
					RegisteredNode: regOwner,
				},
			},
		},
		{
			name:  "simple query with props and name",
			root:  &Owner{},
			query: "n{name}:.",
			expect: &NodeSelection{
				alloc:         &Owner{},
				QuerySelector: qs(".", "n", "name"),
				target: &NodeTarget{
					RegisteredNode: regOwner,
				},
			},
		},
		{
			name:  "relationship with explicit node selector",
			root:  &Owner{},
			query: ". o:Cats c{}:.",
			expect: &NodeSelection{
				alloc:         &Owner{},
				QuerySelector: qs(".", ""),
				target: &NodeTarget{
					RegisteredNode: regOwner,
				},
				next: &RelationshipSelection{
					alloc:         ([]*IsOwner)(nil),
					QuerySelector: qs("Cats", "o"),
					target: &RelationshipTarget{
						Many: true,
						Dir:  true,
						Rel:  regIsOwner,
					},
					next: &NodeSelection{
						alloc:         ([]*Cat)(nil),
						QuerySelector: qs(".", "c", ""),
						target: &NodeTarget{
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
				target: &NodeTarget{
					RegisteredNode: regOwner,
				},
				alloc: &Owner{},
				next: &RelationshipSelection{
					QuerySelector: qs("Cats", ""),
					target: &RelationshipTarget{
						Many: true,
						Dir:  true,
						Rel:  regIsOwner,
					},
					alloc: ([]*IsOwner)(nil),
					next: &NodeSelection{
						QuerySelector: qs(".", ""),
						target: &NodeTarget{
							Field:          "Cat",
							RegisteredNode: regCat,
						},
						alloc: ([]*Cat)(nil),
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
				target: &NodeTarget{
					RegisteredNode: regOwner,
				},
				alloc: &Owner{},
				next: &RelationshipSelection{
					QuerySelector: qs("BestFriend", ""),
					target: &RelationshipTarget{
						Dir:  true,
						Rel:  regBestFriend,
						Many: false,
					},
					next: &NodeSelection{
						QuerySelector: qs(".", ""),
						target: &NodeTarget{
							RegisteredNode: regOwner,
						},
						alloc: &Owner{},
					},
				},
			},
		},
		{
			name:  "multiple relationships with implicit node selector",
			root:  &Owner{},
			query: "BestFriend . Cats Cat Owner",
			expect: &NodeSelection{
				QuerySelector: qs(".", "", ""),
				target: &NodeTarget{
					RegisteredNode: regOwner,
				},
				alloc: &Owner{},
				next: &RelationshipSelection{
					QuerySelector: qs("BestFriend", ""),
					target: &RelationshipTarget{
						Dir: true,
						Rel: regBestFriend,
					},
					next: &NodeSelection{
						QuerySelector: qs(".", ""),
						target: &NodeTarget{
							RegisteredNode: regOwner,
						},
						alloc: &Owner{},
						next: &RelationshipSelection{
							QuerySelector: qs("Cats", ""),
							target: &RelationshipTarget{
								Many: true,
								Dir:  true,
								Rel:  regIsOwner,
							},
							alloc: ([]*IsOwner)(nil),
							next: &NodeSelection{
								QuerySelector: qs("Cat", ""),
								target: &NodeTarget{
									Field:          "Cat",
									RegisteredNode: regCat,
								},
								alloc: ([]*Cat)(nil),
								next: &RelationshipSelection{
									QuerySelector: qs("Owner", ""),
									target: &RelationshipTarget{
										Dir: false,
										Rel: regIsOwner,
									},
									alloc: ([]*IsOwner)(nil),
									next: &NodeSelection{
										QuerySelector: qs(".", ""),
										target: &NodeTarget{
											Field:          "Owner",
											RegisteredNode: regOwner,
										},
										alloc: ([]*Owner)(nil),
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
			query: "Cats . Owner . Cats",
			expect: &NodeSelection{
				QuerySelector: qs(".", "", ""),
				target: &NodeTarget{
					RegisteredNode: regOwner,
				},
				alloc: []*Owner{},
				next: &RelationshipSelection{
					QuerySelector: qs("Cats", ""),
					target: &RelationshipTarget{
						Many: true,
						Dir:  true,
						Rel:  regIsOwner,
					},
					alloc: ([][]*IsOwner)(nil),
					next: &NodeSelection{
						QuerySelector: qs(".", ""),
						target: &NodeTarget{
							Field:          "Cat",
							RegisteredNode: regCat,
						},
						alloc: ([][]*Cat)(nil),
						next: &RelationshipSelection{
							QuerySelector: qs("Owner", ""),
							target: &RelationshipTarget{
								Dir: false,
								Rel: regIsOwner,
							},
							alloc: ([][]*IsOwner)(nil),
							next: &NodeSelection{
								QuerySelector: qs(".", ""),
								target: &NodeTarget{
									Field:          "Owner",
									RegisteredNode: regOwner,
								},
								alloc: ([][]*Owner)(nil),
								next: &RelationshipSelection{
									QuerySelector: qs("Cats", ""),
									target: &RelationshipTarget{
										Many: true,
										Dir:  true,
										Rel:  regIsOwner,
									},
									alloc: ([][][]*IsOwner)(nil),
									next: &NodeSelection{
										QuerySelector: qs(".", ""),
										target: &NodeTarget{
											Field:          "Cat",
											RegisteredNode: regCat,
										},
										alloc: ([][][]*Cat)(nil),
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
