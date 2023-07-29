package internal

import (
	"errors"
	"fmt"
	"reflect"
	"sort"
	"strings"
)

type cypher struct {
	*scope
	*strings.Builder
}

type CompiledCypher struct {
	Cypher     string
	Parameters map[string]any
	Bindings   map[string]reflect.Value
}

func newCypher() *cypher {
	return &cypher{
		scope:   newScope(),
		Builder: &strings.Builder{},
	}
}

func (c *cypher) Params() map[string]any {
	return c.parameters
}

func (c *cypher) Bindings() map[string]reflect.Value {
	return c.bindings
}

var (
	errMergingReturnSubclause = errors.New("cannot merge multiple RETURN sub-clauses (ORDER BY, LIMIT, SKIP, ...)")
	errWhereReturnSubclause   = errors.New("WHERE clause in RETURN sub-clause is not allowed")
	errInvalidPropExpr        = errors.New("invalid property expression. Property expressions must be strings or an identifier of some entity")
)

const indent = "  "

func (cy *cypher) newline() {
	cy.WriteByte('\n')
}

// nodePattern ::= "(" [ nodeVariable ] [ labelExpression ]
//
//	[ propertyKeyValueExpression ] [ "WHERE" booleanExpression ] ")"
func (cy *cypher) writeNode(m *member) {
	if m != nil {
		if !m.isNew {
			fmt.Fprintf(cy, "(%s)", m.name)
		} else {
			nodeLabels := extractNodeLabel(m.entity)
			cy.WriteString("(")
			padProps := false
			if m.name != "" {
				padProps = true
				cy.WriteString(m.name)
			}
			if m.variable != nil && m.variable.Pattern != "" {
				padProps = true
				fmt.Fprintf(cy, ":%s", m.variable.Pattern)
			} else if nodeLabels != nil {
				padProps = true
				fmt.Fprintf(cy, ":%s", strings.Join(nodeLabels, ":"))
			}
			if m.variable != nil && m.variable.Props != nil {
				if padProps {
					cy.WriteRune(' ')
				}
				cy.writeProps(m.variable.Props)
			} else if m.props != "" {
				if padProps {
					cy.WriteRune(' ')
				}
				cy.WriteString(m.props)
			}
			if m.where != nil {
				cy.WriteRune(' ')
				m.where.Entity = m.entity
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
func (cy *cypher) writeRelationship(m *member, rs *relationship) {
	if m != nil {
		var inner string
		if !m.isNew {
			if m.name != "" {
				inner = m.name + inner
			}
		} else {
			label := extractRelationshipType(m.entity)
			if m.variable != nil && m.variable.Pattern != "" {
				inner = ":" + string(m.variable.Pattern)
			} else if label != "" {
				inner = ":" + label
			}
			if m.name != "" {
				inner = m.name + inner
			}
			if m.variable != nil && m.variable.Quantifier != "" {
				inner = inner + string(m.variable.Quantifier)
			}
			if m.variable != nil && m.variable.Props != nil {
				inner = inner + " " + cy.writeToString(func(cy *cypher) {
					cy.writeProps(m.variable.Props)
				})
			} else if m.props != "" {
				inner = inner + " " + m.props
			}
			if m.where != nil {
				m.where.Entity = m.entity
				prevBuilder := cy.Builder
				cy.Builder = &strings.Builder{}
				cy.WriteRune(' ')
				cy.writeWhereClause(m.where, true)
				inner = inner + cy.String()
				cy.Builder = prevBuilder
			}
		}

		if rs.to != nil {
			fmt.Fprintf(cy, "-[%s]->", inner)
		} else if rs.from != nil {
			fmt.Fprintf(cy, "<-[%s]-", inner)
		} else {
			fmt.Fprintf(cy, "-[%s]-", inner)
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
		name := cy.propertyExpression(nil)(k)
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
		v := props[k.Prop]
		fmt.Fprintf(cy, "%s: %s", k.Key, v)
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
				cy.writePattern(c.Path.node())
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

func (cy *cypher) writePattern(pattern *node) {
	cy.catch(func() {
		if pattern.pathName != "" {
			fmt.Fprintf(cy, "%s = ", pattern.pathName)
		}
		for {
			nodeM := cy.registerNode(pattern)
			cy.writeNode(nodeM)
			edge := pattern.relationship
			if edge == nil {
				break
			}
			edgeM := cy.registerEdge(edge)
			cy.writeRelationship(edgeM, edge)

			if next := pattern.next(); next != pattern {
				pattern = next
			} else {
				break
			}
		}
	})
}

func (cy *cypher) writeReadingClause(patterns []*node) {
	cy.catch(func() {
		n := len(patterns)
		for i, pattern := range patterns {
			if i == 0 || (i < n-1 && pattern.Optional != patterns[i+1].Optional) {
				if pattern.Optional {
					cy.WriteString("OPTIONAL ")
				}
				cy.WriteString("MATCH")
				if i < n-1 {
					cy.WriteString("\n" + indent)
				} else {
					cy.WriteString(" ")
				}
			}
			if i > 0 {
				cy.WriteString(",\n" + indent)
			}
			cy.writePattern(pattern)
		}
		cy.newline()
	})
}

func (cy *cypher) writeCreateClause(
	nodes []*node,
) {
	cy.writeMultilineQuery("CREATE", len(nodes), func(i int) {
		cy.writePattern(nodes[i])
	})
}

func (cy *cypher) writeMergeClause(
	node *node,
	opts ...MergeOption,
) {
	merge := &MergeOptions{}
	for _, opt := range opts {
		opt.configureMergeOptions(merge)
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

func (cy *cypher) writeDeleteClause(detach bool, variables ...any) {
	if detach {
		cy.WriteString("DETACH ")
	}
	cy.writeSinglelineQuery("DELETE", len(variables), func(i int) {
		cy.WriteString(cy.propertyExpression(nil)(variables[i]))
	})
	cy.newline()
}

func (cy *cypher) writeWhereClause(where *Where, inline bool) {
	cy.catch(func() {
		cy.WriteString("WHERE ")
		if where.Expr != "" {
			cy.WriteString(string(where.Expr))
		} else {
			var cond *Condition
			if len(where.Conds) == 1 {
				cond = where.Conds[0]
			} else if len(where.Conds) > 1 {
				cond = &Condition{And: where.Conds}
			}
			cy.writeCondition(cond, cy.propertyExpression(where.Entity), cy.valueExpression)
		}
		if !inline {
			cy.newline()
		}
	})
}

func (cy *cypher) writeUnwindClause(expr any, as string) {
	cy.WriteString("UNWIND ")
	m := cy.register(expr, nil)
	fmt.Fprintf(cy, "%s AS %s", m.name, as)
	// Replace name with alias
	m.alias = as
	cy.replaceBinding(m)
	cy.newline()
}

// ProjectionBody = [[SP], (D,I,S,T,I,N,C,T)], SP, ProjectionItems, [SP, Order], [SP, Skip], [SP, Limit] ;
// ProjectionItems = ('*', { [SP], ',', [SP], ProjectionItem }) | (ProjectionItem, { [SP], ',', [SP], ProjectionItem }) ;
// ProjectionItem = (Expression, SP, (A,S), SP, Variable) | Expression ;
//
// It should be noted that any projection body constrains the variables within
// the scope of the query to that which is projected.
func (cy *cypher) writeProjectionBodyClause(clause string, vars ...any) {
	cy.catch(func() {
		cy.WriteString(clause + " ")
		var (
			subclause       *selectionSubClause
			registeredNames = make(map[string]struct{}, len(vars))
		)
		for i, v := range vars {
			m := cy.register(v, nil)
			if m.name != "" {
				if i > 0 {
					cy.WriteString(", ")
				}
			}
			if m.alias != "" {
				registeredNames[m.alias] = struct{}{}
			} else {
				registeredNames[m.name] = struct{}{}
			}
			if m.projectionBody != nil {
				if m.projectionBody.hasProjectionClauses() {
					// Merge subclauses
					if subclause == nil {
						subclause = &selectionSubClause{
							OrderBy: map[string]bool{},
						}
					}
					name := m.alias
					if name == "" {
						name = m.name
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
							if subclause.Where.Expr != "" {
								panic(errMergingReturnSubclause)
							}
							subclause.Where.Conds = append(subclause.Where.Conds, m.projectionBody.Where.Conds...)
							if m.projectionBody.Where.Expr != "" {
								if len(subclause.Where.Conds) > 0 {
									panic(errMergingReturnSubclause)
								}
								subclause.Where.Expr = m.projectionBody.Where.Expr
							}
						}
						subclause.Where = m.projectionBody.Where
					}
					for ob, asc := range m.projectionBody.OrderBy {
						if ob == "" {
							subclause.OrderBy[name] = asc
						} else if name != "" {
							subclause.OrderBy[name+"."+ob] = asc
						} else {
							subclause.OrderBy[ob] = asc
						}
					}
				}
				if m.projectionBody.Distinct {
					cy.WriteString("DISTINCT ")
				}
			}
			cy.WriteString(m.name)
			if m.alias != "" {
				fmt.Fprintf(cy, " AS %s", m.alias)
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
				orderByKeys[i] = key
				i++
			}
			sort.Slice(orderByKeys, func(u, v int) bool {
				return orderByKeys[u] < orderByKeys[v]
			})
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
				fmt.Fprintf(cy, "SKIP %s\n", subclause.Skip)
			}
			if subclause.Limit != "" {
				fmt.Fprintf(cy, "LIMIT %s\n", subclause.Limit)
			}
			if subclause.Where != nil {
				if clause == "RETURN" {
					panic(errWhereReturnSubclause)
				}
				cy.writeWhereClause(subclause.Where, false)
			}
		}
		for name, v := range cy.bindings {
			if _, ok := registeredNames[name]; ok {
				continue
			}
			delete(cy.bindings, name)
			delete(cy.names, v)
		}
	})
}

func (cy *cypher) writeSetClause(items ...SetItem) {
	cy.writeMultilineQuery("SET", len(items), func(i int) {
		item := items[i]
		prop := cy.propertyExpression(nil)(item.Entity)
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
		cy.WriteString(cy.valueExpression(item.Value))
	})
}

func (cy *cypher) writeRemoveClause(items ...RemoveItem) {
	cy.writeMultilineQuery("REMOVE", len(items), func(i int) {
		item := items[i]
		prop := cy.propertyExpression(nil)(item.Entity)
		cy.WriteString(prop)
		if len(item.Labels) > 0 {
			cy.WriteString(":" + strings.Join(item.Labels, ":"))
			return
		}
	})
}

func (cy *cypher) writeForEachClause(entity, elementsExpr any, do func(c *CypherUpdater[any])) {
	cy.catch(func() {
		cy.WriteString("FOREACH (")
		value := cy.valueExpression(elementsExpr)

		foreach := newCypher()
		m := foreach.register(entity, nil)
		fmt.Fprintf(cy, "%s IN %s | ", m.name, value)

		b := &strings.Builder{}
		foreach.Builder = b
		updater := &CypherUpdater[any]{
			cypher: foreach,
			To:     func(c *cypher) any { return nil },
		}
		do(updater)
		if updater.err != nil {
			panic(updater.err)
		}
		cy.WriteString(strings.TrimRight(b.String(), "\n") + ")")
		cy.newline()
	})
}

func (cy *cypher) writeSinglelineQuery(clause string, n int, each func(i int)) {
	cy.catch(func() {
		cy.WriteString(clause + " ")
		for i := 0; i < n; i++ {
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
		for i := 0; i < n; i++ {
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
