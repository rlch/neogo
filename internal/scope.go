package internal

import (
	"errors"
	"fmt"
	"reflect"
	"runtime/debug"
	"strconv"
	"strings"

	"github.com/goccy/go-json"
	"github.com/iancoleman/strcase"

	neogo "github.com/rlch/neogo"
)

func newScope() *scope {
	return &scope{
		bindings:   make(map[string]reflect.Value),
		names:      make(map[reflect.Value]string),
		fields:     make(map[uintptr]struct{ name, entity string }),
		parameters: map[string]any{},
		paramAddrs: map[uintptr]string{},
	}
}

type (
	scope struct {
		err error

		bindings map[string]reflect.Value
		names    map[reflect.Value]string
		fields   map[uintptr]struct{ name, entity string }

		paramCounter     int
		parameters       map[string]any
		parameterFilters map[string]*json.FieldQuery
		paramAddrs       map[uintptr]string
	}
	// An instance of a node/relationship in the cypher query
	member struct {
		// The entity that was registered
		entity any
		// Whether the entity was added to the scope by the query that returned this
		// member.
		isNew bool
		// The name of the variable in the cypher query
		name  string
		alias string
		// The name of the property in the cypher query
		props string

		variable *Variable

		// The where clause that this member is associated with.
		where *Where

		// The projection body that this member is associated with.
		projectionBody *ProjectionBody
	}
)

var (
	nodeType         = reflect.TypeOf((*neogo.INode)(nil)).Elem()
	relationshipType = reflect.TypeOf((*neogo.IRelationship)(nil)).Elem()
)

func (s *scope) catch(op func()) {
	defer func() {
		if r := recover(); r != nil {
			debug.PrintStack()
			err, ok := r.(error)
			if !ok {
				panic(err)
			}
			if s.err == nil {
				s.err = err
			}
		}
	}()
	op()
}

func (s *scope) unfoldEntity(value any) (
	entity any,
	variable *Variable,
	projBody *ProjectionBody,
) {
	entity = value
	// Preference outer overrides
	// Could use mergo.Merge, but it's not worth the dependency
	mergeV := func(v *Variable) {
		if variable == nil {
			variable = v
			return
		}
		if variable.Bind == nil {
			variable.Bind = v.Bind
		}
		if variable.Name == "" {
			variable.Name = v.Name
		}
		if variable.Expr == "" {
			variable.Expr = v.Expr
		}
		if variable.Where == nil {
			variable.Where = v.Where
		}
		if variable.Props == nil {
			variable.Props = v.Props
		}
		if variable.Pattern == "" {
			variable.Pattern = v.Pattern
		}
		if variable.Quantifier == "" {
			variable.Quantifier = v.Quantifier
		}
		if variable.Select == nil {
			variable.Select = v.Select
		}
	}
RecurseToEntity:
	for {
		switch v := entity.(type) {
		case *ProjectionBody:
			projBody = v
			entity = v.Entity
		case ProjectionBody:
			projBody = &v
			entity = v.Entity
		case Variable:
			mergeV(&v)
			entity = v.Entity
		case *Variable:
			mergeV(v)
			entity = v.Entity
		default:
			break RecurseToEntity
		}
	}
	return entity, variable, projBody
}

func (s *scope) replaceBinding(m *member) {
	v := reflect.ValueOf(m.entity)
	vT := v.Type()
	canElem := vT.Kind() == reflect.Ptr ||
		vT.Kind() == reflect.Slice ||
		vT.Kind() == reflect.Array
	if m.variable != nil && m.variable.Bind != nil {
		bind := reflect.ValueOf(m.variable.Bind)
		if bind.Kind() != reflect.Ptr {
			panic(fmt.Errorf("cannot bind to non-pointer value %s", bind))
		}
		name := m.alias
		if name == "" {
			name = m.name
		}
		s.bindings[name] = bind
		s.names[bind] = name
	} else if m.alias != "" && m.alias != m.name {
		s.names[v] = m.alias
		delete(s.bindings, m.name)
		if canElem {
			s.bindings[m.alias] = v
		}
	} else if m.name != "" {
		s.names[v] = m.name
		if canElem {
			s.bindings[m.name] = v
		}
	}
}

func (s *scope) register(value any, isNode *bool) *member {
	if value == nil {
		return nil
	}

	m := &member{isNew: true}
	entity, variable, projBody := s.unfoldEntity(value)

	// Propagate information from Variable to member
	m.entity = entity
	if variable != nil {
		variable.Entity = entity
		m.variable = variable
		if variable.Expr != "" {
			m.name = string(variable.Expr)
			if variable.Name != "" {
				m.alias = variable.Name
			}
		} else if variable.Name != "" {
			m.name = variable.Name
		}
		if variable.Where != nil {
			m.where = variable.Where
		}
	}
	if projBody != nil {
		projBody.Entity = entity
		m.projectionBody = projBody
	}
	if entity == nil {
		return m
	}

	v := reflect.ValueOf(entity)
	vT := v.Type()
	canElem := vT.Kind() == reflect.Ptr ||
		vT.Kind() == reflect.Slice ||
		vT.Kind() == reflect.Array

	// Find the name of the entity
	if m.name != "" {
		if exst, ok := s.bindings[m.name]; ok && exst != v {
			panic(fmt.Errorf("(%s) already bound to different value. want: %s, have: %s", m.name, v, s.bindings[m.name]))
		} else if ok {
			m.isNew = false
			currentName := s.names[exst]
			// Check if name needs to be replaced
			if currentName != "" && currentName != m.name {
				m.alias = m.name
				m.name = currentName
			}
		} else if !ok {
			if canElem {
				// Check if name needs to be replaced
				if oldName, ok := s.names[v]; ok {
					m.alias = m.name
					m.name = oldName
				}
			}
		}
	} else if canElem {
		if name, ok := s.names[v]; ok {
			m.isNew = false
			m.name = name
		}
		needsName := m.name == "" && (projBody != nil || m.where != nil || v.Kind() == reflect.Ptr)
		if needsName {
			var prefix string
			if vT.Implements(nodeType) {
				prefix = strcase.ToLowerCamel(extractNodeLabel(entity)[0])
			} else if vT.Implements(relationshipType) {
				prefix = strcase.ToLowerCamel(extractRelationshipType(entity))
			} else {
				prefix = strcase.ToLowerCamel(vT.Elem().Name())
				if prefix == "" {
					prefix = strcase.ToLowerCamel(vT.Elem().Kind().String())
				}
			}
			if _, ok := s.bindings[prefix]; !ok {
				m.name = prefix
			} else {
				var potentialName string
				i := 1
				for {
					potentialName = fmt.Sprintf("%s%d", prefix, i)
					if _, ok := s.bindings[potentialName]; !ok {
						break
					}
					i++
				}
				m.name = potentialName
			}
		}
	}
	if expr, ok := m.entity.(Expr); ok {
		// Allow strings to be used as names
		if m.name != "" {
			m.alias = m.name
		}
		m.name = string(expr)
	} else if name, ok := m.entity.(string); ok {
		// Allow strings to be used as names
		if m.name != "" {
			m.alias = m.name
		}
		m.name = name
	}

	s.replaceBinding(m)

	// Validate entity type
	canHaveProps := isNode != nil &&
		((*isNode && vT.Implements(nodeType)) ||
			(!*isNode && vT.Implements(relationshipType)))

	// Check if entity has data to inject as a parameter
	inner := v
	for inner.Kind() == reflect.Ptr {
		inner = inner.Elem()
	}
	if inner.IsValid() && m.isNew {
		switch inner.Kind() {
		case reflect.Struct, reflect.Array, reflect.Slice:
			if inner.IsZero() {
				break
			}
			param := s.addParameter(v, m.name)
			if m.variable != nil && m.variable.Select != nil {
				s.parameterFilters[param] = m.variable.Select
			}
			if canHaveProps {
				m.props = param
			} else {
				m.alias = m.name
				m.name = param
			}
		case reflect.Map:
			param := s.addParameter(v, m.name)
			m.alias = m.name
			m.name = param
		}
	}

	// Bind field addresses to their names
	if canElem && inner.Kind() == reflect.Struct {
		vsT := inner.Type()
		for i := 0; i < vsT.NumField(); i++ {
			vf := inner.Field(i)
			jsTag, ok := vsT.Field(i).Tag.Lookup("json")
			if !ok {
				continue
			}
			fieldName := strings.Split(jsTag, ",")[0]
			ptr := uintptr(vf.Addr().UnsafePointer())
			field := struct {
				name   string
				entity string
			}{
				name:   fieldName,
				entity: m.name,
			}
			s.fields[ptr] = field

			name := field.entity + "." + field.name
			vfAddr := vf.Addr()
			s.names[vfAddr] = name
		}
	}

	return m
}

func (s *scope) registerNode(n *node) *member {
	t := true
	return s.register(n.data, &t)
}

func (s *scope) registerEdge(n *relationship) *member {
	f := false
	return s.register(n.data, &f)
}

func (s *scope) entityName(entity any) string {
	entity, _, _ = s.unfoldEntity(entity)
	return s.names[reflect.ValueOf(entity)]
}

func (s *scope) propertyExpression(entity any) func(v any) string {
	entity, _, _ = s.unfoldEntity(entity)
	entityName := s.entityName(entity)
	return func(v any) string {
		if v == entity && entityName != "" {
			return entityName
		}
		if expr, ok := v.(Expr); ok {
			return string(expr)
		} else if str, strOk := v.(string); strOk && entityName != "" {
			// Consider strings as properties if entity is known
			return fmt.Sprintf("%s.%s", entityName, str)
		} else if strOk {
			// Otherwise, consider strings as literals
			return str
		}
		vv := reflect.ValueOf(v)
		if vv.Kind() != reflect.Ptr {
			panic(errors.New("the key in a condition must be addressable"))
		}
		ptr := vv.Pointer()
		if name, ok := s.names[vv]; ok {
			return name
		}
		if field, ok := s.fields[ptr]; ok {
			return fmt.Sprintf("%s.%s", field.entity, field.name)
		}
		panic(fmt.Errorf("could not find a property-representation for %v", v))
	}
}

func (s *scope) valueExpression(v any) string {
	vv := reflect.ValueOf(v)
	switch vv.Kind() {
	case reflect.Bool:
		if v.(bool) {
			return "true"
		} else {
			return "false"
		}
	case reflect.String:
		if expr, ok := v.(Expr); ok {
			return string(expr)
		}
		return v.(string)
	case reflect.Int, reflect.Int8, reflect.Int16,
		reflect.Int32, reflect.Int64, reflect.Uint,
		reflect.Uint8, reflect.Uint16, reflect.Uint32,
		reflect.Uint64, reflect.Float32, reflect.Float64,
		reflect.Array, reflect.Interface, reflect.Map,
		reflect.Slice, reflect.Struct:
		if param, ok := v.(Param); ok {
			return s.addParameter(reflect.ValueOf(*param.Value), param.Name)
		} else {
			return s.addParameter(vv, "")
		}
	case reflect.Pointer:
		ptr := vv.Pointer()
		if name, ok := s.names[vv]; ok {
			return name
		}
		if field, ok := s.fields[ptr]; ok {
			return fmt.Sprintf("%s.%s", field.entity, field.name)
		}
	default:
		panic(fmt.Errorf("unsupported value-type %T", v))
	}
	panic(fmt.Errorf("could not find a value-representation for %v", v))
}

func (s *scope) addParameter(v reflect.Value, optName string) (name string) {
	defer func() {
		s.parameters[name] = v.Interface()
		name = "$" + name
	}()
	if v.CanAddr() {
		addr := v.UnsafeAddr()
		if existing, ok := s.paramAddrs[addr]; ok {
			return existing
		}
		defer func() {
			s.paramAddrs[addr] = name
		}()
	}
	if optName != "" {
		return optName
	}
	s.paramCounter++
	return "v" + strconv.Itoa(s.paramCounter)
}
