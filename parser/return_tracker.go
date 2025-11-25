package parser

import (
	"fmt"

	"github.com/antlr4-go/antlr/v4"
	cypher "github.com/rlch/neogo/grammar"
)

// ReturnItem represents a single item in a RETURN clause
type ReturnItem struct {
	Expression  string // The expression text (e.g., "n", "n.name", "count(*)")
	Alias       string // The alias if present (e.g., "alias" in "n.name as alias"), empty if none
	Position    int    // Position in the return list
	IsAggregate bool   // True if this is an aggregate function like count(*), sum(), etc.
	IsWildcard  bool   // True if this is RETURN * or RETURN n.*
}

// QueryReturnInfo holds all return information from a query
type QueryReturnInfo struct {
	Query               string
	ReturnItems         []ReturnItem
	IsReturnDistinct    bool
	ReturnItemsByAlias  map[string]*ReturnItem // Fast lookup by alias
	ReturnItemsByExpr   map[string]*ReturnItem // Fast lookup by expression
	OriginalReturnText  string                  // The full RETURN clause text
}

// ExtractReturnVariables extracts all RETURN clause items from a Cypher query
func ExtractReturnVariables(query string) (*QueryReturnInfo, error) {
	tree, err := ParseCypherQuery(query)
	if err != nil {
		return nil, fmt.Errorf("failed to parse query: %w", err)
	}

	result := &QueryReturnInfo{
		Query:              query,
		ReturnItems:        []ReturnItem{},
		IsReturnDistinct:   false,
		ReturnItemsByAlias: make(map[string]*ReturnItem),
		ReturnItemsByExpr:  make(map[string]*ReturnItem),
	}

	// Walk the tree to find ReturnStContext
	var foundReturn func(antlr.Tree)
	foundReturn = func(node antlr.Tree) {
		if returnCtx, ok := node.(*cypher.ReturnStContext); ok {
			extractReturnInfo(result, returnCtx)
		}

		// Recursively walk children
		if ruleCtx, ok := node.(antlr.RuleContext); ok {
			for i := 0; i < ruleCtx.GetChildCount(); i++ {
				child := ruleCtx.GetChild(i)
				if child != nil {
					foundReturn(child)
				}
			}
		}
	}

	foundReturn(tree)

	return result, nil
}

// extractReturnInfo processes a ReturnStContext to extract return variables
func extractReturnInfo(result *QueryReturnInfo, returnCtx *cypher.ReturnStContext) {
	// Get the full return clause text
	result.OriginalReturnText = returnCtx.GetText()

	// Get ProjectionBody which contains the return items and DISTINCT keyword
	projBody := returnCtx.ProjectionBody()
	if projBody == nil {
		return
	}

	// Check for DISTINCT keyword (it's on ProjectionBody, not ReturnSt)
	if projBody.DISTINCT() != nil {
		result.IsReturnDistinct = true
	}

	// Get ProjectionItems
	projItems := projBody.ProjectionItems()
	if projItems == nil {
		return
	}

	// Check for RETURN * (wildcard) - it's a MULT token in ProjectionItems
	if projItems.MULT() != nil {
		result.ReturnItems = append(result.ReturnItems, ReturnItem{
			Expression: "*",
			Alias:      "",
			Position:   0,
			IsWildcard: true,
		})
		return
	}

	// Iterate through each projection item
	itemContexts := projItems.AllProjectionItem()
	for i, itemCtx := range itemContexts {
		// Type assert to concrete type
		if projItemCtx, ok := itemCtx.(*cypher.ProjectionItemContext); ok {
			extractProjectionItem(result, projItemCtx, i)
		}
	}
}

// extractProjectionItem processes a single ProjectionItemContext
func extractProjectionItem(result *QueryReturnInfo, itemCtx *cypher.ProjectionItemContext, position int) {
	if itemCtx == nil {
		return
	}

	exprCtx := itemCtx.Expression()
	if exprCtx == nil {
		return
	}

	expression := exprCtx.GetText()

	// Check for alias (AS keyword)
	alias := ""
	if itemCtx.AS() != nil {
		symbolCtx := itemCtx.Symbol()
		if symbolCtx != nil {
			alias = symbolCtx.GetText()
		}
	}

	// Determine if this is an aggregate function
	isAggregate := isAggregateExpression(exprCtx)

	// Check for property wildcard patterns like n.*
	isWildcard := expression == "*" || (len(expression) > 2 && expression[len(expression)-2:] == ".*")

	item := ReturnItem{
		Expression:  expression,
		Alias:       alias,
		Position:    position,
		IsAggregate: isAggregate,
		IsWildcard:  isWildcard,
	}

	result.ReturnItems = append(result.ReturnItems, item)

	// Index by alias if present
	if alias != "" {
		result.ReturnItemsByAlias[alias] = &item
	}

	// Index by expression
	result.ReturnItemsByExpr[expression] = &item
}

// isAggregateExpression checks if an expression is an aggregate function
func isAggregateExpression(exprCtx cypher.IExpressionContext) bool {
	if exprCtx == nil {
		return false
	}

	// Walk the expression tree to find function calls and aggregate constructs
	var checkFunc func(antlr.Tree) bool
	checkFunc = func(node antlr.Tree) bool {
		// Check for CountAllContext (count(*))
		if _, ok := node.(*cypher.CountAllContext); ok {
			return true
		}

		// Check if this is a function call context
		if funcCtx, ok := node.(*cypher.FunctionInvocationContext); ok {
			funcName := funcCtx.GetText()
			// Common aggregate functions in Cypher
			aggregates := map[string]bool{
				"count":      true,
				"sum":        true,
				"avg":        true,
				"min":        true,
				"max":        true,
				"collect":    true,
				"percentile": true,
				"stddev":     true,
				"stddevpop":  true,
				"stdevp":     true,
			}

			// Check if function name matches any aggregate
			for aggFunc := range aggregates {
				if len(funcName) > len(aggFunc) && funcName[:len(aggFunc)] == aggFunc {
					return true
				}
			}
		}

		// Recursively check children
		if ruleCtx, ok := node.(antlr.RuleContext); ok {
			for i := 0; i < ruleCtx.GetChildCount(); i++ {
				child := ruleCtx.GetChild(i)
				if child != nil {
					if checkFunc(child) {
						return true
					}
				}
			}
		}

		return false
	}

	return checkFunc(exprCtx)
}

// GetReturnItemCount returns the number of items in the RETURN clause
func (qri *QueryReturnInfo) GetReturnItemCount() int {
	return len(qri.ReturnItems)
}

// GetReturnExpressions returns all return expressions in order
func (qri *QueryReturnInfo) GetReturnExpressions() []string {
	exprs := make([]string, len(qri.ReturnItems))
	for i, item := range qri.ReturnItems {
		exprs[i] = item.Expression
	}
	return exprs
}

// GetReturnAliases returns all return aliases (empty string if no alias)
func (qri *QueryReturnInfo) GetReturnAliases() []string {
	aliases := make([]string, len(qri.ReturnItems))
	for i, item := range qri.ReturnItems {
		aliases[i] = item.Alias
	}
	return aliases
}

// GetReturnItemsByAlias returns a map of aliases to return items
func (qri *QueryReturnInfo) GetReturnItemsByAlias() map[string]*ReturnItem {
	return qri.ReturnItemsByAlias
}

// GetReturnItemByAlias retrieves a return item by its alias
func (qri *QueryReturnInfo) GetReturnItemByAlias(alias string) (*ReturnItem, bool) {
	item, ok := qri.ReturnItemsByAlias[alias]
	return item, ok
}

// GetReturnItemByExpression retrieves a return item by its expression
func (qri *QueryReturnInfo) GetReturnItemByExpression(expr string) (*ReturnItem, bool) {
	item, ok := qri.ReturnItemsByExpr[expr]
	return item, ok
}

// HasAlias checks if a specific alias exists in the return clause
func (qri *QueryReturnInfo) HasAlias(alias string) bool {
	_, exists := qri.ReturnItemsByAlias[alias]
	return exists
}

// GetAggregateItems returns only aggregate function return items
func (qri *QueryReturnInfo) GetAggregateItems() []ReturnItem {
	var aggregates []ReturnItem
	for _, item := range qri.ReturnItems {
		if item.IsAggregate {
			aggregates = append(aggregates, item)
		}
	}
	return aggregates
}

// GetWildcardItems returns only wildcard return items
func (qri *QueryReturnInfo) GetWildcardItems() []ReturnItem {
	var wildcards []ReturnItem
	for _, item := range qri.ReturnItems {
		if item.IsWildcard {
			wildcards = append(wildcards, item)
		}
	}
	return wildcards
}

// GetSimpleVariables returns simple variable names (identifiers without dots or functions)
func (qri *QueryReturnInfo) GetSimpleVariables() []string {
	var vars []string
	for _, item := range qri.ReturnItems {
		// Simple variable is just an identifier (no dots, no function calls, no wildcards)
		if !item.IsWildcard && !item.IsAggregate && len(item.Expression) > 0 {
			// Check if it's a simple identifier (no dots or parentheses)
			if isSimpleIdentifier(item.Expression) {
				vars = append(vars, item.Expression)
			}
		}
	}
	return vars
}

// isSimpleIdentifier checks if a string is a simple identifier without operators
func isSimpleIdentifier(s string) bool {
	if len(s) == 0 {
		return false
	}
	for _, c := range s {
		if c == '.' || c == '(' || c == ')' || c == ' ' || c == '*' {
			return false
		}
	}
	return true
}
