package parser

import (
	"testing"

	"github.com/antlr4-go/antlr/v4"
	cypher "github.com/rlch/neogo/grammar"
)

// TestSimpleMatchReturnStructure validates the exact AST for "MATCH (n) RETURN n"
func TestSimpleMatchReturnStructure(t *testing.T) {
	query := "MATCH (n) RETURN n"
	tree, err := ParseCypherQuery(query)
	if err != nil {
		t.Fatalf("failed to parse: %v", err)
	}

	// Cast to ScriptContext (root)
	scriptCtx, ok := tree.(*cypher.ScriptContext)
	if !ok {
		t.Fatalf("expected ScriptContext, got %T", tree)
	}

	// Script should have Query child
	queryCtx := scriptCtx.Query()
	if queryCtx == nil {
		t.Fatal("script should have a Query child")
	}

	// Query should have RegularQuery child (not StandaloneCall)
	regularQueryCtx := queryCtx.RegularQuery()
	if regularQueryCtx == nil {
		t.Fatal("query should have a RegularQuery (not StandaloneCall)")
	}

	// RegularQuery should have a SingleQuery child
	singleQueryCtx := regularQueryCtx.SingleQuery()
	if singleQueryCtx == nil {
		t.Fatal("regularQuery should have a SingleQuery child")
	}

	t.Logf("✓ Simple MATCH-RETURN parse tree structure validated")
}

// TestMatchWithWhereStructure validates WHERE clause structure
func TestMatchWithWhereStructure(t *testing.T) {
	query := "MATCH (n:Person) WHERE n.age > 30 RETURN n"
	tree, err := ParseCypherQuery(query)
	if err != nil {
		t.Fatalf("failed to parse: %v", err)
	}

	scriptCtx, ok := tree.(*cypher.ScriptContext)
	if !ok {
		t.Fatalf("expected ScriptContext, got %T", tree)
	}

	// Navigate to find WHERE clause
	queryCtx := scriptCtx.Query()
	regularQueryCtx := queryCtx.RegularQuery()
	singleQueryCtx := regularQueryCtx.SingleQuery()

	// SingleQuery can be either SinglePartQ or MultiPartQ
	// Get all children and find patternWhere (which contains WHERE)
	foundWhere := false
	var walkTreeLocal func(antlr.Tree)
	walkTreeLocal = func(node antlr.Tree) {
		// Check if this is a whereContext
		if _, ok := node.(*cypher.WhereContext); ok {
			foundWhere = true
		}
		// Walk children
		if ruleCtx, ok := node.(antlr.RuleContext); ok {
			for i := 0; i < ruleCtx.GetChildCount(); i++ {
				child := ruleCtx.GetChild(i)
				if child != nil {
					walkTreeLocal(child)
				}
			}
		}
	}
	walkTreeLocal(singleQueryCtx)

	if !foundWhere {
		t.Fatal("expected to find WhereContext in the tree")
	}

	t.Logf("✓ MATCH-WHERE-RETURN parse tree structure validated")
}

// TestMatchRelationshipStructure validates relationship pattern
func TestMatchRelationshipStructure(t *testing.T) {
	query := "MATCH (a)-[r]->(b) RETURN a, b"
	tree, err := ParseCypherQuery(query)
	if err != nil {
		t.Fatalf("failed to parse: %v", err)
	}

	scriptCtx, ok := tree.(*cypher.ScriptContext)
	if !ok {
		t.Fatalf("expected ScriptContext, got %T", tree)
	}

	// Navigate to pattern
	queryCtx := scriptCtx.Query()
	regularQueryCtx := queryCtx.RegularQuery()
	singleQueryCtx := regularQueryCtx.SingleQuery()

	// Find PatternWhere which contains pattern with relationshipPattern
	foundRelationshipPattern := false
	var walkTree1 func(antlr.Tree)
	walkTree1 = func(node antlr.Tree) {
		if _, ok := node.(*cypher.RelationshipPatternContext); ok {
			foundRelationshipPattern = true
		}
		if ruleCtx, ok := node.(antlr.RuleContext); ok {
			for i := 0; i < ruleCtx.GetChildCount(); i++ {
				child := ruleCtx.GetChild(i)
				if child != nil {
					walkTree1(child)
				}
			}
		}
	}
	walkTree1(singleQueryCtx)

	if !foundRelationshipPattern {
		t.Fatal("expected to find RelationshipPatternContext in the tree")
	}

	t.Logf("✓ MATCH-relationship parse tree structure validated")
}

// TestReturnMultipleItemsStructure validates multiple items in RETURN
func TestReturnMultipleItemsStructure(t *testing.T) {
	query := "MATCH (a)-[r]->(b) RETURN a, b, r"
	tree, err := ParseCypherQuery(query)
	if err != nil {
		t.Fatalf("failed to parse: %v", err)
	}

	scriptCtx, ok := tree.(*cypher.ScriptContext)
	if !ok {
		t.Fatalf("expected ScriptContext, got %T", tree)
	}

	// Find all ProjectionItems (there should be 3)
	projectionItemCount := 0
	var walkTree2 func(antlr.Tree)
	walkTree2 = func(node antlr.Tree) {
		if _, ok := node.(*cypher.ProjectionItemContext); ok {
			projectionItemCount++
		}
		if ruleCtx, ok := node.(antlr.RuleContext); ok {
			for i := 0; i < ruleCtx.GetChildCount(); i++ {
				child := ruleCtx.GetChild(i)
				if child != nil {
					walkTree2(child)
				}
			}
		}
	}
	walkTree2(scriptCtx)

	if projectionItemCount != 3 {
		t.Errorf("expected 3 ProjectionItems, got %d", projectionItemCount)
	}

	t.Logf("✓ Multiple RETURN items parse tree structure validated")
}

// TestCreateNodeStructure validates CREATE statement structure
func TestCreateNodeStructure(t *testing.T) {
	query := "CREATE (n:Person {name: 'John'}) RETURN n"
	tree, err := ParseCypherQuery(query)
	if err != nil {
		t.Fatalf("failed to parse: %v", err)
	}

	// Find CreateSt
	foundCreate := false
	var walkTree3 func(antlr.Tree)
	walkTree3 = func(node antlr.Tree) {
		if _, ok := node.(*cypher.CreateStContext); ok {
			foundCreate = true
		}
		if ruleCtx, ok := node.(antlr.RuleContext); ok {
			for i := 0; i < ruleCtx.GetChildCount(); i++ {
				child := ruleCtx.GetChild(i)
				if child != nil {
					walkTree3(child)
				}
			}
		}
	}
	walkTree3(tree)

	if !foundCreate {
		t.Fatal("expected to find CreateStContext in the tree")
	}

	t.Logf("✓ CREATE parse tree structure validated")
}

// TestDeleteStatementStructure validates DELETE statement
func TestDeleteStatementStructure(t *testing.T) {
	query := "MATCH (n) DELETE n"
	tree, err := ParseCypherQuery(query)
	if err != nil {
		t.Fatalf("failed to parse: %v", err)
	}

	// Find DeleteSt
	foundDelete := false
	var walkTree4 func(antlr.Tree)
	walkTree4 = func(node antlr.Tree) {
		if _, ok := node.(*cypher.DeleteStContext); ok {
			foundDelete = true
		}
		if ruleCtx, ok := node.(antlr.RuleContext); ok {
			for i := 0; i < ruleCtx.GetChildCount(); i++ {
				child := ruleCtx.GetChild(i)
				if child != nil {
					walkTree4(child)
				}
			}
		}
	}
	walkTree4(tree)

	if !foundDelete {
		t.Fatal("expected to find DeleteStContext in the tree")
	}

	t.Logf("✓ DELETE parse tree structure validated")
}

// TestOrderByStructure validates ORDER BY structure
func TestOrderByStructure(t *testing.T) {
	query := "MATCH (n) RETURN n ORDER BY n.age DESC"
	tree, err := ParseCypherQuery(query)
	if err != nil {
		t.Fatalf("failed to parse: %v", err)
	}

	// Find OrderSt
	foundOrderSt := false
	var walkTree5 func(antlr.Tree)
	walkTree5 = func(node antlr.Tree) {
		if _, ok := node.(*cypher.OrderStContext); ok {
			foundOrderSt = true
		}
		if ruleCtx, ok := node.(antlr.RuleContext); ok {
			for i := 0; i < ruleCtx.GetChildCount(); i++ {
				child := ruleCtx.GetChild(i)
				if child != nil {
					walkTree5(child)
				}
			}
		}
	}
	walkTree5(tree)

	if !foundOrderSt {
		t.Fatal("expected to find OrderStContext in the tree")
	}

	t.Logf("✓ ORDER BY parse tree structure validated")
}

// TestLimitSkipStructure validates LIMIT and SKIP
func TestLimitSkipStructure(t *testing.T) {
	query := "MATCH (n) RETURN n SKIP 5 LIMIT 10"
	tree, err := ParseCypherQuery(query)
	if err != nil {
		t.Fatalf("failed to parse: %v", err)
	}

	foundLimit := false
	foundSkip := false

	var walkTree6 func(antlr.Tree)
	walkTree6 = func(node antlr.Tree) {
		if _, ok := node.(*cypher.LimitStContext); ok {
			foundLimit = true
		}
		if _, ok := node.(*cypher.SkipStContext); ok {
			foundSkip = true
		}
		if ruleCtx, ok := node.(antlr.RuleContext); ok {
			for i := 0; i < ruleCtx.GetChildCount(); i++ {
				child := ruleCtx.GetChild(i)
				if child != nil {
					walkTree6(child)
				}
			}
		}
	}
	walkTree6(tree)

	if !foundLimit {
		t.Fatal("expected to find LimitStContext in the tree")
	}
	if !foundSkip {
		t.Fatal("expected to find SkipStContext in the tree")
	}

	t.Logf("✓ LIMIT and SKIP parse tree structure validated")
}

// TestSetStatementStructure validates SET statement
func TestSetStatementStructure(t *testing.T) {
	query := "MATCH (n) SET n.name = 'Alice' RETURN n"
	tree, err := ParseCypherQuery(query)
	if err != nil {
		t.Fatalf("failed to parse: %v", err)
	}

	foundSet := false
	var walkTree7 func(antlr.Tree)
	walkTree7 = func(node antlr.Tree) {
		if _, ok := node.(*cypher.SetStContext); ok {
			foundSet = true
		}
		if ruleCtx, ok := node.(antlr.RuleContext); ok {
			for i := 0; i < ruleCtx.GetChildCount(); i++ {
				child := ruleCtx.GetChild(i)
				if child != nil {
					walkTree7(child)
				}
			}
		}
	}
	walkTree7(tree)

	if !foundSet {
		t.Fatal("expected to find SetStContext in the tree")
	}

	t.Logf("✓ SET parse tree structure validated")
}

// TestExpressionPrecedenceStructure validates that expressions are nested correctly
func TestExpressionPrecedenceStructure(t *testing.T) {
	query := "MATCH (n) WHERE n.age > 25 AND n.status = 'active' RETURN n"
	tree, err := ParseCypherQuery(query)
	if err != nil {
		t.Fatalf("failed to parse: %v", err)
	}

	// Find WhereContext
	var whereCtx *cypher.WhereContext
	var walkTree8 func(antlr.Tree)
	walkTree8 = func(node antlr.Tree) {
		if w, ok := node.(*cypher.WhereContext); ok && whereCtx == nil {
			whereCtx = w
		}
		if ruleCtx, ok := node.(antlr.RuleContext); ok {
			for i := 0; i < ruleCtx.GetChildCount(); i++ {
				child := ruleCtx.GetChild(i)
				if child != nil {
					walkTree8(child)
				}
			}
		}
	}
	walkTree8(tree)

	if whereCtx == nil {
		t.Fatal("expected to find WhereContext in the tree")
	}

	// Get the expression
	expr := whereCtx.Expression()
	if expr == nil {
		t.Fatal("WHERE should have an Expression child")
	}

	// Expression should contain AND operator (nested in andExpression)
	foundAnd := false
	var walkTree9 func(antlr.Tree)
	walkTree9 = func(node antlr.Tree) {
		if _, ok := node.(*cypher.AndExpressionContext); ok {
			foundAnd = true
		}
		if ruleCtx, ok := node.(antlr.RuleContext); ok {
			for i := 0; i < ruleCtx.GetChildCount(); i++ {
				child := ruleCtx.GetChild(i)
				if child != nil {
					walkTree9(child)
				}
			}
		}
	}
	walkTree9(expr)

	if !foundAnd {
		t.Fatal("expected to find AndExpressionContext (AND operator handling)")
	}

	t.Logf("✓ Expression precedence parse tree structure validated")
}

// TestComparisonOperatorsStructure validates different comparison operators
func TestComparisonOperatorsStructure(t *testing.T) {
	testCases := []struct {
		query string
		name  string
	}{
		{"MATCH (n) WHERE n.age > 30 RETURN n", "greater than"},
		{"MATCH (n) WHERE n.age >= 30 RETURN n", "greater or equal"},
		{"MATCH (n) WHERE n.age < 30 RETURN n", "less than"},
		{"MATCH (n) WHERE n.age <= 30 RETURN n", "less or equal"},
		{"MATCH (n) WHERE n.status = 'active' RETURN n", "equals"},
		{"MATCH (n) WHERE n.status <> 'inactive' RETURN n", "not equals"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			tree, err := ParseCypherQuery(tc.query)
			if err != nil {
				t.Fatalf("failed to parse: %v", err)
			}

			// Find ComparisonExpression
			foundComparison := false
			var walkTree10 func(antlr.Tree)
			walkTree10 = func(node antlr.Tree) {
				if _, ok := node.(*cypher.ComparisonExpressionContext); ok {
					foundComparison = true
				}
				if ruleCtx, ok := node.(antlr.RuleContext); ok {
					for i := 0; i < ruleCtx.GetChildCount(); i++ {
						child := ruleCtx.GetChild(i)
						if child != nil {
							walkTree10(child)
						}
					}
				}
			}
			walkTree10(tree)

			if !foundComparison {
				t.Fatalf("expected to find ComparisonExpressionContext for %s", tc.name)
			}
		})
	}
}

// TestPropertyAccessStructure validates property access (e.g., n.age, r.since)
func TestPropertyAccessStructure(t *testing.T) {
	query := "MATCH (a)-[r]->(b) RETURN a.name, r.since, b.age"
	tree, err := ParseCypherQuery(query)
	if err != nil {
		t.Fatalf("failed to parse: %v", err)
	}

	// Count PropertyExpressions (should be 3)
	propExprCount := 0
	var walkTree11 func(antlr.Tree)
	walkTree11 = func(node antlr.Tree) {
		if _, ok := node.(*cypher.PropertyExpressionContext); ok {
			propExprCount++
		}
		if ruleCtx, ok := node.(antlr.RuleContext); ok {
			for i := 0; i < ruleCtx.GetChildCount(); i++ {
				child := ruleCtx.GetChild(i)
				if child != nil {
					walkTree11(child)
				}
			}
		}
	}
	walkTree11(tree)

	if propExprCount < 3 {
		t.Errorf("expected at least 3 PropertyExpressions, got %d", propExprCount)
	}

	t.Logf("✓ Property access parse tree structure validated")
}

// TestNodeLabelsStructure validates node label specification
func TestNodeLabelsStructure(t *testing.T) {
	query := "MATCH (n:Person:Employee) RETURN n"
	tree, err := ParseCypherQuery(query)
	if err != nil {
		t.Fatalf("failed to parse: %v", err)
	}

	// Find NodeLabels
	foundLabels := false
	var walkTree12 func(antlr.Tree)
	walkTree12 = func(node antlr.Tree) {
		if _, ok := node.(*cypher.NodeLabelsContext); ok {
			foundLabels = true
		}
		if ruleCtx, ok := node.(antlr.RuleContext); ok {
			for i := 0; i < ruleCtx.GetChildCount(); i++ {
				child := ruleCtx.GetChild(i)
				if child != nil {
					walkTree12(child)
				}
			}
		}
	}
	walkTree12(tree)

	if !foundLabels {
		t.Fatal("expected to find NodeLabelsContext in the tree")
	}

	t.Logf("✓ Node labels parse tree structure validated")
}

// TestMapLiteralStructure validates map literal (property object)
func TestMapLiteralStructure(t *testing.T) {
	query := "CREATE (n:Person {name: 'John', age: 30}) RETURN n"
	tree, err := ParseCypherQuery(query)
	if err != nil {
		t.Fatalf("failed to parse: %v", err)
	}

	// Find MapLit
	foundMapLit := false
	var walkTree13 func(antlr.Tree)
	walkTree13 = func(node antlr.Tree) {
		if _, ok := node.(*cypher.MapLitContext); ok {
			foundMapLit = true
		}
		if ruleCtx, ok := node.(antlr.RuleContext); ok {
			for i := 0; i < ruleCtx.GetChildCount(); i++ {
				child := ruleCtx.GetChild(i)
				if child != nil {
					walkTree13(child)
				}
			}
		}
	}
	walkTree13(tree)

	if !foundMapLit {
		t.Fatal("expected to find MapLitContext in the tree")
	}

	t.Logf("✓ Map literal parse tree structure validated")
}
