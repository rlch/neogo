package internal

import (
	"reflect"
	"testing"

	"github.com/stretchr/testify/require"
)

type Person struct {
	Node `neo4j:"Person"`
	Name string `json:"name"`
}

func TestBindFields(t *testing.T) {
	t.Run("binds composite fields", func(t *testing.T) {
		s := newScope()
		p := &Person{}
		s.bindFields(reflect.ValueOf(p).Elem(), "p")
		require.Equal(t, map[uintptr]field{
			reflect.ValueOf(&p.ID).Pointer(): {
				identifier: "p",
				name:       "id",
			},
			reflect.ValueOf(&p.Name).Pointer(): {
				identifier: "p",
				name:       "name",
			},
		}, s.fields)
		require.Equal(t, map[reflect.Value]string{
			reflect.ValueOf(&p.ID):   "p.id",
			reflect.ValueOf(&p.Name): "p.name",
		}, s.names)
	})
}
