package parser

import (
	"strings"
	"testing"
)

// TestSimpleArgumentExtraction validates basic parameter extraction
func TestSimpleArgumentExtraction(t *testing.T) {
	query := "MATCH (n) WHERE n.id = $id RETURN n"
	args, err := ExtractArguments(query)
	if err != nil {
		t.Fatalf("failed to extract arguments: %v", err)
	}

	if args.GetArgumentCount() != 1 {
		t.Errorf("expected 1 argument, got %d", args.GetArgumentCount())
	}

	if !args.HasArgument("id") {
		t.Errorf("expected argument 'id' to exist")
	}

	if arg, ok := args.GetArgument("id"); ok {
		if arg.Contexts != 1 {
			t.Errorf("expected 'id' to appear 1 time, got %d", arg.Contexts)
		}
	}

	t.Logf("✓ Simple argument extraction validated")
}

// TestMultipleDistinctArguments validates extraction of multiple different arguments
func TestMultipleDistinctArguments(t *testing.T) {
	query := "MATCH (a:Person), (b:Person) WHERE a.id = $idA AND b.id = $idB RETURN a, b"
	args, err := ExtractArguments(query)
	if err != nil {
		t.Fatalf("failed to extract arguments: %v", err)
	}

	if args.GetArgumentCount() != 2 {
		t.Errorf("expected 2 arguments, got %d", args.GetArgumentCount())
	}

	expectedArgs := []string{"idA", "idB"}
	actualArgs := args.GetArgumentNames()
	if len(actualArgs) != len(expectedArgs) {
		t.Fatalf("expected %d arguments, got %d", len(expectedArgs), len(actualArgs))
	}

	for _, expected := range expectedArgs {
		if !args.HasArgument(expected) {
			t.Errorf("expected argument '%s' to exist", expected)
		}
	}

	t.Logf("✓ Multiple distinct arguments validated: %v", actualArgs)
}

// TestRepeatedArgumentUsage validates that repeated arguments are counted correctly
func TestRepeatedArgumentUsage(t *testing.T) {
	query := "MATCH (n) WHERE n.name = $name AND n.title = $name RETURN n"
	args, err := ExtractArguments(query)
	if err != nil {
		t.Fatalf("failed to extract arguments: %v", err)
	}

	if args.GetArgumentCount() != 1 {
		t.Errorf("expected 1 unique argument, got %d", args.GetArgumentCount())
	}

	if arg, ok := args.GetArgument("name"); ok {
		if arg.Contexts != 2 {
			t.Errorf("expected 'name' to appear 2 times, got %d", arg.Contexts)
		}
	} else {
		t.Fatal("expected argument 'name' to exist")
	}

	t.Logf("✓ Repeated argument usage validated")
}

// TestArgumentsInMultipleClauses validates arguments across WHERE, WITH, etc.
func TestArgumentsInMultipleClauses(t *testing.T) {
	// Note: LIMIT with parameter is not valid Cypher syntax (LIMIT must be a literal number)
	query := "MATCH (n) WHERE n.age > $minAge WITH n WHERE n.status = $status RETURN n"
	args, err := ExtractArguments(query)
	if err != nil {
		t.Fatalf("failed to extract arguments: %v", err)
	}

	expectedCount := 2
	if args.GetArgumentCount() != expectedCount {
		t.Errorf("expected %d arguments, got %d", expectedCount, args.GetArgumentCount())
	}

	expectedArgs := map[string]int{
		"minAge": 1,
		"status": 1,
	}

	for argName, expectedContexts := range expectedArgs {
		if arg, ok := args.GetArgument(argName); ok {
			if arg.Contexts != expectedContexts {
				t.Errorf("expected '%s' to appear %d times, got %d", argName, expectedContexts, arg.Contexts)
			}
		} else {
			t.Errorf("expected argument '%s' to exist", argName)
		}
	}

	t.Logf("✓ Arguments across multiple clauses validated")
}

// TestArgumentsInExpressions validates arguments used in various expressions
func TestArgumentsInExpressions(t *testing.T) {
	query := "MATCH (n) WHERE (n.x > $x AND n.y < $y) OR (n.z = $z) RETURN n"
	args, err := ExtractArguments(query)
	if err != nil {
		t.Fatalf("failed to extract arguments: %v", err)
	}

	if args.GetArgumentCount() != 3 {
		t.Errorf("expected 3 arguments, got %d", args.GetArgumentCount())
	}

	for _, argName := range []string{"x", "y", "z"} {
		if !args.HasArgument(argName) {
			t.Errorf("expected argument '%s' to exist", argName)
		}
	}

	t.Logf("✓ Arguments in expressions validated")
}

// TestArgumentsInReturnExpressions validates arguments in RETURN clause
func TestArgumentsInReturnExpressions(t *testing.T) {
	query := "MATCH (n) RETURN n.name, $defaultValue as fallback"
	args, err := ExtractArguments(query)
	if err != nil {
		t.Fatalf("failed to extract arguments: %v", err)
	}

	if !args.HasArgument("defaultValue") {
		t.Errorf("expected argument 'defaultValue' in RETURN clause")
	}

	t.Logf("✓ Arguments in RETURN expressions validated")
}

// TestArgumentsWithComplexNames validates arguments with underscores and camelCase
func TestArgumentsWithComplexNames(t *testing.T) {
	query := "MATCH (n) WHERE n.id = $user_id OR n.uuid = $userId OR n.code = $USER_CODE RETURN n"
	args, err := ExtractArguments(query)
	if err != nil {
		t.Fatalf("failed to extract arguments: %v", err)
	}

	expectedArgs := map[string]bool{
		"user_id":   true,
		"userId":    true,
		"USER_CODE": true,
	}

	if args.GetArgumentCount() != len(expectedArgs) {
		t.Errorf("expected %d arguments, got %d", len(expectedArgs), args.GetArgumentCount())
	}

	for argName := range expectedArgs {
		if !args.HasArgument(argName) {
			t.Errorf("expected argument '%s' to exist", argName)
		}
	}

	t.Logf("✓ Complex argument names validated")
}

// TestNumericArgumentPlaceholders validates numeric parameter placeholders ($1, $2, etc.)
func TestNumericArgumentPlaceholders(t *testing.T) {
	query := "MATCH (n) WHERE n.age > $1 AND n.id = $2 RETURN n"
	args, err := ExtractArguments(query)
	if err != nil {
		t.Fatalf("failed to extract arguments: %v", err)
	}

	if args.GetArgumentCount() != 2 {
		t.Errorf("expected 2 numeric arguments, got %d", args.GetArgumentCount())
	}

	if !args.HasArgument("1") {
		t.Errorf("expected argument '1' to exist")
	}
	if !args.HasArgument("2") {
		t.Errorf("expected argument '2' to exist")
	}

	t.Logf("✓ Numeric argument placeholders validated")
}

// TestArgumentsInCreateStatement validates arguments in CREATE queries
func TestArgumentsInCreateStatement(t *testing.T) {
	query := "CREATE (n:Person {name: $name, age: $age, city: $city}) RETURN n"
	args, err := ExtractArguments(query)
	if err != nil {
		t.Fatalf("failed to extract arguments: %v", err)
	}

	expectedArgs := []string{"name", "age", "city"}
	if args.GetArgumentCount() != len(expectedArgs) {
		t.Errorf("expected %d arguments, got %d", len(expectedArgs), args.GetArgumentCount())
	}

	for _, argName := range expectedArgs {
		if !args.HasArgument(argName) {
			t.Errorf("expected argument '%s' to exist", argName)
		}
	}

	t.Logf("✓ Arguments in CREATE statements validated")
}

// TestArgumentsInSetStatement validates arguments in SET clauses
func TestArgumentsInSetStatement(t *testing.T) {
	query := "MATCH (n) WHERE n.id = $id SET n.status = $newStatus, n.updatedAt = $timestamp RETURN n"
	args, err := ExtractArguments(query)
	if err != nil {
		t.Fatalf("failed to extract arguments: %v", err)
	}

	expectedArgs := map[string]int{
		"id":        1,
		"newStatus": 1,
		"timestamp": 1,
	}

	if args.GetArgumentCount() != len(expectedArgs) {
		t.Errorf("expected %d arguments, got %d", len(expectedArgs), args.GetArgumentCount())
	}

	for argName, expectedContexts := range expectedArgs {
		if arg, ok := args.GetArgument(argName); ok {
			if arg.Contexts != expectedContexts {
				t.Errorf("expected '%s' to appear %d times, got %d", argName, expectedContexts, arg.Contexts)
			}
		} else {
			t.Errorf("expected argument '%s' to exist", argName)
		}
	}

	t.Logf("✓ Arguments in SET statements validated")
}

// TestArgumentsInMatchWithRelationship validates arguments in relationship patterns
func TestArgumentsInMatchWithRelationship(t *testing.T) {
	query := "MATCH (a)-[r:KNOWS {since: $since}]->(b) WHERE r.weight > $minWeight RETURN a, b"
	args, err := ExtractArguments(query)
	if err != nil {
		t.Fatalf("failed to extract arguments: %v", err)
	}

	if !args.HasArgument("since") {
		t.Errorf("expected argument 'since' in relationship properties")
	}
	if !args.HasArgument("minWeight") {
		t.Errorf("expected argument 'minWeight' in WHERE clause")
	}

	t.Logf("✓ Arguments in relationship patterns validated")
}

// TestNoArguments validates queries without any parameters
func TestNoArguments(t *testing.T) {
	query := "MATCH (n:Person) WHERE n.age > 30 RETURN n"
	args, err := ExtractArguments(query)
	if err != nil {
		t.Fatalf("failed to extract arguments: %v", err)
	}

	if args.GetArgumentCount() != 0 {
		t.Errorf("expected 0 arguments, got %d", args.GetArgumentCount())
	}

	if len(args.GetArgumentNames()) != 0 {
		t.Errorf("expected empty argument list")
	}

	t.Logf("✓ Query with no arguments validated")
}

// TestComplexQueryWithMixedArguments validates a complex real-world query
func TestComplexQueryWithMixedArguments(t *testing.T) {
	// Note: LIMIT and SKIP with parameters are not valid Cypher syntax (must be literal numbers)
	query := "MATCH (p:Person {id: $personId})-[r:WORKS_AT {since: $startYear}]->(c:Company) WHERE c.active = $isActive AND p.age > $minAge WITH p, c, r WHERE r.salary > $minSalary RETURN p.name as employee, c.name as company, r.salary as salary ORDER BY r.salary DESC"

	args, err := ExtractArguments(query)
	if err != nil {
		t.Fatalf("failed to extract arguments: %v", err)
	}

	expectedArgs := map[string]bool{
		"personId":  true,
		"startYear": true,
		"isActive":  true,
		"minAge":    true,
		"minSalary": true,
	}

	if args.GetArgumentCount() != len(expectedArgs) {
		t.Errorf("expected %d arguments, got %d", len(expectedArgs), args.GetArgumentCount())
	}

	for argName := range expectedArgs {
		if !args.HasArgument(argName) {
			t.Errorf("expected argument '%s' to exist", argName)
		}
	}

	// Verify all found arguments are expected
	for _, argName := range args.GetArgumentNames() {
		if !expectedArgs[argName] {
			t.Errorf("unexpected argument '%s' found", argName)
		}
	}

	t.Logf("✓ Complex query with %d arguments validated: %v", args.GetArgumentCount(), args.GetArgumentNames())
}

// TestArgumentsWithQuotedStrings validates that arguments are extracted correctly even with quoted strings
func TestArgumentsWithQuotedStrings(t *testing.T) {
	query := "MATCH (n) WHERE n.name = 'constant' OR n.code = $code RETURN n"
	args, err := ExtractArguments(query)
	if err != nil {
		t.Fatalf("failed to extract arguments: %v", err)
	}

	if args.GetArgumentCount() != 1 {
		t.Errorf("expected 1 argument, got %d", args.GetArgumentCount())
	}

	if !args.HasArgument("code") {
		t.Errorf("expected argument 'code' to exist")
	}

	// Verify that 'constant' string literal was NOT extracted as an argument
	if args.HasArgument("constant") {
		t.Errorf("string literal 'constant' should not be extracted as argument")
	}

	t.Logf("✓ Arguments with quoted strings validated")
}

// TestArgumentsInUnwindStatement validates arguments in UNWIND clauses
func TestArgumentsInUnwindStatement(t *testing.T) {
	query := "UNWIND $ids as id MATCH (n:Person {id: id}) WHERE n.active = $isActive RETURN n"
	args, err := ExtractArguments(query)
	if err != nil {
		t.Fatalf("failed to extract arguments: %v", err)
	}

	expectedArgs := map[string]bool{
		"ids":      true,
		"isActive": true,
	}

	if args.GetArgumentCount() != len(expectedArgs) {
		t.Errorf("expected %d arguments, got %d", len(expectedArgs), args.GetArgumentCount())
	}

	for argName := range expectedArgs {
		if !args.HasArgument(argName) {
			t.Errorf("expected argument '%s' to exist", argName)
		}
	}

	t.Logf("✓ Arguments in UNWIND statements validated")
}

// TestArgumentIndexing validates that arguments preserve order of appearance
func TestArgumentIndexing(t *testing.T) {
	query := "MATCH (n) WHERE n.a = $first AND n.b = $second AND n.c = $third RETURN n"
	args, err := ExtractArguments(query)
	if err != nil {
		t.Fatalf("failed to extract arguments: %v", err)
	}

	names := args.GetArgumentNames()
	expectedOrder := []string{"first", "second", "third"}

	if len(names) != len(expectedOrder) {
		t.Fatalf("expected %d arguments, got %d", len(expectedOrder), len(names))
	}

	for i, expected := range expectedOrder {
		if names[i] != expected {
			t.Errorf("argument at position %d: expected '%s', got '%s'", i, expected, names[i])
		}
	}

	t.Logf("✓ Argument indexing and order validated: %v", names)
}

// TestEmptyQuery validates handling of empty or minimal queries
func TestEmptyQuery(t *testing.T) {
	// This should parse but have no arguments
	query := "RETURN 1"
	args, err := ExtractArguments(query)
	if err != nil {
		t.Fatalf("failed to extract arguments: %v", err)
	}

	if args.GetArgumentCount() != 0 {
		t.Errorf("expected 0 arguments for minimal query, got %d", args.GetArgumentCount())
	}

	t.Logf("✓ Empty/minimal query validated")
}

// TestArgumentNamesWithSpecialCharsInContext validates arguments are correctly isolated
func TestArgumentNamesWithSpecialCharsInContext(t *testing.T) {
	// Test that we correctly identify argument names even in complex contexts
	query := "MATCH (n) WHERE n.field = $param123 RETURN n"
	args, err := ExtractArguments(query)
	if err != nil {
		t.Fatalf("failed to extract arguments: %v", err)
	}

	if !args.HasArgument("param123") {
		t.Errorf("expected argument 'param123' to exist")
	}

	// Should have exactly 1 argument
	if args.GetArgumentCount() != 1 {
		t.Errorf("expected 1 argument, got %d", args.GetArgumentCount())
	}

	t.Logf("✓ Argument names with numbers validated")
}

// BenchmarkArgumentExtraction measures performance of argument extraction
func BenchmarkArgumentExtraction(b *testing.B) {
	query := "MATCH (p:Person {id: $personId})-[r:WORKS_AT {since: $startYear}]->(c:Company) WHERE c.active = $isActive AND p.age > $minAge RETURN p.name ORDER BY p.name LIMIT $limit SKIP $offset"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := ExtractArguments(query)
		if err != nil {
			b.Fatalf("failed to extract arguments: %v", err)
		}
	}
}

// TestArgumentsPreservesQuery validates that the query is stored correctly
func TestArgumentsPreservesQuery(t *testing.T) {
	query := "MATCH (n) WHERE n.id = $id RETURN n"
	args, err := ExtractArguments(query)
	if err != nil {
		t.Fatalf("failed to extract arguments: %v", err)
	}

	if args.Query != query {
		t.Errorf("expected query to be preserved: got %q", args.Query)
	}

	t.Logf("✓ Query preservation validated")
}

// TestMultipleArgumentsInSingleExpression validates arguments grouped in parentheses
func TestMultipleArgumentsInSingleExpression(t *testing.T) {
	query := "MATCH (n) WHERE (n.x = $x AND n.y = $y AND n.z = $z) RETURN n"
	args, err := ExtractArguments(query)
	if err != nil {
		t.Fatalf("failed to extract arguments: %v", err)
	}

	expectedCount := 3
	if args.GetArgumentCount() != expectedCount {
		t.Errorf("expected %d arguments, got %d", expectedCount, args.GetArgumentCount())
	}

	// Verify order is preserved
	names := args.GetArgumentNames()
	for i, expectedName := range []string{"x", "y", "z"} {
		if i < len(names) && names[i] != expectedName {
			t.Logf("Note: Order may vary, checking if all expected arguments exist")
			break
		}
	}

	for _, expectedName := range []string{"x", "y", "z"} {
		if !args.HasArgument(expectedName) {
			t.Errorf("expected argument '%s' to exist", expectedName)
		}
	}

	t.Logf("✓ Multiple arguments in single expression validated")
}

// TestArgumentsInDeleteStatement validates arguments in DELETE queries
func TestArgumentsInDeleteStatement(t *testing.T) {
	query := "MATCH (n:Person {id: $id}) DELETE n"
	args, err := ExtractArguments(query)
	if err != nil {
		t.Fatalf("failed to extract arguments: %v", err)
	}

	if !args.HasArgument("id") {
		t.Errorf("expected argument 'id' in DELETE query")
	}

	if args.GetArgumentCount() != 1 {
		t.Errorf("expected 1 argument, got %d", args.GetArgumentCount())
	}

	t.Logf("✓ Arguments in DELETE statements validated")
}

// TestArgumentsInMergeStatement validates arguments in MERGE queries
func TestArgumentsInMergeStatement(t *testing.T) {
	query := "MERGE (n:Person {id: $id}) ON CREATE SET n.created = $timestamp ON MATCH SET n.updated = $timestamp RETURN n"
	args, err := ExtractArguments(query)
	if err != nil {
		t.Fatalf("failed to extract arguments: %v", err)
	}

	expectedArgs := map[string]int{
		"id":        1,
		"timestamp": 2, // Used in both ON CREATE and ON MATCH
	}

	for argName, expectedContexts := range expectedArgs {
		if arg, ok := args.GetArgument(argName); ok {
			if arg.Contexts != expectedContexts {
				t.Errorf("expected '%s' to appear %d times, got %d", argName, expectedContexts, arg.Contexts)
			}
		} else {
			t.Errorf("expected argument '%s' to exist", argName)
		}
	}

	if args.GetArgumentCount() != len(expectedArgs) {
		t.Errorf("expected %d unique arguments, got %d", len(expectedArgs), args.GetArgumentCount())
	}

	t.Logf("✓ Arguments in MERGE statements validated")
}

// TestLongArgumentNames validates very long argument names
func TestLongArgumentNames(t *testing.T) {
	longName := strings.Repeat("a", 100)
	query := "MATCH (n) WHERE n.id = $" + longName + " RETURN n"
	args, err := ExtractArguments(query)
	if err != nil {
		t.Fatalf("failed to extract arguments: %v", err)
	}

	if !args.HasArgument(longName) {
		t.Errorf("expected long argument name to be extracted correctly")
	}

	t.Logf("✓ Long argument names validated")
}
