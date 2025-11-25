package parser

import (
	"testing"
)

// TestSimpleReturnVariable validates basic RETURN extraction
func TestSimpleReturnVariable(t *testing.T) {
	query := "MATCH (n) RETURN n"
	result, err := ExtractReturnVariables(query)
	if err != nil {
		t.Fatalf("failed to extract return variables: %v", err)
	}

	if result.GetReturnItemCount() != 1 {
		t.Errorf("expected 1 return item, got %d", result.GetReturnItemCount())
	}

	items := result.GetReturnExpressions()
	if len(items) > 0 && items[0] != "n" {
		t.Errorf("expected expression 'n', got '%s'", items[0])
	}

	t.Logf("✓ Simple return variable validated")
}

// TestMultipleReturnVariables validates RETURN with multiple items
func TestMultipleReturnVariables(t *testing.T) {
	query := "MATCH (a)-[r]->(b) RETURN a, b, r"
	result, err := ExtractReturnVariables(query)
	if err != nil {
		t.Fatalf("failed to extract return variables: %v", err)
	}

	if result.GetReturnItemCount() != 3 {
		t.Errorf("expected 3 return items, got %d", result.GetReturnItemCount())
	}

	exprs := result.GetReturnExpressions()
	expected := []string{"a", "b", "r"}
	if len(exprs) != len(expected) {
		t.Fatalf("expected %d expressions, got %d", len(expected), len(exprs))
	}

	for i, exp := range expected {
		if exprs[i] != exp {
			t.Errorf("expression at position %d: expected '%s', got '%s'", i, exp, exprs[i])
		}
	}

	t.Logf("✓ Multiple return variables validated: %v", exprs)
}

// TestReturnWithAlias validates RETURN items with aliases
func TestReturnWithAlias(t *testing.T) {
	query := "MATCH (a:Person), (b:Person) RETURN a.name as personA, b.name as personB"
	result, err := ExtractReturnVariables(query)
	if err != nil {
		t.Fatalf("failed to extract return variables: %v", err)
	}

	if result.GetReturnItemCount() != 2 {
		t.Errorf("expected 2 return items, got %d", result.GetReturnItemCount())
	}

	if !result.HasAlias("personA") {
		t.Errorf("expected alias 'personA' to exist")
	}
	if !result.HasAlias("personB") {
		t.Errorf("expected alias 'personB' to exist")
	}

	aliases := result.GetReturnAliases()
	if len(aliases) != 2 {
		t.Fatalf("expected 2 aliases, got %d", len(aliases))
	}

	if aliases[0] != "personA" || aliases[1] != "personB" {
		t.Errorf("aliases don't match: expected [personA, personB], got %v", aliases)
	}

	t.Logf("✓ Return with aliases validated")
}

// TestReturnPropertyAccess validates RETURN with property access (n.name, r.since)
func TestReturnPropertyAccess(t *testing.T) {
	query := "MATCH (a:Person)-[r:KNOWS]->(b:Person) RETURN a.name, r.since, b.age"
	result, err := ExtractReturnVariables(query)
	if err != nil {
		t.Fatalf("failed to extract return variables: %v", err)
	}

	if result.GetReturnItemCount() != 3 {
		t.Errorf("expected 3 return items, got %d", result.GetReturnItemCount())
	}

	exprs := result.GetReturnExpressions()
	expectedExprs := []string{"a.name", "r.since", "b.age"}
	
	for i, expected := range expectedExprs {
		if i < len(exprs) && exprs[i] != expected {
			t.Errorf("expression %d: expected '%s', got '%s'", i, expected, exprs[i])
		}
	}

	t.Logf("✓ Property access in RETURN validated: %v", exprs)
}

// TestReturnDistinct validates RETURN DISTINCT
func TestReturnDistinct(t *testing.T) {
	query := "MATCH (n) RETURN DISTINCT n.type"
	result, err := ExtractReturnVariables(query)
	if err != nil {
		t.Fatalf("failed to extract return variables: %v", err)
	}

	if !result.IsReturnDistinct {
		t.Errorf("expected IsReturnDistinct to be true")
	}

	t.Logf("✓ RETURN DISTINCT validated")
}

// TestReturnWildcard validates RETURN * (return all)
func TestReturnWildcard(t *testing.T) {
	query := "MATCH (n) RETURN *"
	result, err := ExtractReturnVariables(query)
	if err != nil {
		t.Fatalf("failed to extract return variables: %v", err)
	}

	if result.GetReturnItemCount() != 1 {
		t.Errorf("expected 1 return item for RETURN *, got %d", result.GetReturnItemCount())
	}

	wildcards := result.GetWildcardItems()
	if len(wildcards) != 1 {
		t.Errorf("expected 1 wildcard item, got %d", len(wildcards))
	}

	if wildcards[0].Expression != "*" {
		t.Errorf("expected wildcard expression '*', got '%s'", wildcards[0].Expression)
	}

	t.Logf("✓ RETURN wildcard validated")
}

// TestReturnDistinctMultiple validates RETURN DISTINCT with multiple items
func TestReturnDistinctMultiple(t *testing.T) {
	query := "MATCH (n) RETURN DISTINCT n.type, n.status"
	result, err := ExtractReturnVariables(query)
	if err != nil {
		t.Fatalf("failed to extract return variables: %v", err)
	}

	if !result.IsReturnDistinct {
		t.Errorf("expected IsReturnDistinct to be true")
	}

	if result.GetReturnItemCount() < 2 {
		t.Errorf("expected at least 2 return items, got %d", result.GetReturnItemCount())
	}

	t.Logf("✓ RETURN DISTINCT with multiple items validated")
}

// TestReturnWithCountAggregate validates RETURN with count() aggregate
func TestReturnWithCountAggregate(t *testing.T) {
	query := "MATCH (n) RETURN count(*)"
	result, err := ExtractReturnVariables(query)
	if err != nil {
		t.Fatalf("failed to extract return variables: %v", err)
	}

	aggregates := result.GetAggregateItems()
	if len(aggregates) != 1 {
		t.Errorf("expected 1 aggregate item, got %d", len(aggregates))
	}

	if aggregates[0].IsAggregate != true {
		t.Errorf("expected aggregate flag to be true")
	}

	t.Logf("✓ COUNT aggregate validated: %s", aggregates[0].Expression)
}

// TestReturnWithMultipleAggregates validates multiple aggregate functions
func TestReturnWithMultipleAggregates(t *testing.T) {
	query := "MATCH (n) RETURN count(n), sum(n.value), avg(n.score), min(n.age), max(n.age)"
	result, err := ExtractReturnVariables(query)
	if err != nil {
		t.Fatalf("failed to extract return variables: %v", err)
	}

	aggregates := result.GetAggregateItems()
	if len(aggregates) < 4 {
		t.Errorf("expected at least 4 aggregate items, got %d", len(aggregates))
	}

	t.Logf("✓ Multiple aggregates validated, found %d aggregates", len(aggregates))
}

// TestReturnWithAggregateAndAlias validates aggregate with alias
func TestReturnWithAggregateAndAlias(t *testing.T) {
	query := "MATCH (n) RETURN count(*) as total, sum(n.value) as totalValue"
	result, err := ExtractReturnVariables(query)
	if err != nil {
		t.Fatalf("failed to extract return variables: %v", err)
	}

	if !result.HasAlias("total") {
		t.Errorf("expected alias 'total' to exist")
	}
	if !result.HasAlias("totalValue") {
		t.Errorf("expected alias 'totalValue' to exist")
	}

	aggregates := result.GetAggregateItems()
	if len(aggregates) < 2 {
		t.Errorf("expected at least 2 aggregates, got %d", len(aggregates))
	}

	t.Logf("✓ Aggregate with aliases validated")
}

// TestReturnPreservesOrder validates that return items preserve order
func TestReturnPreservesOrder(t *testing.T) {
	query := "MATCH (a), (b), (c), (d), (e) RETURN a, b, c, d, e"
	result, err := ExtractReturnVariables(query)
	if err != nil {
		t.Fatalf("failed to extract return variables: %v", err)
	}

	if result.GetReturnItemCount() != 5 {
		t.Errorf("expected 5 return items, got %d", result.GetReturnItemCount())
	}

	exprs := result.GetReturnExpressions()
	expectedOrder := []string{"a", "b", "c", "d", "e"}

	for i, expected := range expectedOrder {
		if exprs[i] != expected {
			t.Errorf("position %d: expected '%s', got '%s'", i, expected, exprs[i])
		}
	}

	t.Logf("✓ Return order preservation validated: %v", exprs)
}

// TestReturnMixedAliasedAndNot validates mixing aliased and non-aliased items
func TestReturnMixedAliasedAndNot(t *testing.T) {
	query := "MATCH (n) RETURN n, n.name as name, n.age"
	result, err := ExtractReturnVariables(query)
	if err != nil {
		t.Fatalf("failed to extract return variables: %v", err)
	}

	if result.GetReturnItemCount() != 3 {
		t.Errorf("expected 3 return items, got %d", result.GetReturnItemCount())
	}

	aliases := result.GetReturnAliases()
	// Second item should have alias, others should be empty
	if aliases[0] != "" {
		t.Errorf("position 0: expected no alias, got '%s'", aliases[0])
	}
	if aliases[1] != "name" {
		t.Errorf("position 1: expected 'name', got '%s'", aliases[1])
	}
	if aliases[2] != "" {
		t.Errorf("position 2: expected no alias, got '%s'", aliases[2])
	}

	t.Logf("✓ Mixed aliased/non-aliased validated")
}

// TestComplexReturnExpression validates complex expressions in RETURN
func TestComplexReturnExpression(t *testing.T) {
	query := "MATCH (n) RETURN n, n.name + ' ' + n.title as fullName, n.age * 2 as ageDouble"
	result, err := ExtractReturnVariables(query)
	if err != nil {
		t.Fatalf("failed to extract return variables: %v", err)
	}

	if result.GetReturnItemCount() < 2 {
		t.Errorf("expected at least 2 return items, got %d", result.GetReturnItemCount())
	}

	if !result.HasAlias("fullName") {
		t.Errorf("expected alias 'fullName' to exist")
	}
	if !result.HasAlias("ageDouble") {
		t.Errorf("expected alias 'ageDouble' to exist")
	}

	t.Logf("✓ Complex return expressions validated")
}

// TestReturnWithLiterals validates RETURN with literal values
func TestReturnWithLiterals(t *testing.T) {
	query := "MATCH (n) RETURN n, 'literal' as literal, 42 as answer"
	result, err := ExtractReturnVariables(query)
	if err != nil {
		t.Fatalf("failed to extract return variables: %v", err)
	}

	if !result.HasAlias("literal") {
		t.Errorf("expected alias 'literal' to exist")
	}
	if !result.HasAlias("answer") {
		t.Errorf("expected alias 'answer' to exist")
	}

	t.Logf("✓ Return with literals validated")
}

// TestReturnSimpleVariables validates extraction of simple variable names
func TestReturnSimpleVariables(t *testing.T) {
	query := "MATCH (a), (b), (c) RETURN a, b, c"
	result, err := ExtractReturnVariables(query)
	if err != nil {
		t.Fatalf("failed to extract return variables: %v", err)
	}

	simpleVars := result.GetSimpleVariables()
	if len(simpleVars) != 3 {
		t.Errorf("expected 3 simple variables, got %d", len(simpleVars))
	}

	expected := []string{"a", "b", "c"}
	for i, exp := range expected {
		if i < len(simpleVars) && simpleVars[i] != exp {
			t.Errorf("variable %d: expected '%s', got '%s'", i, exp, simpleVars[i])
		}
	}

	t.Logf("✓ Simple variables extraction validated: %v", simpleVars)
}

// TestReturnSimpleVariablesWithProperties validates that properties are not in simple variables
func TestReturnSimpleVariablesWithProperties(t *testing.T) {
	query := "MATCH (a) RETURN a, a.name, a.age"
	result, err := ExtractReturnVariables(query)
	if err != nil {
		t.Fatalf("failed to extract return variables: %v", err)
	}

	simpleVars := result.GetSimpleVariables()
	// Only 'a' should be a simple variable; a.name and a.age are property accesses
	if len(simpleVars) != 1 {
		t.Errorf("expected 1 simple variable, got %d: %v", len(simpleVars), simpleVars)
	}

	if simpleVars[0] != "a" {
		t.Errorf("expected simple variable 'a', got '%s'", simpleVars[0])
	}

	t.Logf("✓ Simple variables filtering validated")
}

// TestReturnInComplexQuery validates return extraction in a complex query
func TestReturnInComplexQuery(t *testing.T) {
	query := `
		MATCH (p:Person {id: $id})-[r:WORKS_AT]->(c:Company)
		WHERE c.active = true
		RETURN
			p.id as personId,
			p.name as personName,
			c.id as companyId,
			c.name as companyName,
			r.since as startDate,
			count(r) as relationshipCount
		ORDER BY p.name DESC
		LIMIT 10
	`

	result, err := ExtractReturnVariables(query)
	if err != nil {
		t.Fatalf("failed to extract return variables: %v", err)
	}

	expectedAliases := map[string]bool{
		"personId":           true,
		"personName":         true,
		"companyId":          true,
		"companyName":        true,
		"startDate":          true,
		"relationshipCount":  true,
	}

	for alias := range expectedAliases {
		if !result.HasAlias(alias) {
			t.Errorf("expected alias '%s' to exist", alias)
		}
	}

	aggregates := result.GetAggregateItems()
	if len(aggregates) < 1 {
		t.Errorf("expected at least 1 aggregate, got %d", len(aggregates))
	}

	t.Logf("✓ Complex query return validation: %d items with %d aggregates", result.GetReturnItemCount(), len(aggregates))
}

// TestReturnPreservesQuery validates that original query is stored
func TestReturnPreservesQuery(t *testing.T) {
	query := "MATCH (n) RETURN n"
	result, err := ExtractReturnVariables(query)
	if err != nil {
		t.Fatalf("failed to extract return variables: %v", err)
	}

	if result.Query != query {
		t.Errorf("expected query to be preserved, got %q", result.Query)
	}

	t.Logf("✓ Query preservation validated")
}

// TestReturnWithCaseExpression validates RETURN with CASE expressions
func TestReturnWithCaseExpression(t *testing.T) {
	query := "MATCH (n) RETURN CASE WHEN n.age > 30 THEN 'adult' ELSE 'young' END as category"
	result, err := ExtractReturnVariables(query)
	if err != nil {
		t.Fatalf("failed to extract return variables: %v", err)
	}

	if !result.HasAlias("category") {
		t.Errorf("expected alias 'category' from CASE expression")
	}

	t.Logf("✓ CASE expression in RETURN validated")
}

// TestReturnWithCollect validates RETURN with collect() aggregate
func TestReturnWithCollect(t *testing.T) {
	query := "MATCH (n:Person)-[r]->(m) RETURN n.name, collect(m) as related"
	result, err := ExtractReturnVariables(query)
	if err != nil {
		t.Fatalf("failed to extract return variables: %v", err)
	}

	if !result.HasAlias("related") {
		t.Errorf("expected alias 'related' to exist")
	}

	aggregates := result.GetAggregateItems()
	if len(aggregates) < 1 {
		t.Errorf("expected at least 1 aggregate (collect), got %d", len(aggregates))
	}

	t.Logf("✓ Collect aggregate validated")
}

// TestReturnItemByAlias validates retrieval by alias
func TestReturnItemByAlias(t *testing.T) {
	query := "MATCH (n) RETURN n.name as name, n.age as age"
	result, err := ExtractReturnVariables(query)
	if err != nil {
		t.Fatalf("failed to extract return variables: %v", err)
	}

	item, ok := result.GetReturnItemByAlias("name")
	if !ok {
		t.Fatal("expected to find return item by alias 'name'")
	}

	if item.Alias != "name" {
		t.Errorf("expected alias 'name', got '%s'", item.Alias)
	}

	t.Logf("✓ Return item retrieval by alias validated")
}

// TestReturnItemByExpression validates retrieval by expression
func TestReturnItemByExpression(t *testing.T) {
	query := "MATCH (n) RETURN n.name, n.age"
	result, err := ExtractReturnVariables(query)
	if err != nil {
		t.Fatalf("failed to extract return variables: %v", err)
	}

	item, ok := result.GetReturnItemByExpression("n.name")
	if !ok {
		t.Errorf("expected to find return item by expression 'n.name'")
	} else {
		if item.Expression != "n.name" {
			t.Errorf("expected expression 'n.name', got '%s'", item.Expression)
		}
	}

	t.Logf("✓ Return item retrieval by expression validated")
}

// TestReturnWithFunctionCalls validates RETURN with various function calls
func TestReturnWithFunctionCalls(t *testing.T) {
	query := "MATCH (n) RETURN n, length(n.name) as nameLength, upper(n.type) as typeUpper, toInteger(n.value) as intValue"
	result, err := ExtractReturnVariables(query)
	if err != nil {
		t.Fatalf("failed to extract return variables: %v", err)
	}

	expectedAliases := []string{"nameLength", "typeUpper", "intValue"}
	for _, alias := range expectedAliases {
		if !result.HasAlias(alias) {
			t.Errorf("expected alias '%s' to exist", alias)
		}
	}

	t.Logf("✓ Function calls in RETURN validated")
}

// TestReturnWithDistinctAndAggregates validates RETURN DISTINCT with aggregates
func TestReturnWithDistinctAndAggregates(t *testing.T) {
	query := "MATCH (n) RETURN DISTINCT n.type, count(*) as count"
	result, err := ExtractReturnVariables(query)
	if err != nil {
		t.Fatalf("failed to extract return variables: %v", err)
	}

	if !result.IsReturnDistinct {
		t.Errorf("expected IsReturnDistinct to be true")
	}

	aggregates := result.GetAggregateItems()
	if len(aggregates) < 1 {
		t.Errorf("expected at least 1 aggregate, got %d", len(aggregates))
	}

	t.Logf("✓ RETURN DISTINCT with aggregates validated")
}

// TestReturnWithManyItems validates RETURN with many items
func TestReturnWithManyItems(t *testing.T) {
	query := "MATCH (a), (b), (c), (d), (e), (f), (g), (h) RETURN a, b, c, d, e, f, g, h"
	result, err := ExtractReturnVariables(query)
	if err != nil {
		t.Fatalf("failed to extract return variables: %v", err)
	}

	if result.GetReturnItemCount() != 8 {
		t.Errorf("expected 8 return items, got %d", result.GetReturnItemCount())
	}

	exprs := result.GetReturnExpressions()
	expectedExprs := []string{"a", "b", "c", "d", "e", "f", "g", "h"}

	for i, expected := range expectedExprs {
		if exprs[i] != expected {
			t.Errorf("position %d: expected '%s', got '%s'", i, expected, exprs[i])
		}
	}

	t.Logf("✓ Many return items validated")
}

// TestReturnWithPercentileAggregate validates percentile-like aggregates
func TestReturnWithPercentileAggregate(t *testing.T) {
	// Note: Cypher percentile function may require integer argument
	query := "MATCH (n) RETURN max(n.score) as maxScore, min(n.score) as minScore"
	result, err := ExtractReturnVariables(query)
	if err != nil {
		t.Fatalf("failed to extract return variables: %v", err)
	}

	aggregates := result.GetAggregateItems()
	if len(aggregates) < 2 {
		t.Errorf("expected at least 2 aggregates, got %d", len(aggregates))
	}

	if !result.HasAlias("maxScore") {
		t.Errorf("expected alias 'maxScore' to exist")
	}
	if !result.HasAlias("minScore") {
		t.Errorf("expected alias 'minScore' to exist")
	}

	t.Logf("✓ Min/Max aggregates validated")
}

// BenchmarkReturnExtraction measures performance of return extraction
func BenchmarkReturnExtraction(b *testing.B) {
	query := "MATCH (p:Person)-[r:WORKS_AT]->(c:Company) RETURN p.id as id, p.name as name, c.name as company, r.since as since, count(r) as count ORDER BY p.name DESC LIMIT 10"
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := ExtractReturnVariables(query)
		if err != nil {
			b.Fatalf("failed to extract return variables: %v", err)
		}
	}
}
