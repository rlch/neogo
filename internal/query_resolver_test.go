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
func qs(field, name string, props ...string) querySelector {
	if len(props) == 1 && props[0] == "" {
		props = []string{}
	} else if len(props) == 0 {
		props = nil
	}
	return querySelector{
		field: field,
		name:  name,
		props: props,
	}
}

func TestResolveQuery(t *testing.T) {
	tx := NewTestTx()
	tx.driver.Registry().RegisterNode(&Owner{})
	var (
		regOwner      = tx.driver.Registry().Get(reflect.TypeOf(&Person{})).(*internal.RegisteredNode)
		regIsOwner    = tx.driver.Registry().Get(reflect.TypeOf(&IsOwner{})).(*internal.RegisteredRelationship)
		regCat        = tx.driver.Registry().Get(reflect.TypeOf(&Cat{})).(*internal.RegisteredNode)
		regBestFriend = tx.driver.Registry().GetByName("BEST_FRIEND").(*internal.RegisteredRelationship)
	)
	for _, test := range []struct {
		name      string
		root      any
		query     string
		expect    *nodeSelection
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
			expect: &nodeSelection{
				alloc:         &Owner{},
				querySelector: qs(".", ""),
				reg:           regOwner,
			},
		},
		{
			name:  "simple query with props and name",
			root:  &Owner{},
			query: "n{name}:.",
			expect: &nodeSelection{
				alloc:         &Owner{},
				querySelector: qs(".", "n", "name"),
				reg:           regOwner,
			},
		},
		{
			name:  "relationship with explicit node selector",
			root:  &Owner{},
			query: ". o:Cats c{}:.",
			expect: &nodeSelection{
				alloc:         &Owner{},
				querySelector: qs(".", ""),
				reg:           regOwner,
				next: &relationshipSelection{
					alloc:         ([]*IsOwner)(nil),
					querySelector: qs("Cats", "o"),
					reg:           regIsOwner,
					next: &nodeSelection{
						alloc:         ([]*Cat)(nil),
						querySelector: qs(".", "c", ""),
						reg:           regCat,
					},
				},
			},
		},
		{
			name:  "relationship with implicit node selector",
			root:  &Owner{},
			query: ". Cats",
			expect: &nodeSelection{
				querySelector: qs(".", ""),
				reg:           regOwner,
				alloc:         &Owner{},
				next: &relationshipSelection{
					querySelector: qs("Cats", ""),
					reg:           regIsOwner,
					alloc:         ([]*IsOwner)(nil),
					next: &nodeSelection{
						querySelector: qs(".", ""),
						reg:           regCat,
						alloc:         ([]*Cat)(nil),
					},
				},
			},
		},
		{
			name:  "shorthand relationship selection",
			root:  &Owner{},
			query: "BestFriend",
			expect: &nodeSelection{
				querySelector: qs(".", "", ""),
				reg:           regOwner,
				alloc:         &Owner{},
				next: &relationshipSelection{
					querySelector: qs("BestFriend", ""),
					reg:           regBestFriend,
					next: &nodeSelection{
						querySelector: qs(".", ""),
						reg:           regOwner,
						alloc:         &Owner{},
					},
				},
			},
		},
		{
			name:  "multiple relationships with implicit node selector",
			root:  &Owner{},
			query: "BestFriend . Cats Cat Owner",
			expect: &nodeSelection{
				querySelector: qs(".", "", ""),
				reg:           regOwner,
				alloc:         &Owner{},
				next: &relationshipSelection{
					querySelector: qs("BestFriend", ""),
					reg:           regBestFriend,
					next: &nodeSelection{
						querySelector: qs(".", ""),
						reg:           regOwner,
						alloc:         &Owner{},
						next: &relationshipSelection{
							querySelector: qs("Cats", ""),
							reg:           regIsOwner,
							alloc:         ([]*IsOwner)(nil),
							next: &nodeSelection{
								querySelector: qs("Cat", ""),
								reg:           regCat,
								alloc:         ([]*Cat)(nil),
								next: &relationshipSelection{
									querySelector: qs("Owner", ""),
									reg:           regIsOwner,
									alloc:         ([]*IsOwner)(nil),
									next: &nodeSelection{
										querySelector: qs(".", ""),
										reg:           regOwner,
										alloc:         ([]*Owner)(nil),
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
			expect: &nodeSelection{
				querySelector: qs(".", "", ""),
				reg:           regOwner,
				alloc:         []*Owner{},
				next: &relationshipSelection{
					querySelector: qs("Cats", ""),
					reg:           regIsOwner,
					alloc:         ([][]*IsOwner)(nil),
					next: &nodeSelection{
						querySelector: qs(".", ""),
						reg:           regCat,
						alloc:         ([][]*Cat)(nil),
						next: &relationshipSelection{
							querySelector: qs("Owner", ""),
							reg:           regIsOwner,
							alloc:         ([][]*IsOwner)(nil),
							next: &nodeSelection{
								querySelector: qs(".", ""),
								reg:           regOwner,
								alloc:         ([][]*Owner)(nil),
								next: &relationshipSelection{
									querySelector: qs("Cats", ""),
									reg:           regIsOwner,
									alloc:         ([][][]*IsOwner)(nil),
									next: &nodeSelection{
										querySelector: qs(".", ""),
										reg:           regCat,
										alloc:         ([][][]*Cat)(nil),
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
			selectors, err := newQueryParser(test.query).parse()
			require.NoError(err)
			selection, err := tx.resolveQuery(
				test.root,
				querySpec{selectors: selectors},
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
