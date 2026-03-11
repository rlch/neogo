package neogo

import (
	"errors"
	"reflect"
	"testing"

	"github.com/neo4j/neo4j-go-driver/v5/neo4j"
	"github.com/stretchr/testify/require"
)

type hookPerson struct {
	Name string `json:"name"`
}

type hookWrapper struct {
	Person *hookPerson `json:"person"`
}

type hookIfaceWrapper struct {
	Item any
}

type hookNestedWrapper struct {
	Person hookPerson `json:"person"`
}

type hookCaseFoldWrapper struct {
	Person hookPerson
}

type hookPtrMarshalJSONPerson struct {
	Name string `json:"name"`
}

func (p *hookPtrMarshalJSONPerson) MarshalJSON() ([]byte, error) {
	return []byte(`{"name":"via-pointer-marshal"}`), nil
}

func setHookName(value reflect.Value, next string) bool {
	field := value.FieldByName("Name")
	if !field.IsValid() || !field.CanSet() || field.Kind() != reflect.String {
		return false
	}
	field.SetString(next)
	return true
}

func TestUnmarshalHook(t *testing.T) {
	var (
		called int
		r      registry
	)
	r.registerUnmarshalHook(func(from any, value reflect.Value) error {
		if setHookName(value, "hooked") {
			called++
		}
		return nil
	})

	person := hookPerson{}
	err := r.bindValue(neo4j.Node{Props: map[string]any{"name": "ignored"}}, reflect.ValueOf(&person))
	require.NoError(t, err)
	require.Equal(t, "hooked", person.Name)

	called = 0
	var people []hookPerson
	props := []any{
		map[string]any{"name": "one"},
		map[string]any{"name": "two"},
	}
	err = r.bindValue(props, reflect.ValueOf(&people))
	require.NoError(t, err)
	require.Len(t, people, 2)
	require.Equal(t, "hooked", people[0].Name)
	require.GreaterOrEqual(t, called, 2)

	called = 0
	var nested [][]hookPerson
	err = r.bindValue([][]any{props}, reflect.ValueOf(&nested))
	require.NoError(t, err)
	require.Len(t, nested, 1)
	require.Equal(t, "hooked", nested[0][0].Name)
	require.GreaterOrEqual(t, called, 2)
}

func TestUnmarshalHookEdgeCases(t *testing.T) {
	t.Run("propagates hook errors", func(t *testing.T) {
		var r registry
		expected := errors.New("boom")
		r.registerUnmarshalHook(func(from any, value reflect.Value) error {
			return expected
		})
		person := hookPerson{}
		err := r.bindValue(map[string]any{"name": "x"}, reflect.ValueOf(&person))
		require.ErrorIs(t, err, expected)
	})

	t.Run("handles nested pointers", func(t *testing.T) {
		var (
			called int
			r      registry
		)
		r.registerUnmarshalHook(func(from any, value reflect.Value) error {
			if setHookName(value, "nested") {
				called++
			}
			return nil
		})
		wrapper := hookWrapper{}
		err := r.bindValue(map[string]any{
			"person": map[string]any{"name": "x"},
		}, reflect.ValueOf(&wrapper))
		require.NoError(t, err)
		require.NotNil(t, wrapper.Person)
		require.Equal(t, "nested", wrapper.Person.Name)
		require.GreaterOrEqual(t, called, 1)
	})

	t.Run("handles interface values", func(t *testing.T) {
		var (
			called int
			r      registry
		)
		r.registerUnmarshalHook(func(from any, value reflect.Value) error {
			if setHookName(value, "iface") {
				called++
			}
			return nil
		})
		wrapper := hookIfaceWrapper{Item: &hookPerson{Name: "x"}}
		err := r.applyUnmarshalHooks(nil, reflect.ValueOf(&wrapper))
		require.NoError(t, err)
		require.Equal(t, "iface", wrapper.Item.(*hookPerson).Name)
		require.GreaterOrEqual(t, called, 1)
	})

	t.Run("applies multiple hooks in order", func(t *testing.T) {
		var r registry
		r.registerUnmarshalHook(func(from any, value reflect.Value) error {
			setHookName(value, "first")
			return nil
		})
		r.registerUnmarshalHook(func(from any, value reflect.Value) error {
			field := value.FieldByName("Name")
			if !field.IsValid() || !field.CanSet() || field.Kind() != reflect.String {
				return nil
			}
			field.SetString(field.String() + "-second")
			return nil
		})
		person := hookPerson{}
		err := r.bindValue(map[string]any{"name": "x"}, reflect.ValueOf(&person))
		require.NoError(t, err)
		require.Equal(t, "first-second", person.Name)
	})
}

func TestMarshalHook(t *testing.T) {
	t.Run("modifies serialized struct map", func(t *testing.T) {
		var called int
		var r registry
		r.registerMarshalHook(func(key string, original reflect.Value, serialized map[string]any) error {
			if _, ok := serialized["name"]; ok {
				serialized["name"] = "hooked"
				called++
			}
			return nil
		})
		result, err := r.canonicalizeParams(
			map[string]any{"props": hookPerson{Name: "raw"}},
		)
		require.NoError(t, err)
		props := result["props"].(map[string]any)
		require.Equal(t, "hooked", props["name"])
		require.Equal(t, 1, called)
	})

	t.Run("fires per element for slice of structs", func(t *testing.T) {
		var called int
		var r registry
		r.registerMarshalHook(func(key string, original reflect.Value, serialized map[string]any) error {
			if name, ok := serialized["name"]; ok {
				serialized["name"] = name.(string) + "-hooked"
				called++
			}
			return nil
		})
		people := []hookPerson{{Name: "Alice"}, {Name: "Bob"}}
		result, err := r.canonicalizeParams(
			map[string]any{"props": people},
		)
		require.NoError(t, err)
		props := result["props"].([]any)
		require.Len(t, props, 2)
		require.Equal(t, "Alice-hooked", props[0].(map[string]any)["name"])
		require.Equal(t, "Bob-hooked", props[1].(map[string]any)["name"])
		require.Equal(t, 2, called)
	})

	t.Run("propagates hook errors", func(t *testing.T) {
		expected := errors.New("hook failed")
		var r registry
		r.registerMarshalHook(func(key string, original reflect.Value, serialized map[string]any) error {
			return expected
		})
		_, err := r.canonicalizeParams(
			map[string]any{"props": hookPerson{Name: "test"}},
		)
		require.ErrorIs(t, err, expected)
	})

	t.Run("propagates hook errors for slice elements", func(t *testing.T) {
		expected := errors.New("slice hook failed")
		var r registry
		r.registerMarshalHook(func(key string, original reflect.Value, serialized map[string]any) error {
			return expected
		})
		_, err := r.canonicalizeParams(
			map[string]any{"props": []hookPerson{{Name: "test"}}},
		)
		require.ErrorIs(t, err, expected)
	})

	t.Run("receives param key name", func(t *testing.T) {
		var receivedKey string
		var r registry
		r.registerMarshalHook(func(key string, original reflect.Value, serialized map[string]any) error {
			receivedKey = key
			return nil
		})
		_, err := r.canonicalizeParams(
			map[string]any{"myParam": hookPerson{Name: "test"}},
		)
		require.NoError(t, err)
		require.Equal(t, "myParam", receivedKey)
	})

	t.Run("can read original struct fields including json-hidden", func(t *testing.T) {
		type hiddenField struct {
			Name   string `json:"name"`
			Secret string `json:"-"`
		}
		var r registry
		r.registerMarshalHook(func(key string, original reflect.Value, serialized map[string]any) error {
			if secret := original.FieldByName("Secret"); secret.IsValid() {
				serialized["secret_value"] = secret.String()
			}
			return nil
		})
		result, err := r.canonicalizeParams(
			map[string]any{"props": hiddenField{Name: "visible", Secret: "hidden"}},
		)
		require.NoError(t, err)
		props := result["props"].(map[string]any)
		require.Equal(t, "visible", props["name"])
		require.Equal(t, "hidden", props["secret_value"])
	})

	t.Run("slice of struct pointers should canonicalize nil elements to nil", func(t *testing.T) {
		var r registry
		r.registerMarshalHook(func(key string, original reflect.Value, serialized map[string]any) error {
			return nil
		})

		people := []*hookPerson{nil, {Name: "Alice"}}
		result, err := r.canonicalizeParams(
			map[string]any{"props": people},
		)
		require.NoError(t, err)
		props := result["props"].([]any)
		require.Len(t, props, 2)
		require.Equal(t, nil, props[0], "nil slice elements should stay plain nil, not typed nil pointers")
	})

	t.Run("slice of struct pointers should preserve pointer MarshalJSON behavior", func(t *testing.T) {
		var r registry
		r.registerMarshalHook(func(key string, original reflect.Value, serialized map[string]any) error {
			return nil
		})

		people := []*hookPtrMarshalJSONPerson{{Name: "raw"}}
		result, err := r.canonicalizeParams(
			map[string]any{"props": people},
		)
		require.NoError(t, err)
		props := result["props"].([]any)
		require.Len(t, props, 1)
		require.Equal(t, "via-pointer-marshal", props[0].(map[string]any)["name"])
	})
}

func TestUnmarshalHookRegressionCases(t *testing.T) {
	t.Run("logical object should only be hooked once during bind", func(t *testing.T) {
		var (
			called int
			r      registry
		)
		r.registerUnmarshalHook(func(from any, value reflect.Value) error {
			field := value.FieldByName("Name")
			if !field.IsValid() || !field.CanSet() || field.Kind() != reflect.String {
				return nil
			}
			field.SetString(field.String() + "!")
			called++
			return nil
		})

		person := hookPerson{}
		err := r.bindValue(neo4j.Node{Props: map[string]any{"name": "x"}}, reflect.ValueOf(&person))
		require.NoError(t, err)
		require.Equal(t, 1, called, "hook should run exactly once per logical object")
		require.Equal(t, "x!", person.Name)
	})

	t.Run("root hooks should retain original neo4j values", func(t *testing.T) {
		type relPayload struct {
			Count int `json:"count"`
		}

		var (
			gotNode         any
			gotRelationship any
			r               registry
		)
		r.registerUnmarshalHook(func(from any, value reflect.Value) error {
			switch value.Type() {
			case reflect.TypeOf(hookPerson{}):
				gotNode = from
			case reflect.TypeOf(relPayload{}):
				gotRelationship = from
			}
			return nil
		})

		person := hookPerson{}
		node := neo4j.Node{Labels: []string{"Person"}, Props: map[string]any{"name": "x"}}
		err := r.bindValue(node, reflect.ValueOf(&person))
		require.NoError(t, err)
		require.Equal(t, node, gotNode)

		rel := relPayload{}
		rawRel := neo4j.Relationship{Type: "KNOWS", Props: map[string]any{"count": int64(2)}}
		err = r.bindValue(rawRel, reflect.ValueOf(&rel))
		require.NoError(t, err)
		require.Equal(t, rawRel, gotRelationship)
	})

	t.Run("nested named struct fields should receive their raw source map", func(t *testing.T) {
		var (
			gotFrom any
			r       registry
		)
		r.registerUnmarshalHook(func(from any, value reflect.Value) error {
			if value.Type() == reflect.TypeOf(hookPerson{}) {
				gotFrom = from
			}
			return nil
		})

		wrapper := hookNestedWrapper{}
		err := r.bindValue(map[string]any{
			"person": map[string]any{"name": "nested"},
		}, reflect.ValueOf(&wrapper))
		require.NoError(t, err)
		require.Equal(t, map[string]any{"name": "nested"}, gotFrom)
	})

	t.Run("nested raw source lookup should allow case-insensitive field-name matching", func(t *testing.T) {
		var (
			gotFrom any
			r       registry
		)
		r.registerUnmarshalHook(func(from any, value reflect.Value) error {
			if value.Type() == reflect.TypeOf(hookPerson{}) {
				gotFrom = from
			}
			return nil
		})

		wrapper := hookCaseFoldWrapper{}
		err := r.bindValue(map[string]any{
			"person": map[string]any{"name": "casefold"},
		}, reflect.ValueOf(&wrapper))
		require.NoError(t, err)
		require.Equal(t, map[string]any{"name": "casefold"}, gotFrom)
	})

	t.Run("slice elements should receive their own raw source maps", func(t *testing.T) {
		var (
			gotFroms []any
			r        registry
		)
		r.registerUnmarshalHook(func(from any, value reflect.Value) error {
			if value.Type() == reflect.TypeOf(hookPerson{}) {
				gotFroms = append(gotFroms, from)
			}
			return nil
		})

		var people []hookPerson
		err := r.bindValue([]any{
			map[string]any{"name": "one"},
			map[string]any{"name": "two"},
		}, reflect.ValueOf(&people))
		require.NoError(t, err)
		require.Equal(t, []any{
			map[string]any{"name": "one"},
			map[string]any{"name": "two"},
		}, gotFroms)
	})

	t.Run("single non-slice source coerced into slice should preserve element raw source", func(t *testing.T) {
		var (
			gotFroms []any
			r        registry
		)
		r.registerUnmarshalHook(func(from any, value reflect.Value) error {
			if value.Type() == reflect.TypeOf(hookPerson{}) {
				gotFroms = append(gotFroms, from)
			}
			return nil
		})

		var people []hookPerson
		err := r.bindValue(neo4j.Node{Props: map[string]any{"name": "solo"}}, reflect.ValueOf(&people))
		require.NoError(t, err)
		require.Equal(t, []any{map[string]any{"name": "solo"}}, gotFroms)
		require.Len(t, people, 1)
		require.Equal(t, "solo", people[0].Name)
	})
}
