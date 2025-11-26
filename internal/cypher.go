package internal

import (
	"errors"
	"fmt"
	"slices"
	"sort"
	"strings"
)

type cypher struct {
	*Scope
	*strings.Builder
}

func newCypher(registry *Registry) *cypher {
	return &cypher{
		Scope:   newScope(registry),
		Builder: &strings.Builder{},
	}
}

func (c *cypher) Params() map[string]any {
	return c.parameters
}

func (c *cypher) Bindings() map[string]any {
	return c.bindings
}

func (c *cypher) Names() map[uintptr]string {
	return c.names
}

type CompiledCypher struct {
	Cypher     string
	Parameters map[string]any
	Bindings   map[string]any            // name -> pointer to user's binding target
	Plans      map[string]*BindingPlan   // Pre-compiled binding plans (avoids per-record reflection)
	IsWrite    bool
}

var (
	errMergingReturnSubclause = errors.New("cannot merge multiple RETURN sub-clauses (ORDER BY, LIMIT, SKIP, ...)")
	errWhereReturnSubclause   = errors.New("WHERE clause in RETURN sub-clause is not allowed")
	errExpressionAndCondition = errors.New("WHERE clause cannot have both expression and conditions")
	errInvalidPropExpr        = errors.New("invalid property expression. Property expressions must be strings or an identifier")
	errSubqueryImportAlias    = errors.New("aliasing or expressions are not supported in importing WITH clauses")
	errUnresolvedProps        = errors.New("resolving from multiple properties is not allowed")
)

func (s *cypher) catch(op func()) {
	defer func() {
		if r := recover(); r != nil {
			err, ok := r.(error)
			if ok {
				s.AddError(err)
			} else {
				s.AddError(fmt.Errorf("unexpected panic: %v", r))
			}
			// Debug output only in development/debug mode
			// fmt.Printf("Caught error while building query:\n%s", s.String())
		}
	}()
	op()
}

const indent = "  "

func (cy *cypher) newline() { cy.WriteByte('\n') }

// nodePattern ::= "(" [ nodeVariable ] [ labelExpression ]
//
//	[ propertyKeyValueExpression ] [ "WHERE" booleanExpression ] ")"
func (cy *cypher) writeNode(m *member) {
	if m != nil {
		if !m.isNew {
			_, err := fmt.Fprintf(cy, "(%s)", m.expr)
			if err != nil {
				cy.AddError(err)
			}
		} else {
			nodeLabels := cy.ExtractNodeLabels(m.identifier)
			cy.WriteString("(")
			padProps := false
			if m.expr != "" {
				padProps = true
				cy.WriteString(m.expr)
			}
			if m.variable != nil && m.variable.Pattern != "" {
				padProps = true
				_, err := fmt.Fprintf(cy, ":%s", m.variable.Pattern)
				if err != nil {
					cy.AddError(err)
				}
			} else if nodeLabels != nil {
				padProps = true
				_, err := fmt.Fprintf(cy, ":%s", strings.Join(nodeLabels, ":"))
				if err != nil {
					cy.AddError(err)
				}
			}
			var resolvedProps int
			if m.variable != nil {
				if m.variable.Props != nil {
					resolvedProps++
					if padProps {
						cy.WriteRune(' ')
					}
					cy.writeProps(m.variable.Props)
				}
				if m.variable.PropsExpression != "" {
					resolvedProps++
					if padProps {
						cy.WriteRune(' ')
					}
					cy.WriteString(string(m.variable.PropsExpression))
				}
			}
			if m.propsParam != "" {
				resolvedProps++
				if padProps {
					cy.WriteRune(' ')
				}
				cy.WriteString(m.propsParam)
			}
			if resolvedProps > 1 {
				panic(errUnresolvedProps)
			}
			if m.where != nil {
				cy.WriteRune(' ')
				// Use the name (m.expr) as the identifier for property lookup in WHERE.
				// This handles non-pointer identifiers like Person{} which can't be looked up by address.
				m.where.Identifier = m.expr
				cy.writeWhereClause(m.where, true)
			}
			cy.WriteString(")")
		}
	} else {
		cy.WriteString("()")
	}
}

// relationshipPattern ::= fullPattern | abbreviatedRelationship
// fullPattern ::=
//
//	  "<-[" patternFiller "]-"
//	| "-[" patternFiller "]->"
//	| "-[" patternFiller "]-"
//
// abbreviatedRelationship ::= "<--" | "--" | "-->"
// patternFiller ::= [ relationshipVariable ] [ typeExpression ]
//
//	[ propertyKeyValueExpression ] [ "WHERE" booleanExpression ]
func (cy *cypher) writeRelationship(m *member, rs *rsPatternPart) {
	if m != nil {
		var inner string
		if !m.isNew {
			if m.expr != "" {
				inner = m.expr + inner
			}
		} else {
			label := cy.ExtractRelationshipType(m.identifier)
			if m.variable != nil && m.variable.Pattern != "" {
				inner = ":" + string(m.variable.Pattern)
			} else if label != "" {
				inner = ":" + label
			}
			if m.expr != "" {
				inner = m.expr + inner
			}
			if m.variable != nil && m.variable.VarLength != "" {
				inner = inner + string(m.variable.VarLength)
			}
			var resolvedProps int
			if m.variable != nil {
				if m.variable.Props != nil {
					resolvedProps++
					inner = inner + " " + cy.writeToString(func(cy *cypher) {
						cy.writeProps(m.variable.Props)
					})
				}
				if m.variable.PropsExpression != "" {
					resolvedProps++
					inner = inner + " " + string(m.variable.PropsExpression)
				}
			}
			if m.propsParam != "" {
				resolvedProps++
				inner = inner + " " + m.propsParam
			}
			if resolvedProps > 1 {
				panic(errUnresolvedProps)
			}
			if m.where != nil {
				// Use the name (m.expr) as the identifier for property lookup in WHERE.
				// This handles non-pointer identifiers which can't be looked up by address.
				m.where.Identifier = m.expr
				prevBuilder := cy.Builder
				cy.Builder = &strings.Builder{}
				cy.WriteRune(' ')
				cy.writeWhereClause(m.where, true)
				inner = inner + cy.String()
				cy.Builder = prevBuilder
			}
		}

		var err error
		if rs.to != nil {
			_, err = fmt.Fprintf(cy, "-[%s]->", inner)
		} else if rs.from != nil {
			_, err = fmt.Fprintf(cy, "<-[%s]-", inner)
		} else {
			_, err = fmt.Fprintf(cy, "-[%s]-", inner)
		}
		if err != nil {
			cy.AddError(err)
		}
	} else {
		if rs.to != nil {
			cy.WriteString("-->")
		} else if rs.from != nil {
			cy.WriteString("<--")
		} else {
			cy.WriteString("--")
		}
	}
}

func (cy *cypher) writeProps(props Props) {
	cy.WriteString("{")
	keys := make([]struct {
		Key  string
		Prop any
	}, len(props))
	i := 0
	for k := range props {
		name := cy.propertyIdentifier(nil)(k)
		accessors := strings.Split(name, ".")
		if len(accessors) == 2 {
			name = accessors[1]
		} else if len(accessors) > 2 || name == "" {
			panic(errInvalidPropExpr)
		}
		keys[i] = struct {
			Key  string
			Prop any
		}{
			Key:  name,
			Prop: k,
		}
		i++
	}
	sort.Slice(keys, func(u, v int) bool {
		return keys[u].Key < keys[v].Key
	})
	for i, k := range keys {
		if i > 0 {
			cy.WriteString(", ")
		}
		v := cy.valueIdentifier(props[k.Prop])
		_, err := fmt.Fprintf(cy, "%s: %s", k.Key, v)
		if err != nil {
			cy.AddError(err)
		}
	}
	cy.WriteString("}")
}

func (cy *cypher) writeCondition(c *Condition, parseKey, parseValue func(any) string) {
	cy.catch(func() {
		var recurse func(*Condition, bool) string
		recurse = func(c *Condition, wrap bool) (s string) {
			conjuctive := len(c.Xor) > 0 || len(c.Or) > 0 || len(c.And) > 0
			defer func() {
				if conjuctive && wrap {
					s = "(" + s + ")"
				}
				if c.Not {
					s = "NOT " + s
				}
			}()
			if c.Path != nil {
				prevBuilder := cy.Builder
				cy.Builder = &strings.Builder{}
				cy.writePattern(c.Path.createNodePattern(cy.Registry))
				s += cy.String()
				cy.Builder = prevBuilder
			} else if n := len(c.Xor); n > 0 {
				for i, cond := range c.Xor {
					s += recurse(cond, true)
					if i < n-1 {
						s += " XOR "
					}
				}
			} else if n := len(c.Or); n > 0 {
				for i, cond := range c.Or {
					s += recurse(cond, true)
					if i < n-1 {
						s += " OR "
					}
				}
			} else if n := len(c.And); n > 0 {
				for i, cond := range c.And {
					s += recurse(cond, true)
					if i < n-1 {
						s += " AND "
					}
				}
			} else {
				if c.Op == "" && c.Value == nil {
					s = parseKey(c.Key)
					return
				}
				s = fmt.Sprintf("%s %s %s", parseKey(c.Key), c.Op, parseValue(c.Value))
			}
			return
		}
		cy.WriteString(recurse(c, false))
	})
}

func (cy *cypher) writePattern(pattern *nodePatternPart) {
	cy.catch(func() {
		if pattern.pathName != "" {
			_, err := fmt.Fprintf(cy, "%s = ", pattern.pathName)
			if err != nil {
				cy.AddError(err)
			}
		}
		for {
			nodeM := cy.addNode(pattern)
			cy.writeNode(nodeM)
			rs := pattern.relationship
			if rs == nil {
				break
			}
			rsM := cy.addRelationship(rs)
			cy.writeRelationship(rsM, rs)

			if next := pattern.next(); next != pattern {
				pattern = next
			} else {
				break
			}
		}
	})
}

func (cy *cypher) writeReadingClause(patterns []*nodePatternPart, optional bool) {
	clause := "MATCH"
	if optional {
		clause = "OPTIONAL " + clause
	}
	cy.writeMultilineQuery(clause, len(patterns), func(i int) {
		pattern := patterns[i]
		cy.writePattern(pattern)
	})
}

func (cy *cypher) writeUseClause(graphExpr string) {
	cy.WriteString("USE " + graphExpr)
	cy.newline()
}

func (cy *cypher) writeUnionClause(unions []func(*CypherClient) *CypherRunner, all bool, parent *Scope) {
	clause := "UNION"
	if all {
		clause += " ALL"
	}
	runners := make([]*CypherRunner, len(unions))
	for i, union := range unions {
		rootScope := cy.clone()
		rootCy := newCypher(cy.Registry)
		rootCy.Scope = rootScope
		childCy := newCypherClient(rootCy)
		// Parent scope of CALL should be propagated to UNION if exists
		childCy.Parent = parent
		runners[i] = union(childCy)
	}
	cy.clear()
	// TODO: Potentially perform a check to ensure bindings and names are
	// equivalent from compiled runner. We assume they are and let Neo4J handle
	// errors.
	queries := make([]string, len(runners))
	for i, runner := range runners {
		comp, err := runner.Compile()
		if err != nil {
			panic(err)
		}
		queries[i] = comp.Cypher
		cy.MergeChildScope(runner.Scope)
	}
	cy.WriteString(strings.Join(queries, "\n"+clause+"\n"))
}

func (cy *cypher) writeCreateClause(
	nodes []*nodePatternPart,
) {
	cy.writeMultilineQuery("CREATE", len(nodes), func(i int) {
		cy.writePattern(nodes[i])
	})
}

func (cy *cypher) writeMergeClause(
	node *nodePatternPart,
	opts ...MergeOption,
) {
	merge := &Merge{}
	for _, opt := range opts {
		opt.configureMerge(merge)
	}
	cy.catch(func() {
		cy.WriteString("MERGE ")
		cy.writePattern(node)
		cy.newline()

		if merge.OnCreate != nil {
			cy.WriteString("ON CREATE\n")
			cy.writeIndented("  ", func(cy *cypher) {
				cy.writeSetClause(merge.OnCreate...)
			})
		}
		if merge.OnMatch != nil {
			cy.WriteString("ON MATCH\n")
			cy.writeIndented("  ", func(cy *cypher) {
				cy.writeSetClause(merge.OnMatch...)
			})
		}
	})
}

func (cy *cypher) writeDeleteClause(detach bool, propIdentifiers ...any) {
	if detach {
		cy.WriteString("DETACH ")
	}
	cy.writeSinglelineQuery("DELETE", len(propIdentifiers), func(i int) {
		cy.WriteString(cy.propertyIdentifier(nil)(propIdentifiers[i]))
	})
	cy.newline()
}

func (cy *cypher) writeWhereClause(where *Where, inline bool) {
	cy.catch(func() {
		cy.WriteString("WHERE ")
		if where.Expr != nil && len(where.Conds) > 0 {
			cy.AddError(errExpressionAndCondition)
			return
		}
		if where.Expr != nil {
			expression := cy.compileExpression(where.Identifier)(*where.Expr)
			cy.WriteString(expression)
		} else {
			var cond *Condition
			if len(where.Conds) == 1 {
				cond = where.Conds[0]
			} else if len(where.Conds) > 1 {
				cond = &Condition{And: where.Conds}
			}
			cy.writeCondition(cond, cy.propertyIdentifier(where.Identifier), cy.valueIdentifier)
		}
		if !inline {
			cy.newline()
		}
	})
}

func (cy *cypher) writeUnwindClause(expr any, as string) {
	cy.WriteString("UNWIND ")
	m := cy.add(expr, false, nil)
	_, err := fmt.Fprintf(cy, "%s AS %s", m.expr, as)
	if err != nil {
		cy.AddError(err)
	}
	// Replace name with alias
	m.alias = as
	cy.replaceBinding(m)
	cy.newline()
}

func (cy *cypher) writeSubqueryClause(subquery func(c *CypherClient) *CypherRunner) {
	cy.catch(func() {
		child := NewCypherClient(cy.Registry)
		child.Parent = cy.Scope
		child.mergeParentScope(child.Parent)
		runSubquery := subquery(child)

		_, err := fmt.Fprintf(cy, "CALL {\n")
		if err != nil {
			cy.AddError(err)
		}
		cy.writeIndented("  ", func(cy *cypher) {
			compiled, err := runSubquery.Compile()
			if err != nil {
				panic(err)
			}
			cy.WriteString(compiled.Cypher)
			cy.MergeChildScope(runSubquery.Scope)
			cy.isWrite = cy.isWrite || compiled.IsWrite
		})
		cy.WriteString("\n}\n")
	})
}

// ProjectionBody = [[SP], (D,I,S,T,I,N,C,T)], SP, ProjectionItems, [SP, Order], [SP, Skip], [SP, Limit] ;
// ProjectionItems = ('*', { [SP], ',', [SP], ProjectionItem }) | (ProjectionItem, { [SP], ',', [SP], ProjectionItem }) ;
// ProjectionItem = (Expression, SP, (A,S), SP, Variable) | Expression ;
//
// It should be noted that any projection body constrains the variables within
// the scope of the query to that which is projected.
func (cy *cypher) writeProjectionBodyClause(clause string, parent *Scope, vars ...any) {
	isWith := clause == "WITH"
	register := func(v any) (m *member, allowAlias bool) {
		if isWith && parent != nil {
			// WITH is a special case, as it allows for reusing variables from the
			// parent scope.
			m := parent.lookup(v)
			if m != nil {
				// Bind the variable from the parent scope to the child scope.
				cy.replaceBinding(m)
				return m, false
			}
		}
		return cy.add(v, false, nil), true
	}
	cy.catch(func() {
		cy.WriteString(clause + " ")
		var (
			subclause       *selectionSubClause
			registeredNames = make(map[string]struct{}, len(vars))
		)
		for i, v := range vars {
			m, allowAlias := register(v)
			if m.expr != "" {
				if i > 0 {
					cy.WriteString(", ")
				}
			}
			if m.alias != "" {
				if !allowAlias {
					panic(errSubqueryImportAlias)
				}
				registeredNames[m.alias] = struct{}{}
			} else {
				registeredNames[m.expr] = struct{}{}
			}
			if m.projectionBody != nil {
				if m.projectionBody.hasProjectionClauses() {
					// Merge subclauses
					if subclause == nil {
						subclause = &selectionSubClause{
							OrderBy: map[any]bool{},
						}
					}
					if m.projectionBody.Limit != "" {
						if subclause.Limit != "" {
							panic(errMergingReturnSubclause)
						}
						subclause.Limit = m.projectionBody.Limit
					}
					if m.projectionBody.Skip != "" {
						if subclause.Skip != "" {
							panic(errMergingReturnSubclause)
						}
						subclause.Skip = m.projectionBody.Skip
					}
					if m.projectionBody.Where != nil {
						if subclause.Where != nil {
							if subclause.Where.Expr != nil {
								panic(errMergingReturnSubclause)
							}
							subclause.Where.Conds = append(subclause.Where.Conds, m.projectionBody.Where.Conds...)
							if m.projectionBody.Where.Expr != nil {
								if len(subclause.Where.Conds) > 0 {
									panic(errMergingReturnSubclause)
								}
								subclause.Where.Expr = m.projectionBody.Where.Expr
							}
						}
						subclause.Where = m.projectionBody.Where
					}
					for ob, asc := range m.projectionBody.OrderBy {
						getKey := cy.propertyIdentifier(m.identifier)
						var key string
						if ob == "" || ob == nil {
							key = getKey(m.identifier)
						} else {
							key = getKey(ob)
						}
						subclause.OrderBy[key] = asc
					}
				}
				if m.projectionBody.Distinct {
					cy.WriteString("DISTINCT ")
				}
			}
			cy.WriteString(m.expr)
			if m.alias != "" {
				_, err := fmt.Fprintf(cy, " AS %s", m.alias)
				if err != nil {
					cy.AddError(err)
				}
			}
		}
		cy.newline()
		if subclause != nil {
			n := len(subclause.OrderBy)
			if n > 0 {
				cy.WriteString("ORDER BY ")
			}
			orderByKeys := make([]string, len(subclause.OrderBy))
			i := 0
			for key := range subclause.OrderBy {
				orderByKeys[i] = key.(string)
				i++
			}
			slices.Sort(orderByKeys)
			for i, sb := range orderByKeys {
				asc := subclause.OrderBy[sb]
				if i > 0 {
					cy.WriteString(", ")
				}
				cy.WriteString(sb)
				if !asc {
					cy.WriteString(" DESC")
				}
				if i == n-1 {
					cy.newline()
				}
			}
			if subclause.Skip != "" {
				_, err := fmt.Fprintf(cy, "SKIP %s\n", subclause.Skip)
				if err != nil {
					cy.AddError(err)
				}
			}
			if subclause.Limit != "" {
				_, err := fmt.Fprintf(cy, "LIMIT %s\n", subclause.Limit)
				if err != nil {
					cy.AddError(err)
				}
			}
			if subclause.Where != nil {
				if !isWith {
					panic(errWhereReturnSubclause)
				}
				cy.writeWhereClause(subclause.Where, false)
			}
		}
		if _, hasWildcard := registeredNames["*"]; hasWildcard {
			return
		}
		for name, v := range cy.bindings {
			if _, ok := registeredNames[name]; ok {
				continue
			}
			delete(cy.bindings, name)
			delete(cy.names, ptrAddr(v))
		}
	})
}

func (cy *cypher) writeSetClause(items ...SetItem) {
	cy.writeMultilineQuery("SET", len(items), func(i int) {
		item := items[i]
		prop := cy.propertyIdentifier(nil)(item.PropIdentifier)
		cy.WriteString(prop)
		if len(item.Labels) > 0 {
			cy.WriteString(":" + strings.Join(item.Labels, ":"))
			return
		}
		if item.Merge {
			cy.WriteString(" += ")
		} else {
			cy.WriteString(" = ")
		}
		cy.WriteString(cy.valueIdentifier(item.ValIdentifier))
	})
}

func (cy *cypher) writeRemoveClause(items ...RemoveItem) {
	cy.writeMultilineQuery("REMOVE", len(items), func(i int) {
		item := items[i]
		prop := cy.propertyIdentifier(nil)(item.PropIdentifier)
		cy.WriteString(prop)
		if len(item.Labels) > 0 {
			cy.WriteString(":" + strings.Join(item.Labels, ":"))
			return
		}
	})
}

func (cy *cypher) writeForEachClause(identifier, elementsExpr any, do func(c *CypherUpdater[any])) {
	cy.catch(func() {
		cy.WriteString("FOREACH (")
		value := cy.valueIdentifier(elementsExpr)

		foreach := newCypher(cy.Registry)
		m := foreach.add(identifier, false, nil)
		_, err := fmt.Fprintf(cy, "%s IN %s | ", m.expr, value)
		if err != nil {
			cy.AddError(err)
		}

		b := &strings.Builder{}
		foreach.Builder = b
		updater := &CypherUpdater[any]{
			cypher: foreach,
			To:     func(c *cypher) any { return nil },
		}
		do(updater)
		if updater.Error() != nil {
			panic(updater.Error())
		}
		cy.WriteString(strings.TrimRight(b.String(), "\n") + ")")
		cy.newline()
	})
}

func (cy *cypher) writeCallClause(procedure string) {
	cy.WriteString("CALL " + procedure)
	cy.newline()
}

func (cy *cypher) writeShowClause(procedure string) {
	cy.WriteString("SHOW " + procedure)
	cy.newline()
}

func (cy *cypher) writeYieldClause(identifiers ...any) {
	cy.writeSinglelineQuery("YIELD", len(identifiers), func(i int) {
		v := identifiers[i]
		m := cy.add(v, false, nil)
		cy.WriteString(m.expr)
		if m.alias != "" {
			_, err := fmt.Fprintf(cy, " AS %s", m.alias)
			if err != nil {
				cy.AddError(err)
			}
		}
	})
}

func (cy *cypher) writeSinglelineQuery(clause string, n int, each func(i int)) {
	cy.catch(func() {
		cy.WriteString(clause + " ")
		for i := range n {
			if i > 0 {
				cy.WriteString(", ")
			}
			each(i)
		}
		cy.newline()
	})
}

func (cy *cypher) writeMultilineQuery(clause string, n int, each func(i int)) {
	cy.catch(func() {
		cy.WriteString(clause)
		if n > 1 {
			cy.WriteString("\n" + indent)
		} else {
			cy.WriteString(" ")
		}
		for i := range n {
			if i > 0 {
				cy.WriteString(",\n" + indent)
			}
			each(i)
		}
		cy.newline()
	})
}

func (cy *cypher) writeIndented(indent string, write func(cy *cypher)) {
	cy.catch(func() {
		prevBuilder := cy.Builder
		indentBuilder := &strings.Builder{}
		cy.Builder = indentBuilder
		write(cy)
		cy.Builder = prevBuilder
		for i, line := range strings.Split(indentBuilder.String(), "\n") {
			if i > 0 {
				cy.WriteString("\n")
			}
			if line == "" {
				continue
			}
			if strings.HasPrefix(line, "UNION") {
				cy.WriteString(line)
				continue
			}
			cy.WriteString(indent + line)
		}
	})
}

func (cy *cypher) writeToString(write func(cy *cypher)) string {
	prevBuilder := cy.Builder
	stringBuilder := &strings.Builder{}
	cy.Builder = stringBuilder
	write(cy)
	cy.Builder = prevBuilder
	return stringBuilder.String()
}
