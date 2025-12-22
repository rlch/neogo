package internal

import (
	"fmt"
	"regexp"
	"strings"
)

// CypherClient is the simplified query builder that only supports raw Cypher queries.
type CypherClient struct {
	*Registry
	query      string
	parameters map[string]any
	bindings   map[string]any
	isWrite    bool
}

// NewCypherClient creates a new CypherClient.
func NewCypherClient(registry *Registry) *CypherClient {
	return &CypherClient{
		Registry:   registry,
		parameters: make(map[string]any),
		bindings:   make(map[string]any),
	}
}

var isWriteRe = regexp.MustCompile(`\b(CREATE|MERGE|DELETE|SET|REMOVE|CALL\s+\w.*)\b`)

// Cypher sets the Cypher query string.
func (c *CypherClient) Cypher(query string) *CypherRunner {
	c.query = query
	c.isWrite = isWriteRe.MatchString(strings.ToUpper(query))
	return &CypherRunner{client: c}
}

// CypherRunner executes Cypher queries.
type CypherRunner struct {
	client *CypherClient
}

// CompiledCypher contains the compiled query ready for execution.
type CompiledCypher struct {
	Cypher     string
	Parameters map[string]any
	Bindings   map[string]any
	Plans      map[string]*BindingPlan
	IsWrite    bool
}

// Compile compiles the query with the given bindings.
// bindings should be pairs of (name string, target any).
func (r *CypherRunner) Compile(bindings ...any) (*CompiledCypher, error) {
	return r.CompileWithParams(nil, bindings...)
}

// CompileWithParams compiles the query with parameters and bindings.
func (r *CypherRunner) CompileWithParams(params map[string]any, bindings ...any) (*CompiledCypher, error) {
	if len(bindings)%2 != 0 {
		return nil, fmt.Errorf("bindings must be pairs of (name, target), got %d args", len(bindings))
	}

	// Parse bindings
	bindingMap := make(map[string]any, len(bindings)/2)
	for i := 0; i < len(bindings); i += 2 {
		name, ok := bindings[i].(string)
		if !ok {
			return nil, fmt.Errorf("binding name at position %d must be a string, got %T", i, bindings[i])
		}
		target := bindings[i+1]
		bindingMap[name] = target
	}

	// Merge params
	parameters := make(map[string]any, len(r.client.parameters)+len(params))
	for k, v := range r.client.parameters {
		parameters[k] = v
	}
	for k, v := range params {
		parameters[k] = v
	}

	cy := &CompiledCypher{
		Cypher:     strings.TrimRight(r.client.query, "\n"),
		Parameters: parameters,
		Bindings:   bindingMap,
		IsWrite:    r.client.isWrite,
	}

	// Build binding plans for zero-reflection hot path
	if len(bindingMap) > 0 {
		cy.Plans = BuildBindingPlans(bindingMap, r.client.Codecs())
	}

	return cy, nil
}

// Print prints the query to stdout.
func (r *CypherRunner) Print() *CypherRunner {
	fmt.Println(r.client.query)
	return r
}
