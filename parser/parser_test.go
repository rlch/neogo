package parser

import (
	"testing"
)

func TestParseCypherQuery(t *testing.T) {
	testCases := []struct {
		name        string
		query       string
		shouldParse bool
	}{
		{
			name:        "simple match return",
			query:       "MATCH (n) RETURN n",
			shouldParse: true,
		},
		{
			name:        "match with where clause",
			query:       "MATCH (n:Person) WHERE n.age > 30 RETURN n",
			shouldParse: true,
		},
		{
			name:        "match relationship",
			query:       "MATCH (a)-[r]->(b) RETURN a, b",
			shouldParse: true,
		},
		{
			name:        "create node",
			query:       "CREATE (n:Person {name: 'John'}) RETURN n",
			shouldParse: true,
		},
		{
			name:        "delete",
			query:       "MATCH (n) DELETE n",
			shouldParse: true,
		},
		{
			name:        "match with limit",
			query:       "MATCH (n) RETURN n LIMIT 10",
			shouldParse: true,
		},
		{
			name:        "match with order by",
			query:       "MATCH (n) RETURN n ORDER BY n.name ASC",
			shouldParse: true,
		},
		{
			name:        "set node property",
			query:       "MATCH (n) SET n.name = 'Alice' RETURN n",
			shouldParse: true,
		},
		{
			name:        "complex query",
			query:       "MATCH (a:Person)-[r:KNOWS]->(b:Person) WHERE a.age > 25 AND b.age < 60 RETURN a.name, b.name, r.since ORDER BY r.since DESC LIMIT 10",
			shouldParse: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			tree, err := ParseCypherQuery(tc.query)
			if !tc.shouldParse {
				if err == nil {
					t.Errorf("expected error but got none")
				}
				return
			}
			if err != nil {
				t.Errorf("failed to parse: %v", err)
				return
			}
			if tree == nil {
				t.Errorf("expected parse tree but got nil")
				return
			}
			t.Logf("✓ Parsed successfully: %s", tc.query)
		})
	}
}

func TestParseWithMetadata(t *testing.T) {
	query := "MATCH (n:Person) WHERE n.age > 30 RETURN n.name"
	parsed, err := ParseWithMetadata(query)
	if err != nil {
		t.Fatalf("failed to parse: %v", err)
	}

	if parsed.Query != query {
		t.Errorf("expected query %q, got %q", query, parsed.Query)
	}

	if parsed.Tree == nil {
		t.Errorf("expected parse tree but got nil")
	}

	if len(parsed.RuleNames) == 0 {
		t.Errorf("expected rule names but got empty list")
	}

	t.Logf("✓ Parsed with metadata: %d rule names available", len(parsed.RuleNames))
}

func BenchmarkParseCypherQuery(b *testing.B) {
	query := "MATCH (a:Person)-[r:KNOWS]->(b:Person) WHERE a.age > 25 RETURN a.name, b.name"
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := ParseCypherQuery(query)
		if err != nil {
			b.Fatalf("failed to parse: %v", err)
		}
	}
}
