package internal

import (
	"reflect"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestBindFields(t *testing.T) {
	t.Run("binds composite fields to fields map only", func(t *testing.T) {
		r := NewRegistry()
		s := newScope(r)
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
		// Fields are NOT added to names map to avoid conflict with struct pointer
		// (first field has same address as struct itself)
		require.Equal(t, map[uintptr]string{}, s.names)
	})
	// TODO: Need more tests
}
