package internal

import (
	"errors"
	"fmt"
	"maps"
	"os"
	"regexp"
	"strings"
	"text/template"
)

func NewCypherClient(registry *Registry) *CypherClient {
	cy := newCypher(registry)
	return newCypherClient(cy)
}

func newCypherClient(cy *cypher) *CypherClient {
	return &CypherClient{
		cypher:        cy,
		CypherReader:  newCypherReader(cy, nil),
		CypherUpdater: newCypherUpdater(cy),
	}
}

type (
	CypherClient struct {
		*cypher
		*CypherReader
		*CypherUpdater[*CypherQuerier]
	}
	CypherQuerier struct {
		*cypher
		*CypherReader
		*CypherRunner
		*CypherUpdater[*CypherQuerier]
	}
	CypherReader struct {
		*cypher
		*CypherRunner
		Parent *Scope
	}
	CypherYielder struct {
		*cypher
		*CypherQuerier
	}
	CypherUpdater[To any] struct {
		*cypher
		*CypherRunner
		To func(*cypher) To
	}
	CypherRunner struct {
		*cypher
		isReturn bool
	}
)

var (
	isWriteRe               = regexp.MustCompile(`\b(CREATE|MERGE|DELETE|SET|REMOVE|CALL\s+\w.*)\b`)
	errInvalidConditionArgs = errors.New("expected condition to be ICondition, <key> <op> <value> or <expr> <args>")
)

func newCypherQuerier(cy *cypher) *CypherQuerier {
	q := &CypherQuerier{
		cypher:        cy,
		CypherReader:  newCypherReader(cy, nil),
		CypherUpdater: newCypherUpdater(cy),
		CypherRunner:  newCypherRunner(cy, false),
	}
	return q
}

func newCypherReader(cy *cypher, parent *Scope) *CypherReader {
	return &CypherReader{
		cypher:       cy,
		CypherRunner: newCypherRunner(cy, false),
		Parent:       parent,
	}
}

func newCypherUpdater(cy *cypher) *CypherUpdater[*CypherQuerier] {
	return &CypherUpdater[*CypherQuerier]{
		cypher: cy,
		To: func(c *cypher) *CypherQuerier {
			// We know if this is executed, the query has some update clause.
			c.isWrite = true
			return newCypherQuerier(c)
		},
		CypherRunner: newCypherRunner(cy, true),
	}
}

func newCypherYielder(cy *cypher) *CypherYielder {
	return &CypherYielder{
		cypher:        cy,
		CypherQuerier: newCypherQuerier(cy),
	}
}

func newCypherRunner(cy *cypher, isReturn bool) *CypherRunner {
	return &CypherRunner{cypher: cy, isReturn: isReturn}
}

func (c *CypherClient) Use(graphExpr string) *CypherQuerier {
	c.writeUseClause(graphExpr)
	return newCypherQuerier(c.cypher)
}

func (c *CypherClient) Union(unions ...func(c *CypherClient) *CypherRunner) *CypherQuerier {
	c.writeUnionClause(unions, false, c.Parent)
	q := newCypherQuerier(c.cypher)
	q.isReturn = true
	return q
}

func (c *CypherClient) UnionAll(unions ...func(c *CypherClient) *CypherRunner) *CypherQuerier {
	c.writeUnionClause(unions, true, c.Parent)
	q := newCypherQuerier(c.cypher)
	q.isReturn = true
	return q
}

func (c *CypherReader) OptionalMatch(patterns Patterns) *CypherQuerier {
	var nodes []*nodePatternPart
	if patterns != nil {
		nodes = patterns.nodes(c.Registry)
	}
	c.writeReadingClause(nodes, true)
	return newCypherQuerier(c.cypher)
}

func (c *CypherReader) Match(patterns Patterns) *CypherQuerier {
	var nodes []*nodePatternPart
	if patterns != nil {
		nodes = patterns.nodes(c.Registry)
	}
	c.writeReadingClause(nodes, false)
	return newCypherQuerier(c.cypher)
}

func (c *CypherReader) Subquery(subquery func(c *CypherClient) *CypherRunner) *CypherQuerier {
	c.writeSubqueryClause(subquery)
	return newCypherQuerier(c.cypher)
}

func (c *CypherReader) With(identifiers ...any) *CypherQuerier {
	c.writeProjectionBodyClause("WITH", c.Parent, identifiers...)
	return newCypherQuerier(c.cypher)
}

func (c *CypherReader) Unwind(identifier any, as string) *CypherQuerier {
	c.writeUnwindClause(identifier, as)
	return newCypherQuerier(c.cypher)
}

func (c *CypherReader) Call(procedure string) *CypherYielder {
	c.writeCallClause(procedure)
	c.isWrite = true
	return newCypherYielder(c.cypher)
}

func (c *CypherReader) Show(command string) *CypherYielder {
	c.writeShowClause(command)
	return newCypherYielder(c.cypher)
}

func (c *CypherReader) Return(identifiers ...any) *CypherRunner {
	c.writeProjectionBodyClause("RETURN", nil, identifiers...)
	return newCypherRunner(c.cypher, true)
}

func (c *CypherReader) Cypher(query string) *CypherQuerier {
	b := strings.ToUpper(query)
	c.isWrite = c.isWrite || isWriteRe.Find([]byte(b)) != nil
	c.WriteString(query + "\n")
	return newCypherQuerier(c.cypher)
}

func (c *CypherReader) Eval(expression func(*Scope, *strings.Builder)) *CypherQuerier {
	b := strings.ToUpper(c.String())
	expression(c.Scope, c.Builder)
	c.isWrite = c.isWrite || isWriteRe.Find([]byte(b)) != nil
	c.newline()
	return newCypherQuerier(c.cypher)
}

func (c *CypherQuerier) Where(args ...any) *CypherQuerier {
	where := &Where{}
	argsToCondition := func() (ICondition, error) {
		if len(args) == 0 {
			return nil, errInvalidConditionArgs
		}
		if firstCond, ok := args[0].(ICondition); ok {
			if len(args) == 1 {
				return firstCond, nil
			}
			conds := make([]*Condition, len(args))
			for i, arg := range args {
				if cond, ok := arg.(ICondition); ok {
					conds[i] = cond.Condition()
				} else {
					return nil, fmt.Errorf("expected all args to be ICondition, but arg %d is %T", i, arg)
				}
			}
			return &Condition{And: conds}, nil
		}
		tryParseCond := func() ICondition {
			key := args[0]
			op, ok := args[1].(string)
			if !ok {
				return nil
			}
			value := args[2]
			return &Condition{Key: key, Op: op, Value: value}
		}
		if len(args) == 3 {
			if cond := tryParseCond(); cond != nil {
				return cond, nil
			}
		}
		query, ok := args[0].(string)
		if !ok {
			return nil, errInvalidConditionArgs
		}
		return &Expr{Value: query, Args: args[1:]}, nil
	}
	cond, err := argsToCondition()
	if err != nil {
		panic(err)
	} else {
		cond.configureWhere(where)
	}
	c.writeWhereClause(where, false)
	return newCypherQuerier(c.cypher)
}

func (c *CypherUpdater[To]) Create(pattern Patterns) To {
	c.writeCreateClause(pattern.nodes(c.Registry))
	return c.To(c.cypher)
}

func (c *CypherUpdater[To]) Merge(pattern Pattern, opts ...MergeOption) To {
	c.writeMergeClause(pattern.createNodePattern(c.Registry), opts...)
	return c.To(c.cypher)
}

func (c *CypherUpdater[To]) DetachDelete(propIdentifiers ...any) To {
	c.writeDeleteClause(true, propIdentifiers...)
	return c.To(c.cypher)
}

func (c *CypherUpdater[To]) Delete(identifiers ...any) To {
	c.writeDeleteClause(false, identifiers...)
	return c.To(c.cypher)
}

func (c *CypherUpdater[To]) Set(items ...SetItem) To {
	c.writeSetClause(items...)
	return c.To(c.cypher)
}

func (c *CypherUpdater[To]) Remove(items ...RemoveItem) To {
	c.writeRemoveClause(items...)
	return c.To(c.cypher)
}

func (c *CypherUpdater[To]) ForEach(identifier, elementsExpr any, do func(c *CypherUpdater[any])) To {
	c.writeForEachClause(identifier, elementsExpr, do)
	return c.To(c.cypher)
}

func (c *CypherYielder) Yield(identifiers ...any) *CypherQuerier {
	c.writeYieldClause(identifiers...)
	return newCypherQuerier(c.cypher)
}

func (c *CypherRunner) CompileWithParams(params map[string]any) (*CompiledCypher, error) {
	maps.Copy(c.parameters, params)
	return c.Compile()
}

func (c *CypherRunner) Compile() (*CompiledCypher, error) {
	out := c.String()
	out = strings.TrimRight(out, "\n")
	if !c.isReturn {
		c.bindings = map[string]any{}
	}
	cy := &CompiledCypher{
		Cypher:     out,
		Parameters: c.parameters,
		Bindings:   c.bindings,
		IsWrite:    c.isWrite,
	}
	// Build binding plans for zero-reflection hot path
	if len(c.bindings) > 0 {
		cy.Plans = BuildBindingPlans(c.bindings, c.Registry.Codecs())
	}
	if c.err != nil {
		return nil, c.err
	}
	return cy, nil
}

func (c *CypherRunner) Print() *CypherRunner {
	out := c.String()
	out = strings.TrimRight(out, "\n")
	fmt.Println(out)
	return c
}

func (c *CypherRunner) DebugPrint() *CypherRunner {
	t, err := template.New("").Parse(`
Cypher (write: {{ .IsWrite }}):
{{ .Cypher }}

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

` + "\n")
	if err != nil {
		panic(err)
	}
	err = t.Execute(os.Stdout, struct {
		Cypher     string
		Parameters map[string]any
		Bindings   map[string]any
		IsWrite    bool
	}{
		Cypher:     c.String(),
		Parameters: c.parameters,
		Bindings:   c.bindings,
		IsWrite:    c.isWrite,
	})
	if err != nil {
		panic(err)
	}
	return c
}
