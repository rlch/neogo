package internal

import (
	"errors"
	"fmt"
	"reflect"
	"strconv"
	"strings"

	"github.com/goccy/go-json"
	"github.com/iancoleman/strcase"
)

func newScope() *Scope {
	return &Scope{
		bindings:       make(map[string]reflect.Value),
		names:          make(map[reflect.Value]string),
		generatedNames: map[string]struct{}{},
		fields:         make(map[uintptr]struct{ name, entity string }),
		parameters:     map[string]any{},
		paramAddrs:     map[uintptr]string{},
	}
}

type (
	Scope struct {
		err error

		isWrite        bool
		bindings       map[string]reflect.Value
		generatedNames map[string]struct{}
		names          map[reflect.Value]string
		fields         map[uintptr]struct{ name, entity string }

		paramCounter int
		paramPrefix  string

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
	nodeType         = reflect.TypeOf((*INode)(nil)).Elem()
	relationshipType = reflect.TypeOf((*IRelationship)(nil)).Elem()
)

func (m *member) Print() {
	fmt.Printf(
`{
  entity: %+v,
  isNew: %v,
  name: %s,
  alias: %s,
  props: %s,
  variable: %+v,
  where: %+v,
  projection: %+v,
}` + "\n" , m.entity, m.isNew, m.name, m.alias, m.props, m.variable, m.where, m.projectionBody)
}

func (s *Scope) clone() *Scope {
	bindings := make(map[string]reflect.Value, len(s.bindings))
	for k, v := range s.bindings {
		bindings[k] = v
	}
	generatedNames := make(map[string]struct{}, len(s.generatedNames))
	for k, v := range s.generatedNames {
		generatedNames[k] = v
	}
	names := make(map[reflect.Value]string, len(s.names))
	for k, v := range s.names {
		names[k] = v
	}
	fields := make(map[uintptr]struct{ name, entity string }, len(s.fields))
	for k, v := range s.fields {
		fields[k] = v
	}
	paramCounter := s.paramCounter
	parameters := make(map[string]any, len(s.parameters))
	for k, v := range s.parameters {
		parameters[k] = v
	}
	parameterFilters := make(map[string]*json.FieldQuery, len(s.parameterFilters))
	for k, v := range s.parameterFilters {
		parameterFilters[k] = v
	}
	paramAddrs := make(map[uintptr]string, len(s.paramAddrs))
	for k, v := range s.paramAddrs {
		paramAddrs[k] = v
	}
	return &Scope{
		bindings:         bindings,
		generatedNames:   generatedNames,
		names:            names,
		fields:           fields,
		paramCounter:     paramCounter,
		parameters:       parameters,
		parameterFilters: parameterFilters,
		paramAddrs:       paramAddrs,
	}
}

func (child *Scope) mergeParentScope(parent *Scope) {
	// We merge the param counter for avoiding parameter name collisions; and
	// bindings to ensure variables cannot be overridden in the child scope.
	// We assume people that aren't using generated names know what they're
	// doing (and therefore delegate potential errors to Neo4J).
	child.paramCounter = parent.paramCounter
	for generatedName := range parent.generatedNames {
		v := parent.bindings[generatedName]
		child.bindings[generatedName] = v
		child.names[v] = generatedName
	}
	for k, v := range parent.fields {
		child.fields[k] = v
	}
}

func (s *Scope) clear() {
	s.bindings = map[string]reflect.Value{}
	s.names = map[reflect.Value]string{}
	s.generatedNames = map[string]struct{}{}
	s.fields = map[uintptr]struct{ name, entity string }{}
	s.parameters = map[string]any{}
	s.paramAddrs = map[uintptr]string{}
	s.parameterFilters = map[string]*json.FieldQuery{}
}

func (s *Scope) mergeChildScope(child *Scope) {
	for k, v := range child.bindings {
		s.bindings[k] = v
	}
	for k, v := range child.names {
		s.names[k] = v
	}
	for k, v := range child.generatedNames {
		s.generatedNames[k] = v
	}
	for k, v := range child.fields {
		s.fields[k] = v
	}
	for k, v := range child.parameters {
		s.parameters[k] = v
	}
	for k, v := range child.paramAddrs {
		s.paramAddrs[k] = v
	}
	for k, v := range child.parameterFilters {
		s.parameterFilters[k] = v
	}
	s.paramCounter = child.paramCounter
}

func (s *Scope) unfoldEntity(value any) (
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

func (s *Scope) replaceBinding(m *member) {
	v := reflect.ValueOf(m.entity)
	vT := v.Type()
	canElem := vT.Kind() == reflect.Ptr ||
		vT.Kind() == reflect.Slice

	name := m.alias
	if name == "" {
		name = m.name
	}
	if m.variable != nil && m.variable.Bind != nil {
		bind := reflect.ValueOf(m.variable.Bind)
		if bind.Kind() != reflect.Ptr {
			panic(fmt.Errorf("cannot bind to non-pointer value %s", bind))
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

	// Bind field addresses to their names
	inner := v
	for inner.Kind() == reflect.Ptr {
		inner = inner.Elem()
	}
	if canElem && inner.Kind() == reflect.Struct {
		vsT := inner.Type()
		for i := 0; i < vsT.NumField(); i++ {
			vf := inner.Field(i)
			jsTag, ok := vsT.Field(i).Tag.Lookup("json")
			if !ok {
				continue
			}
			accessor := strings.Split(jsTag, ",")[0]
			ptr := uintptr(vf.Addr().UnsafePointer())
			field := struct {
				name   string
				entity string
			}{
				name:   accessor,
				entity: name,
			}
			s.fields[ptr] = field

			fieldName := field.entity + "." + field.name
			vfAddr := vf.Addr()
			s.names[vfAddr] = fieldName
		}
	}
}

func (s *Scope) lookup(value any) *member {
	return s.register(value, true, nil)
}

func (s *Scope) register(value any, lookup bool, isNode *bool) *member {
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
		vT.Kind() == reflect.Slice

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
			if lookup {
				return nil
			}
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
		} else if lookup {
			return nil
		}
		needsName := m.name == "" && (projBody != nil || m.where != nil || v.Kind() == reflect.Ptr)
		if needsName {
			var prefix string
			if vT.Implements(nodeType) {
				prefix = strcase.ToLowerCamel(ExtractNodeLabels(entity)[0])
			} else if vT.Implements(relationshipType) {
				prefix = strcase.ToLowerCamel(ExtractRelationshipType(entity))
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
			s.generatedNames[m.name] = struct{}{}
		}
	}

	// If we are looking up a member, we are done
	if lookup {
		if m.isNew {
			return nil
		} else {
			return m
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
		case reflect.Struct, reflect.Slice:
			if inner.IsZero() {
				break
			}
			effProp := v
			effName := m.alias
			if effName == "" {
				effName = m.name
			}
			if p, ok := inner.Interface().(Param); ok {
				effName = p.Name
				prop := *p.Value
				effProp = reflect.ValueOf(prop)
			}
			param := s.addParameter(effProp, effName)
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
	return m
}

func (s *Scope) registerNode(n *node) *member {
	t := true
	return s.register(n.data, false, &t)
}

func (s *Scope) registerEdge(n *relationship) *member {
	f := false
	return s.register(n.data, false, &f)
}

func (s *Scope) Name(entity any) string {
	return s.lookupName(entity)
}

func (s *Scope) lookupName(entity any) string {
	entity, _, _ = s.unfoldEntity(entity)
	return s.names[reflect.ValueOf(entity)]
}

func (s *Scope) propertyExpression(entity any) func(v any) string {
	entity, _, _ = s.unfoldEntity(entity)
	entityName := s.lookupName(entity)
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

func (s *Scope) valueExpression(v any) string {
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

func (s *Scope) addParameter(v reflect.Value, optName string) (name string) {
	defer func() {
		if v.IsValid() && v.CanInterface() {
			s.parameters[name] = v.Interface()
		} else {
			fmt.Printf("[WARNING] invalid paramter: %s\n", name)
			s.parameters[name] = nil
		}
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
	paramPrefix := "v"
	if s.paramPrefix != "" {
		paramPrefix = s.paramPrefix
	}
	s.paramCounter++
	return paramPrefix + strconv.Itoa(s.paramCounter)
}
