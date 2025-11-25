package parser

import (
	"fmt"

	"github.com/antlr4-go/antlr/v4"
	cypher "github.com/rlch/neogo/grammar"
)

// ArgumentInfo holds information about a single argument/parameter
type ArgumentInfo struct {
	Name     string // The parameter name without $ (e.g., "x" for $x)
	Position int    // Position in the query where it appears (for ordering)
	Contexts int    // Number of times this argument appears
}

// QueryArguments holds all arguments found in a query
type QueryArguments struct {
	Query      string
	Arguments  []ArgumentInfo
	IndexByKey map[string]*ArgumentInfo // Fast lookup by name
}

// ExtractArguments extracts all function arguments ($x, $y, etc.) from a Cypher query
func ExtractArguments(query string) (*QueryArguments, error) {
	tree, err := ParseCypherQuery(query)
	if err != nil {
		return nil, fmt.Errorf("failed to parse query: %w", err)
	}

	result := &QueryArguments{
		Query:      query,
		Arguments:  []ArgumentInfo{},
		IndexByKey: make(map[string]*ArgumentInfo),
	}

	// Walk the tree to find all ParameterContext nodes
	var walkTreeFunc func(antlr.Tree)
	walkTreeFunc = func(node antlr.Tree) {
		if paramCtx, ok := node.(*cypher.ParameterContext); ok {
			// Extract parameter name from Symbol() or NumLit()
			if symbol := paramCtx.Symbol(); symbol != nil {
				paramName := symbol.GetText()
				addOrUpdateArgument(result, paramName, int(node.(antlr.ParseTree).GetSourceInterval().Start))
			}
			// NumLit parameters like $1, $2 are also valid (numeric placeholders)
			if numLit := paramCtx.NumLit(); numLit != nil {
				paramName := numLit.GetText()
				addOrUpdateArgument(result, paramName, int(node.(antlr.ParseTree).GetSourceInterval().Start))
			}
		}

		// Recursively walk children
		if ruleCtx, ok := node.(antlr.RuleContext); ok {
			for i := 0; i < ruleCtx.GetChildCount(); i++ {
				child := ruleCtx.GetChild(i)
				if child != nil {
					walkTreeFunc(child)
				}
			}
		}
	}

	walkTreeFunc(tree)

	return result, nil
}

// addOrUpdateArgument adds a new argument or updates the count if it already exists
func addOrUpdateArgument(qa *QueryArguments, name string, position int) {
	if arg, exists := qa.IndexByKey[name]; exists {
		arg.Contexts++
	} else {
		arg := ArgumentInfo{
			Name:     name,
			Position: position,
			Contexts: 1,
		}
		qa.Arguments = append(qa.Arguments, arg)
		qa.IndexByKey[name] = &arg
	}
}

// GetArgumentCount returns the total number of unique arguments
func (qa *QueryArguments) GetArgumentCount() int {
	return len(qa.Arguments)
}

// GetArgumentNames returns all argument names in order of appearance
func (qa *QueryArguments) GetArgumentNames() []string {
	names := make([]string, len(qa.Arguments))
	for i, arg := range qa.Arguments {
		names[i] = arg.Name
	}
	return names
}

// HasArgument checks if a specific argument exists
func (qa *QueryArguments) HasArgument(name string) bool {
	_, exists := qa.IndexByKey[name]
	return exists
}

// GetArgument retrieves argument info by name
func (qa *QueryArguments) GetArgument(name string) (ArgumentInfo, bool) {
	if arg, exists := qa.IndexByKey[name]; exists {
		return *arg, true
	}
	return ArgumentInfo{}, false
}
