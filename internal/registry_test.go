package internal

import (
	"maps"
	"reflect"
	"testing"

	"github.com/stretchr/testify/require"
)

func panicToErr[T any](f func() T) (out T, err error) {
	defer func() {
		if r := recover(); r != nil {
			err = r.(error)
		}
	}()
	out = f()
	return
}

type (
	simpleNode struct {
		Node `neo4j:"Simple"`
	}
	nestedLabelsNode struct {
		simpleNode `neo4j:"Nested"`
	}
	labelNode struct {
		Label `neo4j:"Label"`
	}
	nestedLabelsUsingLabelNode struct {
		simpleNode `neo4j:"Nested"`
		labelNode
	}
	nodeWithProperties struct {
		simpleNode
		Name   string `json:"name"`
		Ignore string `json:"-"`
	}
	nodeWithRelationship struct {
		simpleNode
		Forward   *simpleRelationship       `json:"-" neo4j:"->"`
		Backward  *simpleRelationship       `json:"-" neo4j:"<-"`
		Forwards  Many[*simpleRelationship] `json:"-" neo4j:"->"`
		Backwards Many[*simpleRelationship] `json:"-" neo4j:"<-"`
	}
	simpleRelationship struct {
		Relationship `neo4j:"SIMPLE"`
		Field        string                `json:"field"`
		StartNode    *nodeWithRelationship `json:"-" neo4j:"startNode"`
		EndNode      *nodeWithRelationship `json:"-" neo4j:"endNode"`
	}
	shorthandRelationshipNode struct {
		simpleNode
		Forward   *simpleNode       `json:"-" neo4j:"SHORTHAND>"`
		Backward  *simpleNode       `json:"-" neo4j:"<SHORTHAND"`
		Forwards  Many[*simpleNode] `json:"-" neo4j:"SHORTHAND>"`
		Backwards Many[*simpleNode] `json:"-" neo4j:"<SHORTHAND"`
	}
)

var (
	simpleNodeReg = &RegisteredNode{
		rType:         reflect.TypeOf(simpleNode{}),
		name:          "simpleNode",
		Labels:        []string{"Simple"},
		fieldsToProps: map[string]string{"ID": "id"},
	}
	nestedLabelsNodeReg = &RegisteredNode{
		rType:         reflect.TypeOf(nestedLabelsNode{}),
		name:          "nestedLabelsNode",
		Labels:        []string{"Simple", "Nested"},
		fieldsToProps: map[string]string{"ID": "id"},
	}
	nestedLabelsUsingLabelNodeReg = &RegisteredNode{
		rType:         reflect.TypeOf(nestedLabelsUsingLabelNode{}),
		name:          "nestedLabelsUsingLabelNode",
		Labels:        []string{"Simple", "Nested", "Label"},
		fieldsToProps: map[string]string{"ID": "id"},
	}
	nodeWithPropertiesReg = &RegisteredNode{
		rType:         reflect.TypeOf(nodeWithProperties{}),
		name:          "nodeWithProperties",
		Labels:        []string{"Simple"},
		fieldsToProps: map[string]string{"ID": "id", "Name": "name"},
	}
	simpleRelationshipReg = &RegisteredRelationship{
		rType:         reflect.TypeOf(simpleRelationship{}),
		name:          "simpleRelationship",
		Reltype:       "SIMPLE",
		fieldsToProps: map[string]string{"Field": "field"},
	}
	nodeWithRelationshipReg = &RegisteredNode{
		rType:         reflect.TypeOf(nodeWithRelationship{}),
		name:          "nodeWithRelationship",
		Labels:        []string{"Simple"},
		fieldsToProps: map[string]string{"ID": "id"},
		Relationships: map[string]*RelationshipTarget{
			"Forward": {
				Dir: true,
				Rel: simpleRelationshipReg,
			},
			"Backward": {
				Dir: false,
				Rel: simpleRelationshipReg,
			},
			"Forwards": {
				Dir:  true,
				Rel:  simpleRelationshipReg,
				Many: true,
			},
			"Backwards": {
				Dir:  false,
				Rel:  simpleRelationshipReg,
				Many: true,
			},
		},
	}
	shorthandRelationshipReg = func(dir bool) *RegisteredRelationship {
		r := &RegisteredRelationship{
			Reltype: "SHORTHAND",
		}
		if dir {
			r.EndNode.RegisteredNode = shorthandRelationshipNodeReg
			r.StartNode.RegisteredNode = simpleNodeReg
		} else {
			r.EndNode.RegisteredNode = simpleNodeReg
			r.StartNode.RegisteredNode = shorthandRelationshipNodeReg
		}
		return r
	}
	shorthandRelationshipNodeReg *RegisteredNode = new(RegisteredNode)
)

func init() {
	simpleRelationshipReg.StartNode = NodeTarget{
		Field:          "StartNode",
		RegisteredNode: nodeWithRelationshipReg,
	}
	simpleRelationshipReg.EndNode = NodeTarget{
		Field:          "EndNode",
		RegisteredNode: nodeWithRelationshipReg,
	}

	maps.Copy(nodeWithRelationshipReg.Relationships, nodeWithRelationshipReg.Relationships)

	*shorthandRelationshipNodeReg = RegisteredNode{
		rType:         reflect.TypeOf(shorthandRelationshipNode{}),
		name:          "shorthandRelationshipNode",
		Labels:        []string{"Simple"},
		fieldsToProps: map[string]string{"ID": "id"},
		Relationships: map[string]*RelationshipTarget{
			"Forward": {
				Dir: true,
				Rel: shorthandRelationshipReg(true),
			},
			"Backward": {
				Dir: false,
				Rel: shorthandRelationshipReg(false),
			},
			"Forwards": {
				Dir:  true,
				Rel:  shorthandRelationshipReg(true),
				Many: true,
			},
			"Backwards": {
				Dir:  false,
				Rel:  shorthandRelationshipReg(false),
				Many: true,
			},
		},
	}

	maps.Copy(shorthandRelationshipNodeReg.Relationships, shorthandRelationshipNodeReg.Relationships)
}

func TestRegisterNode(t *testing.T) {
	for _, test := range []struct {
		name    string
		node    INode
		want    *RegisteredNode
		wantErr string
	}{
		{
			name: "registers a simple node",
			node: &simpleNode{},
			want: simpleNodeReg,
		},
		{
			name: "registers a node with nested labels",
			node: &nestedLabelsNode{},
			want: nestedLabelsNodeReg,
		},
		{
			name: "registers a node with nested labels using Label type",
			node: &nestedLabelsUsingLabelNode{},
			want: nestedLabelsUsingLabelNodeReg,
		},
		{
			name: "registers a node with properties",
			node: &nodeWithProperties{},
			want: nodeWithPropertiesReg,
		},
		{
			name: "registers a node with relationships",
			node: &nodeWithRelationship{},
			want: nodeWithRelationshipReg,
		},
		{
			name: "registers a node with shorthand relationship syntax",
			node: &shorthandRelationshipNode{},
			want: shorthandRelationshipNodeReg,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			require := require.New(t)
			r := NewRegistry()
			reg, err := panicToErr(
				func() *RegisteredNode { return r.RegisterNode(test.node) },
			)
			if test.wantErr != "" {
				require.ErrorContains(err, test.wantErr)
			} else if err != nil {
				t.Fatalf("expected no error, got: %s", err)
			}
			require.Equal(test.want, reg)
		})
	}
}

func TestGet(t *testing.T) {
	r := NewRegistry()
	r.RegisterTypes(&BaseOrganism{}, &ActedIn{})
	t.Run("gets a node", func(t *testing.T) {
		require := require.New(t)
		got := r.Get(reflect.TypeOf(Human{}))
		require.Equal(
			&RegisteredNode{
				name:   "Human",
				rType:  reflect.TypeOf(Human{}),
				Labels: []string{"Organism", "Human"},
				fieldsToProps: map[string]string{
					"ID":    "id",
					"Alive": "alive",
					"Name":  "name",
				},
			},
			got,
		)
	})
	t.Run("gets a pointer to node", func(t *testing.T) {
		require := require.New(t)
		got := r.Get(reflect.TypeOf(&Human{}))
		require.Equal(
			&RegisteredNode{
				name:   "Human",
				rType:  reflect.TypeOf(Human{}),
				Labels: []string{"Organism", "Human"},
				fieldsToProps: map[string]string{
					"ID":    "id",
					"Alive": "alive",
					"Name":  "name",
				},
			},
			got,
		)
	})
}

func TestGetConcreteImplementation(t *testing.T) {
	t.Run("error when no abstract node found for labels", func(t *testing.T) {
		require := require.New(t)
		r := NewRegistry()
		impl, err := r.GetConcreteImplementation([]string{"Human", "Organism"})
		require.Nil(impl)
		require.Error(err)
	})

	t.Run("error when no concrete implementation found that satisfies labels", func(t *testing.T) {
		type Alien struct {
			Abstract `neo4j:"Organism"`
			Node     `neo4j:"Alien"`
		}
		require := require.New(t)
		r := NewRegistry()
		r.RegisterTypes(&Alien{})
		impl, err := r.GetConcreteImplementation([]string{"Human", "Organism"})
		require.Nil(impl)
		require.Error(err)
	})

	t.Run("finds base type that satisfies labels", func(t *testing.T) {
		require := require.New(t)
		r := NewRegistry()
		r.RegisterTypes(&BaseOrganism{})
		impl, err := r.GetConcreteImplementation([]string{"Organism"})
		require.NoError(err)
		require.Equal(reflect.TypeOf(BaseOrganism{}), impl.Type())
	})

	t.Run("finds concrete implementation that satisfies labels", func(t *testing.T) {
		require := require.New(t)
		r := NewRegistry()
		r.RegisterTypes(&BaseOrganism{})
		impl, err := r.GetConcreteImplementation([]string{"Human", "Organism"})
		require.NoError(err)
		require.Equal(reflect.TypeOf(Human{}), impl.Type())
	})
}
