package internal

import (
	"errors"
	"fmt"
	"maps"
	"os"
	"reflect"
	"strconv"
	"strings"
	"text/template"

	"github.com/dlclark/regexp2"
	"github.com/iancoleman/strcase"
)

func newScope(registry *Registry) *Scope {
	return &Scope{
		Registry:       registry,
		bindings:       make(map[string]reflect.Value),
		queries:        make(map[string]*NodeSelection),
		names:          make(map[reflect.Value]string),
		generatedNames: map[string]struct{}{},
		fields:         make(map[uintptr]field),
		parameters:     map[string]any{},
		paramAddrs:     map[uintptr]string{},
	}
}

type (
	Scope struct {
		*Registry
		err error

		isWrite        bool
		bindings       map[string]reflect.Value
		queries        map[string]*NodeSelection
		generatedNames map[string]struct{}
		names          map[reflect.Value]string
		fields         map[uintptr]field

		paramCounter int
		paramPrefix  string

		parameters map[string]any
		paramAddrs map[uintptr]string
	}
	// An instance of a node/relationship in the cypher query
	member struct {
		identifier any
		// Whether the identifier was added to the scope by the query that returned this
		// member.
		isNew bool
		// The expr of the variable in the cypher query
		expr string
		// alias is the qualified name of the variable
		alias string
		// The name of the properties as a parameter in the cypher query
		propsParam string

		variable *Variable

		// The where clause that this member is associated with.
		where *Where

		// The projection body that this member is associated with.
		projectionBody *ProjectionBody
	}
	field struct {
		name       string
		identifier string
	}
)

var (
	nodeType                        = reflect.TypeOf((*INode)(nil)).Elem()
	relationshipType                = reflect.TypeOf((*IRelationship)(nil)).Elem()
	ErrExpressionAlreadyBound error = errors.New("expression already bound to different value")
	ErrAliasAlreadyBound      error = errors.New("alias already bound to expression")
)

func (s *Scope) Print() {
	t, err := template.New("").Parse(`
Parameters:
{
{{- range $key, $value := .Parameters }}
  {{ $key }}: {{ $value | printf "%v" }},
{{- end }}
}

Bindings:
{
{{- range $key, $value := .Bindings }}
  {{ $key }}: {{ $value | printf "%v" }},
{{- end }}
}

Queries:
{
{{- range $key, $value := .Queries }}
  {{ $key }}: {{ $value | printf "%v" }},
{{- end }}
}` + "\n")
	if err != nil {
		panic(err)
	}
	err = t.Execute(os.Stdout, struct {
		Parameters map[string]any
		Bindings   map[string]reflect.Value
		Queries    map[string]*NodeSelection
	}{
		Parameters: s.parameters,
		Bindings:   s.bindings,
		Queries:    s.queries,
	})
	if err != nil {
		panic(err)
	}
}

func (m *member) name() string {
	name := m.alias
	if name == "" {
		name = m.expr
	}
	return name
}

func (m *member) Print() {
	fmt.Printf(
		`{
  identifier: %+v,
  isNew: %v,
  name: %s,
  alias: %s,
  props: %s,
  variable: %+v,
  where: %+v,
  projection: %+v,
}`+"\n", m.identifier, m.isNew, m.expr, m.alias, m.propsParam, m.variable, m.where, m.projectionBody)
}

func (s *Scope) clone() *Scope {
	bindings := make(map[string]reflect.Value, len(s.bindings))
	maps.Copy(bindings, s.bindings)
	queries := make(map[string]*NodeSelection, len(s.queries))
	maps.Copy(queries, s.queries)
	generatedNames := make(map[string]struct{}, len(s.generatedNames))
	maps.Copy(generatedNames, s.generatedNames)
	names := make(map[reflect.Value]string, len(s.names))
	maps.Copy(names, s.names)
	fields := make(map[uintptr]field, len(s.fields))
	maps.Copy(fields, s.fields)
	paramCounter := s.paramCounter
	parameters := make(map[string]any, len(s.parameters))
	maps.Copy(parameters, s.parameters)
	paramAddrs := make(map[uintptr]string, len(s.paramAddrs))
	maps.Copy(paramAddrs, s.paramAddrs)
	return &Scope{
		Registry:       s.Registry,
		bindings:       bindings,
		queries:        queries,
		generatedNames: generatedNames,
		names:          names,
		fields:         fields,
		paramCounter:   paramCounter,
		parameters:     parameters,
		paramAddrs:     paramAddrs,
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
	maps.Copy(child.fields, parent.fields)
	child.Registry = parent.Registry
}

func (s *Scope) clear() {
	s.bindings = map[string]reflect.Value{}
	s.queries = map[string]*NodeSelection{}
	s.names = map[reflect.Value]string{}
	s.generatedNames = map[string]struct{}{}
	s.fields = map[uintptr]field{}
	s.parameters = map[string]any{}
	s.paramAddrs = map[uintptr]string{}
}

func (s *Scope) MergeChildScope(child *Scope) {
	maps.Copy(s.bindings, child.bindings)
	maps.Copy(s.queries, child.queries)
	maps.Copy(s.names, child.names)
	maps.Copy(s.generatedNames, child.generatedNames)
	maps.Copy(s.fields, child.fields)
	maps.Copy(s.parameters, child.parameters)
	maps.Copy(s.paramAddrs, child.paramAddrs)
	s.paramCounter = child.paramCounter
	if child.isWrite {
		s.isWrite = true
	}
	s.AddError(child.err)
}

func (s *Scope) unravelIdentifier(value any) (
	identifier any,
	variable *Variable,
	projBody *ProjectionBody,
) {
	identifier = value
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
		if variable.Expression == "" {
			variable.Expression = v.Expression
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
		if variable.VarLength == "" {
			variable.VarLength = v.VarLength
		}
		if variable.PropsExpression == "" {
			variable.PropsExpression = v.PropsExpression
		}
	}
RecurseToEntity:
	for {
		switch v := identifier.(type) {
		case *ProjectionBody:
			projBody = v
			identifier = v.Identifier
		case ProjectionBody:
			projBody = &v
			identifier = v.Identifier
		case Variable:
			mergeV(&v)
			identifier = v.Identifier
		case *Variable:
			mergeV(v)
			identifier = v.Identifier
		case Expr:
			// We allow expressions to be used as identifiers, only if they don't use _
			// as there would be no identifier to bind to. Hence the nil.
			identifier = s.compileExpression(nil)(v)
			break RecurseToEntity
		default:
			break RecurseToEntity
		}
	}
	return identifier, variable, projBody
}

func (s *Scope) replaceBinding(m *member) {
	v := reflect.ValueOf(m.identifier)
	vT := v.Type()
	canElem := vT.Kind() == reflect.Ptr ||
		vT.Kind() == reflect.Slice

	name := m.name()
	if m.variable != nil && m.variable.Bind != nil {
		bind := reflect.ValueOf(m.variable.Bind)
		if bind.Kind() != reflect.Ptr {
			panic(fmt.Errorf("cannot bind to non-pointer value %s", bind))
		}
		s.bindings[name] = bind
		s.names[bind] = name
	} else if m.alias != "" && m.alias != m.expr {
		s.names[v] = m.alias
		delete(s.bindings, m.expr)
		if canElem {
			s.bindings[m.alias] = v
		}
	} else if m.expr != "" {
		s.names[v] = m.expr
		if canElem {
			s.bindings[m.expr] = v
		}
	}

	// Bind field addresses to their names
	inner := v
	for inner.Kind() == reflect.Ptr {
		inner = inner.Elem()
	}
	if canElem && inner.Kind() == reflect.Struct {
		s.bindFields(inner, name)
	}
}

func (s *Scope) bindFields(strct reflect.Value, memberName string) {
	if err := WalkStruct(
		strct,
		func(i int, typ reflect.StructField, val reflect.Value) (bool, error) {
			accessor, ok := extractJSONFieldName(typ)
			if !ok {
				return true, nil
			}
			ptr := uintptr(val.Addr().UnsafePointer())
			f := field{name: accessor, identifier: memberName}
			s.fields[ptr] = f
			fieldName := f.identifier + "." + f.name
			addr := val.Addr()
			s.names[addr] = fieldName
			return true, nil
		},
	); err != nil {
		s.AddError(err)
	}
}

func (s *Scope) lookup(value any) *member {
	return s.add(value, true, nil)
}

func (s *Scope) add(
	value any,
	lookup bool,
	// true if the identifier is a node, false if it is a relationship
	isNode *bool,
) *member {
	if value == nil {
		return nil
	}

	m := &member{isNew: true}
	identifier, variable, projBody := s.unravelIdentifier(value)

	// Propagate information from Variable to member
	m.identifier = identifier
	if variable != nil {
		variable.Identifier = identifier
		m.variable = variable
		if variable.Expression != "" {
			m.expr = string(variable.Expression)
			if variable.Name != "" {
				m.alias = variable.Name
			}
		} else if variable.Name != "" {
			m.expr = variable.Name
		}
		if variable.Where != nil {
			m.where = variable.Where
		}
	}
	if projBody != nil {
		projBody.Identifier = identifier
		m.projectionBody = projBody
	}
	if identifier == nil {
		return m
	}

	v := reflect.ValueOf(identifier)
	vT := v.Type()
	canElem := vT.Kind() == reflect.Ptr ||
		vT.Kind() == reflect.Slice

		// Find the name of the identifier
	if m.expr != "" {
		if exst, ok := s.bindings[m.expr]; ok && exst != v {
			panic(fmt.Errorf("%w (%s): want: %v, have: %v", ErrExpressionAlreadyBound, m.expr, v, exst))
		} else if ok {
			m.isNew = false
			currentName := s.names[exst]
			// Check if name needs to be replaced
			if currentName != "" && currentName != m.expr {
				m.alias = m.expr
				m.expr = currentName
			}
		} else if !ok {
			if lookup {
				return nil
			}
			if canElem {
				// Check if name needs to be replaced
				if oldName, ok := s.names[v]; ok {
					m.alias = m.expr
					m.expr = oldName
				}
			}
		}
	} else if canElem {
		if name, ok := s.names[v]; ok {
			m.isNew = false
			m.expr = name
		} else if lookup {
			return nil
		}
		needsName := m.expr == "" && (projBody != nil || m.where != nil || v.Kind() == reflect.Ptr)
		if needsName {
			var prefix string
			if vT.Implements(nodeType) {
				prefix = strcase.ToLowerCamel(s.ExtractNodeLabels(identifier)[0])
			} else if vT.Implements(relationshipType) {
				prefix = strcase.ToLowerCamel(s.ExtractRelationshipType(identifier))
			} else {
				prefix = strcase.ToLowerCamel(vT.Elem().Name())
				if prefix == "" {
					prefix = strcase.ToLowerCamel(vT.Elem().Kind().String())
				}
			}
			if _, ok := s.bindings[prefix]; !ok {
				m.expr = prefix
			} else {
				var generatedName string
				i := 1
				for {
					generatedName = fmt.Sprintf("%s%d", prefix, i)
					if _, ok := s.bindings[generatedName]; !ok {
						break
					}
					i++
				}
				m.expr = generatedName
			}
			s.generatedNames[m.expr] = struct{}{}
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

	if name, ok := m.identifier.(string); ok {
		// Allow strings to be used as names
		if m.expr != "" {
			m.alias = m.expr
		}
		m.expr = name
	}

	s.replaceBinding(m)

	// Validate identifier type
	canHaveProps := isNode != nil &&
		((*isNode && vT.Implements(nodeType)) ||
			(!*isNode && vT.Implements(relationshipType)))

	// Check if identifier has data to inject as a parameter
	inner := v
	for inner.Kind() == reflect.Ptr {
		inner = inner.Elem()
	}
	if inner.IsValid() && m.isNew && !inner.IsZero() {
		// We have data to inject, so we need to check if
		// if m.alias != "" {
		// 	panic(fmt.Errorf("%w: alias %s already bound to expression %s", ErrAliasAlreadyBound, m.alias, m.expr))
		// }

		injectParams := func() {
			effProp := v
			effName := m.alias
			if effName == "" {
				effName = m.expr
			}
			if p, ok := inner.Interface().(Param); ok {
				effName = p.Name
				prop := *p.Value
				effProp = reflect.ValueOf(prop)
			}
			param := s.addParameter(effProp, effName)
			if canHaveProps {
				m.propsParam = param
			} else {
				m.alias = m.expr
				m.expr = param
			}
		}
		switch inner.Kind() {
		case reflect.Struct:
			if inner.CanInterface() {
				// Special case, we don't inject the fields of a Param.
				if _, ok := inner.Interface().(Param); ok {
					injectParams()
					break
				}
			}

			// Instead of injecting struct as parameter, inject its fields as
			// qualified parameters. This allows props to be used in MATCH and MERGE
			// clause for instance, where a property expression is not allowed.
			props := make(Props)
			var bindFieldsFrom func(reflect.Value)
			bindFieldsFrom = func(value reflect.Value) {
				for value.Kind() == reflect.Ptr {
					value = value.Elem()
				}
				innerT := value.Type()
				for i := range innerT.NumField() {
					f := value.Field(i)
					if !f.IsValid() || !f.CanInterface() || f.IsZero() {
						continue
					}
					fT := innerT.Field(i)
					name, ok := extractJSONFieldName(fT)
					if !ok {
						if fT.Anonymous {
							bindFieldsFrom(f)
						}
						continue
					}
					propName := name
					if m.expr != "" {
						propName = m.expr + "_" + name
					}

					prop := f.Interface()
					props[name] = Param{
						Name:  propName,
						Value: &prop,
					}
				}
			}
			bindFieldsFrom(inner)
			if len(props) > 0 {
				if m.variable == nil {
					m.variable = &Variable{}
				}
				m.variable.Props = props
			}
		case reflect.Slice, reflect.Map:
			injectParams()
		}
	}
	return m
}

func (s *Scope) addNode(n *nodePatternPart) *member {
	t := true
	member := s.add(n.data, false, &t)
	if n.selection != nil && member.expr != "" {
		s.queries[member.name()] = n.selection
	}
	return member
}

func (s *Scope) addRelationship(n *rsPatternPart) *member {
	f := false
	return s.add(n.data, false, &f)
}

func (s *Scope) Name(identifier any) string {
	return s.lookupName(identifier)
}

func (s *Scope) AddError(err error) {
	if err == nil {
		return
	}
	if s.err != nil {
		s.err = errors.Join(s.err, err)
	} else {
		s.err = err
	}
}

func (s *Scope) Error() error { return s.err }

func (s *Scope) lookupName(identifier any) string {
	identifier, _, _ = s.unravelIdentifier(identifier)
	return s.names[reflect.ValueOf(identifier)]
}

var (
	reVariableExpr = regexp2.MustCompile(`(?<![\w'"])_(?![\w'"])`, regexp2.None)
	reValueExpr    = regexp2.MustCompile(`(?<![\w'"])\?(?![\w'"])`, regexp2.None)
)

type exprStringArg string

func (s *Scope) compileExpression(identifier any) func(expr Expr) string {
	identifier, _, _ = s.unravelIdentifier(identifier)
	identifierName := s.lookupName(identifier)
	return func(expr Expr) string {
		var (
			err      error
			compiled = expr.Value
		)
		// Replace _ with identifier name
		if identifier != nil {
			compiled, err = reVariableExpr.Replace(expr.Value, identifierName, -1, -1)
			if err != nil {
				s.AddError(err)
				return ""
			}
		}
		// Replace ? with evaluated value identifier
		var valueCounter int
		compiled, err = reValueExpr.ReplaceFunc(compiled, func(m regexp2.Match) string {
			arg := expr.Args[valueCounter]
			if str, ok := arg.(string); ok {
				// If the arg is a string, we need to escape it. We use this type to delineate.
				arg = exprStringArg(str)
			}
			out := s.valueIdentifier(arg)
			valueCounter++
			return out
		}, -1, -1)
		if err != nil {
			s.AddError(err)
			return ""
		}
		return compiled
	}
}

func (s *Scope) propertyIdentifier(identifier any) func(v any) string {
	identifier, _, _ = s.unravelIdentifier(identifier)
	identifierName := s.lookupName(identifier)
	return func(v any) string {
		if v == identifier && identifierName != "" {
			return identifierName
		}
		if str, strOk := v.(string); strOk && identifierName != "" {
			// Consider strings as properties if identifier is known
			return fmt.Sprintf("%s.%s", identifierName, str)
		} else if strOk {
			// Otherwise, consider strings as literals
			return str
		} else if expr, ok := v.(Expr); ok {
			panic(fmt.Errorf("expression %s is not supported in propertyIdentifier", expr.Value))
		}
		vv := reflect.ValueOf(v)
		if vv.Kind() != reflect.Ptr {
			panic(errors.New("the key in a condition must be addressable"))
		}
		ptr := vv.Pointer()
		if name, ok := s.names[vv]; ok {
			return name
		}
		if f, ok := s.fields[ptr]; ok {
			return fmt.Sprintf("%s.%s", f.identifier, f.name)
		}
		panic(fmt.Errorf("could not find a property-representation for %v", v))
	}
}

func (s *Scope) valueIdentifier(v any) string {
	if v == nil {
		return "null"
	}
	vv := reflect.ValueOf(v)
	switch vv.Kind() {
	case reflect.Bool:
		if v.(bool) {
			return "true"
		} else {
			return "false"
		}
	case reflect.String:
		if v, ok := v.(exprStringArg); ok {
			return s.addParameter(reflect.ValueOf(string(v)), "")
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
		if f, ok := s.fields[ptr]; ok {
			return fmt.Sprintf("%s.%s", f.identifier, f.name)
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
			fmt.Printf("[WARNING] nil parameter for %s", name)
			s.parameters[name] = nil
		}
		if !strings.HasPrefix(name, "$") {
			name = "$" + name
		}
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
