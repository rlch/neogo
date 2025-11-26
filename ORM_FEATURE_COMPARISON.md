# neogo vs GORM Feature Comparison

This document compares neogo's current capabilities against GORM (the most popular Go ORM) to identify gaps and potential improvements.

## Feature Matrix

| Category | Feature | GORM | neogo | Notes |
|----------|---------|------|-------|-------|
| **Schema/Migrations** |||||
|| Auto-migration | ✅ | ❌ | GORM creates/updates tables automatically |
|| Index definitions via tags | ✅ | ❌ | `gorm:"index"`, `gorm:"uniqueIndex"` |
|| Constraint definitions | ✅ | ❌ | `gorm:"unique"`, `gorm:"check:age>0"` |
|| Composite indexes | ✅ | ❌ | |
|| Foreign key constraints | ✅ | ❌ | Neo4j doesn't have FK, but has constraints |
|| Schema introspection | ✅ | ❌ | |
| **CRUD Helpers** |||||
|| `Create()` | ✅ | ⚠️ | neogo has Cypher `Create()`, not model-level |
|| `Save()` (upsert) | ✅ | ⚠️ | neogo has `Merge()` but no simple Save |
|| `First()` / `Find()` | ✅ | ❌ | neogo requires full Cypher |
|| `Update()` / `Updates()` | ✅ | ⚠️ | neogo has `Set()` but no model-level update |
|| `Delete()` | ✅ | ⚠️ | neogo has Cypher Delete, not model-level |
|| Batch insert | ✅ | ❌ | `CreateInBatches()` |
|| Upsert | ✅ | ⚠️ | neogo has Merge |
| **Associations** |||||
|| Belongs To | ✅ | ❌ | |
|| Has One | ✅ | ⚠️ | Via relationship structs |
|| Has Many | ✅ | ⚠️ | Via `[]*T` slice |
|| Many To Many | ✅ | ⚠️ | Via relationship structs |
|| Eager loading (Preload) | ✅ | ❌ | `Preload("Orders")` |
|| Association mode | ✅ | ❌ | `Append()`, `Replace()`, `Clear()` |
|| Nested create | ✅ | ❌ | |
| **Hooks/Callbacks** |||||
|| BeforeCreate | ✅ | ❌ | |
|| AfterCreate | ✅ | ❌ | |
|| BeforeUpdate | ✅ | ❌ | |
|| AfterUpdate | ✅ | ❌ | |
|| BeforeSave | ✅ | ❌ | |
|| AfterSave | ✅ | ❌ | |
|| BeforeDelete | ✅ | ❌ | |
|| AfterDelete | ✅ | ❌ | |
|| BeforeFind | ✅ | ❌ | |
|| AfterFind | ✅ | ❌ | |
| **Query Features** |||||
|| Raw SQL/Cypher | ✅ | ✅ | `Cypher()` method |
|| Query builder | ✅ | ✅ | Full Cypher DSL |
|| Scopes | ✅ | ❌ | Reusable query conditions |
|| Pagination | ✅ | ⚠️ | `.Skip().Limit()` exists |
|| Sorting | ✅ | ✅ | `.OrderBy()` |
|| Grouping | ✅ | ⚠️ | Via Cypher, no helper |
|| Joins | ✅ | ✅ | Pattern matching |
|| Subqueries | ✅ | ✅ | `Subquery()` |
|| Named arguments | ✅ | ✅ | Parameters |
| **Soft Delete** |||||
|| Soft delete support | ✅ | ❌ | `gorm.DeletedAt` |
|| Unscoped queries | ✅ | ❌ | |
| **Transactions** |||||
|| Manual transactions | ✅ | ✅ | `BeginTransaction()` |
|| Auto transactions | ✅ | ✅ | `ReadTransaction()`, `WriteTransaction()` |
|| Savepoints | ✅ | ❌ | Nested transactions |
| **Logging/Debug** |||||
|| Query logging | ✅ | ⚠️ | `.Print()` exists |
|| Slow query log | ✅ | ❌ | |
|| Dry run | ✅ | ⚠️ | `.Print()` |
|| Explain | ✅ | ❌ | Neo4j has EXPLAIN/PROFILE |
| **Advanced** |||||
|| Connection pooling | ✅ | ✅ | Via neo4j driver |
|| Context support | ✅ | ✅ | |
|| Prepared statements | ✅ | ❌ | |
|| Plugins | ✅ | ❌ | |
|| Custom types | ✅ | ✅ | `Valuer` interface |
|| Serializer | ✅ | ✅ | Zero-reflection codec |

Legend: ✅ Full support | ⚠️ Partial/different approach | ❌ Not implemented

---

## Priority 1: Schema Management (HIGH IMPACT)

### 1.1 Index Creation

Neo4j supports several index types. neogo should support them via struct tags:

```go
type Person struct {
    neogo.Node `neo4j:"Person"`
    
    ID    string `neo4j:"id,index"`           // CREATE INDEX FOR (p:Person) ON (p.id)
    Email string `neo4j:"email,unique"`       // CREATE CONSTRAINT FOR (p:Person) REQUIRE p.email IS UNIQUE
    Name  string `neo4j:"name,fulltext"`      // CREATE FULLTEXT INDEX FOR (p:Person) ON EACH [p.name]
}
```

**Neo4j Index Types:**
- Range indexes (default, for equality/range queries)
- Text indexes (for string prefix/suffix/contains)
- Point indexes (for spatial queries)
- Full-text indexes (for text search)
- Token lookup indexes (for label/type lookups)

### 1.2 Constraints

```go
type Person struct {
    neogo.Node `neo4j:"Person"`
    
    Email string `neo4j:"email,unique"`                     // Unique constraint
    Age   int    `neo4j:"age,check:value >= 0"`             // Property existence/value constraint
    Name  string `neo4j:"name,notNull"`                     // NOT NULL constraint (Neo4j 5.x)
}
```

**Neo4j Constraints:**
- Unique node property constraints
- Unique relationship property constraints
- Node property existence constraints
- Relationship property existence constraints
- Node key constraints (composite unique)

### 1.3 Migration API

```go
// Auto-migrate (create indexes/constraints if not exist)
driver.AutoMigrate(&Person{}, &Movie{})

// Or explicit migration
driver.Migrate().
    CreateIndex("Person", "id").
    CreateUniqueConstraint("Person", "email").
    Run(ctx)

// Check schema
schema := driver.Schema()
indexes := schema.GetIndexes(ctx)
constraints := schema.GetConstraints(ctx)
```

---

## Priority 2: Model-Level CRUD (MEDIUM IMPACT)

Currently neogo requires full Cypher queries. High-level helpers would simplify common operations:

### 2.1 Repository Pattern

```go
type PersonRepo struct {
    neogo.Repository[Person]
}

// Find by ID
person, err := repo.FindByID(ctx, "123")

// Find all
people, err := repo.FindAll(ctx)

// Find with conditions
people, err := repo.Find(ctx, db.Where("age", ">", 18))

// Create
person := &Person{Name: "John", Age: 30}
err := repo.Create(ctx, person)

// Save (upsert)
err := repo.Save(ctx, person)

// Delete
err := repo.Delete(ctx, person)
```

### 2.2 Chainable Query Builder (GORM-style)

```go
var people []Person
err := driver.Model(&Person{}).
    Where("age > ?", 18).
    Order("name").
    Limit(10).
    Find(&people)
```

**Consideration:** This may conflict with neogo's explicit Cypher philosophy. Perhaps offer both:
- `driver.Exec()` - Full Cypher DSL (current)
- `driver.Model()` - GORM-style helpers (new)

---

## Priority 3: Hooks/Callbacks (MEDIUM IMPACT)

```go
type Person struct {
    neogo.Node `neo4j:"Person"`
    
    ID        string    `neo4j:"id"`
    Name      string    `neo4j:"name"`
    CreatedAt time.Time `neo4j:"created_at"`
    UpdatedAt time.Time `neo4j:"updated_at"`
}

// Implement hook interfaces
func (p *Person) BeforeCreate() error {
    p.CreatedAt = time.Now()
    p.UpdatedAt = time.Now()
    return nil
}

func (p *Person) BeforeUpdate() error {
    p.UpdatedAt = time.Now()
    return nil
}

func (p *Person) AfterFind() error {
    // Post-processing after fetch
    return nil
}
```

**Hook Interface Definitions:**
```go
type BeforeCreator interface { BeforeCreate() error }
type AfterCreator interface { AfterCreate() error }
type BeforeUpdater interface { BeforeUpdate() error }
type AfterUpdater interface { AfterUpdate() error }
type BeforeSaver interface { BeforeSave() error }
type AfterSaver interface { AfterSave() error }
type BeforeDeleter interface { BeforeDelete() error }
type AfterDeleter interface { AfterDelete() error }
type AfterFinder interface { AfterFind() error }
```

---

## Priority 4: Eager Loading / Preload (MEDIUM IMPACT)

```go
type Person struct {
    neogo.Node `neo4j:"Person"`
    
    Name   string           `neo4j:"name"`
    Movies neogo.Many[Movie] `neo4j:"ACTED_IN>"`  // Outgoing relationships
}

// Current: Manual pattern matching
var person Person
var movies []*Movie
driver.Exec().
    Match(db.Node(&person).To(ActedIn{}, db.Qual(&movies, "movies"))).
    Return(&person, &movies).
    Run(ctx)

// Proposed: Preload syntax
var person Person
driver.Exec().
    Match(db.Node(&person)).
    Preload("Movies").  // Auto-generates relationship pattern
    Return(&person).
    Run(ctx)

// Or via Model API
driver.Model(&Person{}).
    Preload("Movies").
    First(&person, "id = ?", "123")
```

---

## Priority 5: Scopes (LOW IMPACT)

Reusable query conditions:

```go
// Define scopes
func Active(q neogo.Querier) neogo.Querier {
    return q.Where(db.Prop("status"), "=", "active")
}

func Adult(q neogo.Querier) neogo.Querier {
    return q.Where(db.Prop("age"), ">=", 18)
}

// Use scopes
driver.Exec().
    Match(db.Node(&person)).
    Scope(Active).
    Scope(Adult).
    Return(&person).
    Run(ctx)
```

---

## Priority 6: Soft Delete (LOW IMPACT)

```go
type Person struct {
    neogo.Node `neo4j:"Person"`
    neogo.SoftDelete  // Adds DeletedAt field
    
    Name string `neo4j:"name"`
}

// Delete sets deleted_at instead of removing
driver.Delete(&person)

// Queries automatically filter deleted
driver.Model(&Person{}).Find(&people)  // WHERE deleted_at IS NULL

// Include deleted
driver.Model(&Person{}).Unscoped().Find(&people)

// Permanently delete
driver.Model(&Person{}).Unscoped().Delete(&person)
```

---

## Implementation Roadmap

### Phase 1: Schema Management (Highest Value)
1. Parse index/constraint tags during registration
2. Add `Schema()` method to Driver
3. Implement `AutoMigrate()` 
4. Implement `GetIndexes()`, `GetConstraints()`

### Phase 2: Hooks
1. Define hook interfaces
2. Check for interface implementation during registration
3. Call hooks in Encode/Decode paths

### Phase 3: Model-Level CRUD
1. Add `Repository[T]` generic type
2. Implement `FindByID`, `FindAll`, `Create`, `Save`, `Delete`
3. Keep existing Cypher DSL as primary API

### Phase 4: Eager Loading
1. Add `Preload()` method to query builder
2. Auto-generate relationship patterns
3. Handle nested preloads

### Phase 5: Scopes & Soft Delete
1. Add `Scope()` method
2. Add `SoftDelete` embedded type
3. Auto-filter in queries

---

## Neo4j-Specific Features (Beyond GORM)

neogo should also support Neo4j-specific features that GORM doesn't have:

### Graph-Specific
- [ ] Path finding (shortestPath, allShortestPaths)
- [ ] Graph algorithms integration (GDS library)
- [ ] Relationship traversal depth control
- [ ] Pattern comprehension helpers

### Neo4j 5.x Features
- [ ] VECTOR indexes (for embeddings/AI)
- [ ] Node key constraints
- [ ] Relationship key constraints
- [ ] Enhanced SHOW commands

### Performance
- [x] Zero-reflection hot path (DONE)
- [ ] Query plan caching
- [ ] Batch operations (UNWIND patterns)
- [ ] Async operations

---

## Tag Syntax Proposal

Unified tag syntax for all features:

```go
type Person struct {
    neogo.Node `neo4j:"Person"`
    
    // Basic field
    Name string `neo4j:"name"`
    
    // With index
    Email string `neo4j:"email,index"`
    
    // With unique constraint  
    Username string `neo4j:"username,unique"`
    
    // With full-text index
    Bio string `neo4j:"bio,fulltext"`
    
    // Multiple options
    Age int `neo4j:"age,index,check:value>=0"`
    
    // Skip field
    Internal string `neo4j:"-"`
    
    // Relationships (using zero-cost One/Many types)
    BestFriend neogo.One[Person]  `neo4j:"BEST_FRIEND>"`  // Single outgoing relationship
    Friends    neogo.Many[Person] `neo4j:"FRIENDS>"`      // Multiple outgoing relationships
}
```

**Supported Options:**
- `index` - Create range index
- `unique` - Unique constraint
- `fulltext` - Full-text index  
- `text` - Text index
- `point` - Point index (for spatial)
- `notNull` - NOT NULL constraint
- `check:expr` - Property value constraint
- `rel=TYPE` - Relationship type (existing)
- `dir=->|<-` - Relationship direction (existing)
