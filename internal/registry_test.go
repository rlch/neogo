package internal

import (
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
		Name   string `neo4j:"name"`
		Ignore string `neo4j:"-"`
	}

	simpleNode struct {
		Node `neo4j:"Simple"`
	}
	nodeWithRelationship struct {
		Node `neo4j:"Simple"`

		Forward   *simpleRelationship   `neo4j:"->"`
		Backward  *simpleRelationship   `neo4j:"<-"`
		Forwards  []*simpleRelationship `neo4j:"->"`
		Backwards []*simpleRelationship `neo4j:"<-"`
	}
	simpleRelationship struct {
		Relationship `neo4j:"SIMPLE"`

		Field     string                `neo4j:"field"`
		StartNode *nodeWithRelationship `neo4j:"startNode"`
		EndNode   *nodeWithRelationship `neo4j:"endNode"`
	}
	shorthandRelationshipNode struct {
		simpleNode
		Forward   *simpleNode       `neo4j:"->"`
		Backward  *simpleNode       `neo4j:"<-"`
		Forwards  Many[*simpleNode] `neo4j:"->"`
		Backwards Many[*simpleNode] `neo4j:"<-"`
	}
)

// Test expectations (comparing behavior via methods)
type nodeExpectation struct {
	name          string
	labels        []string
	fieldsToProps map[string]string
	relCount      int
}

func TestRegisterNode(t *testing.T) {
	for _, test := range []struct {
		name    string
		node    INode
		want    nodeExpectation
		wantErr string
	}{
		{
			name: "registers a simple node",
			node: &simpleNode{},
			want: nodeExpectation{
				name:          "simpleNode",
				labels:        []string{"Simple"},
				fieldsToProps: map[string]string{"ID": "id"},
				relCount:      0,
			},
		},
		{
			name: "registers a node with nested labels",
			node: &nestedLabelsNode{},
			want: nodeExpectation{
				name:          "nestedLabelsNode",
				labels:        []string{"Simple", "Nested"},
				fieldsToProps: map[string]string{"ID": "id"},
				relCount:      0,
			},
		},
		{
			name: "registers a node with nested labels using Label type",
			node: &nestedLabelsUsingLabelNode{},
			want: nodeExpectation{
				name:          "nestedLabelsUsingLabelNode",
				labels:        []string{"Simple", "Nested", "Label"},
				fieldsToProps: map[string]string{"ID": "id"},
				relCount:      0,
			},
		},
		{
			name: "registers a node with properties",
			node: &nodeWithProperties{},
			want: nodeExpectation{
				name:          "nodeWithProperties",
				labels:        []string{"Simple"},
				fieldsToProps: map[string]string{"ID": "id", "Name": "name"},
				relCount:      0,
			},
		},
		{
			name: "registers a node with relationships",
			node: &nodeWithRelationship{},
			want: nodeExpectation{
				name:          "nodeWithRelationship",
				labels:        []string{"Simple"},
				fieldsToProps: map[string]string{"ID": "id"},
				relCount:      4, // Forward, Backward, Forwards, Backwards
			},
		},
		{
			name: "registers a node with shorthand relationship syntax",
			node: &shorthandRelationshipNode{},
			want: nodeExpectation{
				name:          "shorthandRelationshipNode",
				labels:        []string{"Simple"},
				fieldsToProps: map[string]string{"ID": "id"},
				relCount:      4, // Forward, Backward, Forwards, Backwards
			},
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
				return
			}
			if err != nil {
				t.Fatalf("expected no error, got: %s", err)
			}

			// Compare via methods (delegation)
			require.Equal(test.want.name, reg.Name())
			require.Equal(test.want.labels, reg.Labels())
			require.Equal(test.want.fieldsToProps, reg.FieldsToProps())
			if test.want.relCount == 0 {
				require.Nil(reg.Relationships)
			} else {
				require.Len(reg.Relationships, test.want.relCount)
			}
		})
	}
}

func TestRegisterNodeWithRelationships(t *testing.T) {
	require := require.New(t)
	r := NewRegistry()
	reg := r.RegisterNode(&nodeWithRelationship{})

	// Verify relationships
	require.NotNil(reg.Relationships)
	require.Len(reg.Relationships, 4)

	// Forward relationship
	fwd := reg.Relationships["Forward"]
	require.NotNil(fwd)
	require.True(fwd.Dir)
	require.False(fwd.Many)
	require.Equal("SIMPLE", fwd.Rel.Reltype)

	// Backward relationship
	bwd := reg.Relationships["Backward"]
	require.NotNil(bwd)
	require.False(bwd.Dir)
	require.False(bwd.Many)

	// Many forward relationships
	fwds := reg.Relationships["Forwards"]
	require.NotNil(fwds)
	require.True(fwds.Dir)
	require.True(fwds.Many)

	// Many backward relationships
	bwds := reg.Relationships["Backwards"]
	require.NotNil(bwds)
	require.False(bwds.Dir)
	require.True(bwds.Many)
}

func TestRegisterNodeShorthand(t *testing.T) {
	require := require.New(t)
	r := NewRegistry()
	reg := r.RegisterNode(&shorthandRelationshipNode{})

	// Verify shorthand relationships
	require.NotNil(reg.Relationships)
	require.Len(reg.Relationships, 4)

	// Forward shorthand
	fwd := reg.Relationships["Forward"]
	require.NotNil(fwd)
	require.True(fwd.Dir)
	require.False(fwd.Many)
	require.Equal("SHORTHAND", fwd.Rel.Reltype)

	// Backward shorthand
	bwd := reg.Relationships["Backward"]
	require.NotNil(bwd)
	require.False(bwd.Dir)
	require.False(bwd.Many)
}

func TestGet(t *testing.T) {
	r := NewRegistry()
	r.RegisterTypes(&BaseOrganism{}, &ActedIn{})

	t.Run("gets a node", func(t *testing.T) {
		require := require.New(t)
		got := r.Get(reflect.TypeOf(Human{}))
		require.NotNil(got)

		reg := got.(*RegisteredNode)
		require.Equal("Human", reg.Name())
		require.Equal([]string{"Organism", "Human"}, reg.Labels())
		require.Equal(map[string]string{
			"ID":    "id",
			"Alive": "alive",
			"Name":  "name",
		}, reg.FieldsToProps())
		require.Equal(reflect.TypeOf(Human{}), reg.Type())
	})

	t.Run("gets a pointer to node", func(t *testing.T) {
		require := require.New(t)
		got := r.Get(reflect.TypeOf(&Human{}))
		require.NotNil(got)

		reg := got.(*RegisteredNode)
		require.Equal("Human", reg.Name())
		require.Equal([]string{"Organism", "Human"}, reg.Labels())
		require.Equal(reflect.TypeOf(Human{}), reg.Type())
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
