# Parser Package

This package provides Cypher query parsing using ANTLR-generated parser and lexer.

## Files

### Core Parsing
- **`parser.go`** - Main API for parsing Cypher queries
  - `ParseCypherQuery(query string)` - Parse and return parse tree
  - `ParseWithMetadata(query string)` - Parse and return metadata
  - `ParseErrorListener` - Custom error handling

### Argument & Return Tracking
- **`argument_tracker.go`** - Extract function arguments ($x, $y, etc.)
  - `ExtractArguments(query string)` - Extract all parameters from query
  - `QueryArguments` - Holds argument information with indexing and lookup
  - Supports named parameters ($name), numeric placeholders ($1, $2)
  - Tracks parameter usage count and position

- **`return_tracker.go`** - Extract RETURN clause items
  - `ExtractReturnVariables(query string)` - Extract all RETURN items
  - `QueryReturnInfo` - Holds return item information with aliases
  - Detects aggregate functions (count, sum, avg, min, max, collect, etc.)
  - Supports RETURN DISTINCT and wildcard returns (RETURN *)
  - Maps items by alias and expression for fast lookup

### Generated Code (in `grammar/` directory)
- `cypher_parser.go` - Generated parser from ANTLR
- `cypher_lexer.go` - Generated lexer from ANTLR
- `cypherparser_*_visitor.go` - Visitor pattern support
- `cypherparser_*_listener.go` - Listener pattern support

### Testing

#### Structure Tests (Validate AST)
- **`parser_structure_test.go`** - Validates exact parse tree structure for various queries
  - `TestSimpleMatchReturnStructure` - Basic MATCH-RETURN
  - `TestMatchWithWhereStructure` - WHERE clauses
  - `TestMatchRelationshipStructure` - Relationships
  - `TestReturnMultipleItemsStructure` - Multiple RETURN items
  - `TestCreateNodeStructure` - CREATE statements
  - `TestDeleteStatementStructure` - DELETE statements
  - `TestOrderByStructure` - ORDER BY clauses
  - `TestLimitSkipStructure` - LIMIT and SKIP
  - `TestSetStatementStructure` - SET statements
  - `TestExpressionPrecedenceStructure` - Operator precedence
  - `TestComparisonOperatorsStructure` - All comparison operators
  - `TestPropertyAccessStructure` - Property access chains
  - `TestNodeLabelsStructure` - Node label specifications
  - `TestMapLiteralStructure` - Map literals in properties

#### Argument Tracking Tests
- **`argument_tracker_test.go`** - Comprehensive argument extraction tests (20+ tests)
  - Simple and multiple arguments
  - Repeated argument usage and counting
  - Arguments across multiple clauses (WHERE, WITH, etc.)
  - Complex argument names (underscores, camelCase, UPPERCASE)
  - Numeric placeholders ($1, $2, etc.)
  - Arguments in CREATE, SET, DELETE, MERGE, UNWIND statements
  - Arguments in relationships and expressions
  - Long argument names
  - Benchmark testing

#### Return Variable Tracking Tests
- **`return_tracker_test.go`** - Comprehensive return variable extraction tests (30+ tests)
  - Simple and multiple return variables
  - Aliases (RETURN x as y)
  - Property access (n.name, r.since)
  - RETURN DISTINCT
  - Wildcard returns (RETURN *)
  - Aggregate functions (count, sum, avg, min, max, collect)
  - Complex expressions and function calls
  - CASE expressions
  - Simple variable extraction
  - Return item lookup by alias and expression
  - Benchmark testing

#### Smoke Tests
- **`parser_test.go`** - Basic smoke tests confirming queries parse without error
  - `TestParseCypherQuery` - 9 different query types
  - `TestParseWithMetadata` - Metadata parsing
  - `BenchmarkParseCypherQuery` - Performance benchmark



## Usage

### Parse a Query
```go
import "github.com/rlch/neogo/parser"

tree, err := parser.ParseCypherQuery("MATCH (n) RETURN n")
if err != nil {
    log.Fatal(err)
}
```

### Extract Arguments (Parameters)
```go
query := "MATCH (n) WHERE n.id = $id AND n.status = $status RETURN n"
args, err := parser.ExtractArguments(query)
if err != nil {
    log.Fatal(err)
}

// Get argument information
fmt.Println(args.GetArgumentCount())      // 2
fmt.Println(args.GetArgumentNames())      // [id status]
fmt.Println(args.HasArgument("id"))       // true
fmt.Println(args.HasArgument("unknown"))  // false

if arg, ok := args.GetArgument("id"); ok {
    fmt.Println(arg.Name)      // "id"
    fmt.Println(arg.Contexts)  // 1 (appears once)
}
```

### Extract Return Variables
```go
query := "MATCH (p:Person) RETURN p.id as id, p.name as name, count(*) as total"
result, err := parser.ExtractReturnVariables(query)
if err != nil {
    log.Fatal(err)
}

// Get return information
fmt.Println(result.GetReturnItemCount())    // 3
fmt.Println(result.GetReturnExpressions())  // [p.id, p.name, count(*)]
fmt.Println(result.GetReturnAliases())      // [id, name, total]
fmt.Println(result.IsReturnDistinct)        // false

// Find aggregates
aggregates := result.GetAggregateItems()
fmt.Println(len(aggregates))                // 1

// Lookup by alias
if item, ok := result.GetReturnItemByAlias("name"); ok {
    fmt.Println(item.Expression)            // "p.name"
}

// Get simple variables only (no functions or property access)
simpleVars := result.GetSimpleVariables()
```

### Validate Structure
```go
scriptCtx, ok := tree.(*cypher.ScriptContext)
if !ok {
    log.Fatal("expected ScriptContext")
}

query := scriptCtx.Query()
// ... continue validation
```

### Walk the Tree
```go
// Use type assertions to work with specific context types
if whereCtx, ok := tree.(*cypher.WhereContext); ok {
    expr := whereCtx.Expression()
    // ... work with expression
}
```

## Testing Approaches

### Approach 1: Structure Validation
Use type-safe context getters to validate the exact shape of the AST.

See: `parser_structure_test.go`

```go
scriptCtx := tree.(*cypher.ScriptContext)
queryCtx := scriptCtx.Query()
regularQueryCtx := queryCtx.RegularQuery()
```

**Best for:** Confirming that a query parses with the expected structure

### Approach 2: Tree Walking
Walk the tree to find and count specific node types.

See: `parser_structure_test.go` (helper functions)

```go
foundWhere := false
walkTree := func(node antlr.ParseTree) {
    if _, ok := node.(*cypher.WhereContext); ok {
        foundWhere = true
    }
    // ... walk children
}
walkTree(tree)
```

**Best for:** Finding specific constructs anywhere in the tree



## Running Tests

```bash
# All parser tests
go test -v ./parser

# Structure tests only
go test -v ./parser -run Structure

# Specific test
go test -v ./parser -run TestSimpleMatchReturnStructure

# With coverage
go test -cover ./parser

# Benchmark
go test -bench=. ./parser
```

## Grammar

The parser is generated from two ANTLR4 grammar files:

- `grammar/CypherLexer.g4` - Lexer rules (tokens)
- `grammar/CypherParser.g4` - Parser rules (syntax)

Both are based on the official Cypher grammar from the Neo4j repository.

## Key Concepts

### Parse Tree Nodes

The parse tree is a hierarchy of **rule contexts** (for grammar rules) and **terminal nodes** (for tokens).

### Rule Contexts
Generated classes representing grammar rules, with type-safe getters:
- `ScriptContext`, `QueryContext`, `MatchStContext`, etc.
- Each has methods like `.Query()`, `.Expression()`, `.MATCH()`, etc.
- Methods return nil if the element is optional and not present

### Terminal Nodes
Represent tokens (keywords, identifiers, literals)
- Accessed via methods like `.MATCH()`, `.RETURN()`, etc.
- Or through `GetToken(tokenType, index)`



## Context Types Available

**Statement Contexts:**
`ScriptContext`, `QueryContext`, `RegularQueryContext`, `SingleQueryContext`, `MatchStContext`, `CreateStContext`, `DeleteStContext`, `SetStContext`, `MergeStContext`, `RemoveStContext`, `ReturnStContext`, `WithStContext`, `UnwindStContext`

**Pattern Contexts:**
`PatternContext`, `PatternPartContext`, `PatternElemContext`, `NodePatternContext`, `RelationshipPatternContext`, `RelationshipTypesContext`

**Expression Contexts:**
`ExpressionContext`, `ComparisonExpressionContext`, `AndExpressionContext`, `OrExpressionContext`, `XorExpressionContext`, `PropertyExpressionContext`, `AtomicExpressionContext`, `LiteralContext`

**Projection Contexts:**
`ProjectionBodyContext`, `ProjectionItemsContext`, `ProjectionItemContext`, `OrderStContext`, `OrderItemContext`, `LimitStContext`, `SkipStContext`

See `grammar/cypher_parser.go` for complete type definitions.

## Resources

- **ANTLR Documentation:** https://www.antlr.org/
- **ANTLR Go Runtime:** https://github.com/antlr4-go/antlr/v4
- **Cypher Grammar:** https://github.com/neo4j-documentation/cypher-ebnf
- **Testing Guide:** See `../ANTLR_TESTING_GUIDE.md`
- **Test Improvements:** See `../PARSER_TEST_IMPROVEMENTS.md`
