# neogo vs GORM Feature Comparison

This document compares neogo's capabilities against GORM (the most popular Go ORM) to contextualize neogo's approach.

## Philosophy Difference

| Aspect | GORM | neogo |
|--------|------|-------|
| Query Language | Query builder that generates SQL | Raw Cypher with type-safe binding |
| Abstraction Level | High (hides SQL) | Low (exposes Cypher) |
| Learning Curve | Learn GORM API | Know Cypher, minimal API |
| Control | Framework controls SQL | Developer controls Cypher |

**neogo's philosophy**: Write the Cypher you know, let neogo handle marshalling/unmarshalling.

## Feature Matrix

| Category | Feature | GORM | neogo | Notes |
|----------|---------|------|-------|-------|
| **Query Execution** |||||
|| Raw queries | ✅ | ✅ | neogo is raw-Cypher-first |
|| Parameterized queries | ✅ | ✅ | `RunWithParams()` |
|| Result binding | ✅ | ✅ | Name/pointer pairs |
|| Streaming results | ✅ | ✅ | `Stream()` / `StreamWithParams()` |
| **Schema/Migrations** |||||
|| Auto-migration | ✅ | ✅ | `Schema().AutoMigrate()` |
|| Index definitions via tags | ✅ | ✅ | `index`, `fulltext`, `text`, `point` |
|| Constraint definitions | ✅ | ✅ | `unique`, `notNull`, `nodeKey` |
|| Composite indexes | ✅ | ✅ | Via shared names |
|| Schema introspection | ✅ | ✅ | `GetIndexes()`, `GetConstraints()` |
| **Transactions** |||||
|| Manual transactions | ✅ | ✅ | `BeginTransaction()` |
|| Managed transactions | ✅ | ✅ | `ReadTransaction()`, `WriteTransaction()` |
|| Auto-commit | ✅ | ✅ | `Exec()` |
| **Sessions** |||||
|| Session management | ✅ | ✅ | `ReadSession()`, `WriteSession()` |
|| Connection pooling | ✅ | ✅ | Via neo4j driver |
|| Causal consistency | N/A | ✅ | Bookmark-based |
| **Type System** |||||
|| Struct mapping | ✅ | ✅ | Via `neo4j` tags |
|| Custom types | ✅ | ✅ | `Valuer` interface |
|| Embedded structs | ✅ | ✅ | Node/Relationship base types |
|| Abstract types | ❌ | ✅ | Polymorphic nodes |
| **NOT in neogo** |||||
|| Query builder | ✅ | ❌ | Write raw Cypher instead |
|| Hooks/callbacks | ✅ | ❌ | Handle in application code |
|| Eager loading (Preload) | ✅ | ❌ | Write relationship queries |
|| Scopes | ✅ | ❌ | Compose Cypher strings |
|| Soft delete | ✅ | ❌ | Implement as property pattern |

Legend: ✅ Supported | ❌ Not implemented | N/A Not applicable

## What neogo Does Well

### 1. **Raw Cypher with Safety**
No DSL to learn - use the Cypher you know:
```go
var people []Person
err := driver.Exec().
    Cypher(`
        MATCH (p:Person)-[:KNOWS*1..3]->(friend:Person)
        WHERE p.id = $id AND friend.age > $minAge
        WITH friend, count(*) AS connectionDepth
        ORDER BY connectionDepth DESC
        LIMIT 10
        RETURN friend
    `).
    RunWithParams(ctx, map[string]any{"id": id, "minAge": 21}, "friend", &people)
```

### 2. **Type-Safe Result Binding**
Simple name/pointer pairs:
```go
var person Person
var count int64
err := c.Cypher(`... RETURN p, cnt`).
    RunWithParams(ctx, params, "p", &person, "cnt", &count)
```

### 3. **Schema from Tags**
Define once, migrate automatically:
```go
type Person struct {
    internal.Node `neo4j:"Person"`
    Email string `neo4j:"email,unique"`
    Name  string `neo4j:"name,index"`
}

// Auto-create indexes/constraints
actions, _ := driver.Schema().AutoMigrate(ctx)
```

### 4. **Neo4j-Specific Features**
- Abstract nodes (polymorphic types)
- Causal consistency with bookmarks
- Relationship structs with properties
- Full Cypher support (APOC, graph algorithms, etc.)

## What neogo Doesn't Do

neogo intentionally omits features that would:
- Generate Cypher (you write it)
- Hide Neo4j concepts behind abstractions
- Add runtime overhead for rarely-used features

**For these patterns, use application code:**

```go
// Soft delete - just a property pattern
err := c.Cypher(`
    MATCH (p:Person {id: $id})
    SET p.deletedAt = datetime(), p.deleted = true
`).RunWithParams(ctx, params)

// Eager loading - explicit relationship query
err := c.Cypher(`
    MATCH (p:Person {id: $id})-[:KNOWS]->(friend:Person)
    RETURN p, collect(friend) AS friends
`).RunWithParams(ctx, params, "p", &person, "friends", &friends)

// Hooks - call functions before/after
func CreatePerson(ctx context.Context, c neogo.Client, p *Person) error {
    p.CreatedAt = time.Now()  // Before hook
    err := c.Cypher(`CREATE (p:Person) SET p = $props`).
        RunWithParams(ctx, map[string]any{"props": p})
    if err == nil {
        sendWelcomeEmail(p)  // After hook
    }
    return err
}
```

## When to Use What

| Use Case | Recommendation |
|----------|----------------|
| Complex graph traversals | neogo (raw Cypher) |
| Simple CRUD on nodes | Either works |
| Relational data | GORM |
| Graph algorithms | neogo (access to GDS) |
| Rapid prototyping | GORM (more helpers) |
| Production Neo4j | neogo (full control) |
