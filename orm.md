# neogo ORM Development Notes

This document tracks design decisions and implementation plans for neogo's ORM features.

---

## Implementation Progress

### Phase 1: Tag Parsing ✅ COMPLETE
- [x] `IndexSpec` and `ConstraintSpec` types in `internal/codec/schema.go`
- [x] `IndexDef` and `ConstraintDef` types for aggregated definitions
- [x] `parseIndexSpec()` - parses index options from tags
- [x] `parseConstraintSpec()` - parses constraint options from tags
- [x] Updated `FieldInfo` with Index/Constraint fields
- [x] Updated `parseFieldInfo()` to process schema options
- [x] `AggregateIndexes()` - groups fields into composite indexes
- [x] `AggregateConstraints()` - groups fields into composite constraints
- [x] `GenerateIndexCypher()` - generates CREATE INDEX statements
- [x] `GenerateConstraintCypher()` - generates CREATE CONSTRAINT statements
- [x] Comprehensive tests for all parsing and generation

### Phase 2: Registry Integration ✅ COMPLETE
- [x] `CodecRegistry` is single source of truth for all type data
- [x] `nodeMeta`/`relMeta` maps store Neo4j-specific metadata
- [x] `Neo4jNodeMetadata.Schema` field with aggregated indexes/constraints
- [x] `RelationshipStructMeta.Schema` field for relationship schema
- [x] Schema extracted during `RegisterTypes()` in one pass
- [x] Auto-add `unique_{Label}_id` constraint for nodes with ID field
- [x] `RegisteredNode`/`RegisteredRelationship` are thin wrappers with delegation
- [x] `Labels()`, `FieldsToProps()`, `Schema()`, `Type()` delegation methods

### Phase 3: Schema Interface ✅ COMPLETE
- [x] Add `Schema` interface to Driver (`schema.go`)
- [x] Implement `GetIndexes()` - SHOW INDEXES introspection
- [x] Implement `GetConstraints()` - SHOW CONSTRAINTS introspection
- [x] Implement `AutoMigrate()` - additive schema migration (returns executed actions)
- [x] Implement `NeedsMigration()` - check if migration needed
- [x] `IndexInfo` and `ConstraintInfo` types for DB schema representation
- [x] `MigrationAction` type for pending/executed changes
- [x] Implementation in `schema_impl.go`

### Phase 4: Testing ✅ COMPLETE
- [x] Unit tests for Phase 2 (Schema delegation) in `internal/registry_test.go`:
  - [x] `TestRegisteredNodeSchema` - tests Schema() delegation for nodes
  - [x] `TestRegisteredRelationshipSchema` - tests Schema() delegation for relationships
  - [x] `TestRegisteredAbstractNodeSchema` - tests Schema() for abstract nodes
  - [x] `TestSchemaWithNestedLabels` - tests schema with inheritance
- [x] Unit tests for Phase 3 (Migration logic) in `schema_test.go`:
  - [x] `TestMigrationLogic_*` - comprehensive migration logic tests
  - [x] `TestSchemaFromRegistry` - tests schema collection from registry
  - [x] `TestMigrationActionTypes` - tests action type constants
  - [x] `TestIndexInfo/TestConstraintInfo` - tests info types
  - [x] `TestIndexTypeConstants/TestConstraintTypeConstants` - tests re-exported constants
  - [x] Edge case tests: empty DB, partial migration, fully migrated, additive only, etc.
- [x] Integration tests with real Neo4j in `schema_integration_test.go`:
  - [x] `TestSchemaIntegration_GetIndexes` - SHOW INDEXES parsing
  - [x] `TestSchemaIntegration_GetConstraints` - SHOW CONSTRAINTS parsing
  - [x] `TestSchemaIntegration_AutoMigrate` - CREATE INDEX/CONSTRAINT execution
    - Basic node schema (unique, index, notNull)
    - Fulltext indexes
    - Composite indexes
    - Node key constraints
    - Relationship indexes/constraints
    - Idempotency (running twice succeeds)
  - [x] `TestSchemaIntegration_NeedsMigration` - migration detection
  - [x] `TestSchemaIntegration_RoundTrip` - full migration cycle
  - Uses testcontainers-go with Neo4j Enterprise for full feature coverage

---

## Schema Management Design

### Overview

The biggest gap in neogo compared to GORM is schema management. This includes:
- Index definitions via struct tags
- Constraint definitions (unique, notNull, nodeKey)
- AutoMigrate() functionality
- Schema introspection (SHOW INDEXES, SHOW CONSTRAINTS)

### Neo4j 5.x Schema Features

#### Index Types

| Type | Cypher Syntax | Use Case |
|------|--------------|----------|
| **Range** | `CREATE INDEX name FOR (n:Label) ON (n.prop)` | Default for equality/range queries |
| **Text** | `CREATE TEXT INDEX name FOR (n:Label) ON (n.prop)` | String prefix/suffix/contains |
| **Point** | `CREATE POINT INDEX name FOR (n:Label) ON (n.prop)` | Spatial/geographic queries |
| **Fulltext** | `CREATE FULLTEXT INDEX name FOR (n:Label) ON EACH [n.prop]` | Advanced text search |
| **Vector** | `CREATE VECTOR INDEX name FOR (n:Label) ON (n.embedding) OPTIONS {...}` | ML embeddings (5.11+) |

**Notes:**
- Range, Text, Point support composite indexes (multiple properties)
- Fulltext uses `ON EACH [...]` syntax, supports multiple labels
- Vector requires `OPTIONS {vector.dimensions: N, vector.similarity_function: 'cosine'}`

#### Constraint Types

| Type | Cypher Syntax | Use Case |
|------|--------------|----------|
| **Unique** | `CREATE CONSTRAINT name FOR (n:Label) REQUIRE n.prop IS UNIQUE` | Unique property values |
| **Node Key** | `CREATE CONSTRAINT name FOR (n:Label) REQUIRE (n.prop1, n.prop2) IS NODE KEY` | Unique + Existence (composite) |
| **Not Null** | `CREATE CONSTRAINT name FOR (n:Label) REQUIRE n.prop IS NOT NULL` | Property existence |

**Notes:**
- Node Key implies both uniqueness AND existence for all properties in the key
- Relationship constraints also exist for relationship properties

---

### Tag Syntax Design

#### Design Principles

1. **Consistency**: Follow existing `neo4j` tag conventions
2. **Discoverability**: Simple tags for common cases, options for advanced
3. **Neo4j-native**: Map directly to Neo4j concepts (not SQL abstractions)
4. **Composable**: Support composite indexes via shared names

#### Proposed Tag Format

```
neo4j:"fieldName[,option1[,option2[:value][,option3]]]"
```

#### Index Tags

```go
type Person struct {
    neogo.Node `neo4j:"Person"`
    
    // Simple range index (most common)
    Email string `neo4j:"email,index"`
    
    // Named index
    Username string `neo4j:"username,index:idx_person_username"`
    
    // Specific index type
    Bio string `neo4j:"bio,index:text"`           // Text index
    Location Point `neo4j:"location,index:point"` // Point index
    
    // Fulltext index
    Description string `neo4j:"description,fulltext"`
    
    // Vector index (for embeddings)
    Embedding []float32 `neo4j:"embedding,vector:1536"`  // 1536 dimensions
    
    // Composite index (shared name with priority)
    FirstName string `neo4j:"first_name,index:idx_person_name,priority:1"`
    LastName  string `neo4j:"last_name,index:idx_person_name,priority:2"`
}
```

#### Constraint Tags

```go
type Person struct {
    neogo.Node `neo4j:"Person"`
    
    // Unique constraint
    Email string `neo4j:"email,unique"`
    
    // Named unique constraint
    SSN string `neo4j:"ssn,unique:uniq_person_ssn"`
    
    // Not null constraint
    Name string `neo4j:"name,notNull"`
    
    // Node key (unique + not null, typically composite)
    TenantID string `neo4j:"tenant_id,nodeKey:key_person_tenant"`
    PersonID string `neo4j:"person_id,nodeKey:key_person_tenant"`
    
    // Combine constraints
    Code string `neo4j:"code,unique,notNull"`
}
```

#### Full Example

```go
type Movie struct {
    neogo.Node `neo4j:"Movie"`
    
    // ID field (inherited from Node, already indexed by default?)
    // Or explicit: ID string `neo4j:"id,unique"`
    
    // Unique title within a year (composite unique)
    Title string `neo4j:"title,unique:uniq_movie_title_year,priority:1"`
    Year  int    `neo4j:"year,unique:uniq_movie_title_year,priority:2,index"`
    
    // Full-text search on plot
    Plot string `neo4j:"plot,fulltext:ft_movie_plot"`
    
    // Required fields
    Director string `neo4j:"director,notNull,index"`
    
    // Vector embeddings for similarity search
    PlotEmbedding []float32 `neo4j:"plot_embedding,vector:1536,similarity:cosine"`
}
```

---

### Internal Representation

#### Schema Metadata Types

```go
// IndexType represents Neo4j index types
type IndexType string

const (
    IndexTypeRange    IndexType = "RANGE"    // Default
    IndexTypeText     IndexType = "TEXT"
    IndexTypePoint    IndexType = "POINT"
    IndexTypeFulltext IndexType = "FULLTEXT"
    IndexTypeVector   IndexType = "VECTOR"
)

// ConstraintType represents Neo4j constraint types
type ConstraintType string

const (
    ConstraintTypeUnique  ConstraintType = "UNIQUE"
    ConstraintTypeNodeKey ConstraintType = "NODE_KEY"
    ConstraintTypeNotNull ConstraintType = "NOT_NULL"
)

// IndexDef represents an index definition parsed from tags
type IndexDef struct {
    Name       string            // Auto-generated or explicit
    Type       IndexType         // RANGE, TEXT, POINT, FULLTEXT, VECTOR
    Label      string            // Node label (from struct)
    Properties []IndexProperty   // Ordered by priority
    Options    map[string]string // Type-specific options (dimensions, similarity, etc.)
}

type IndexProperty struct {
    Name     string // DB property name
    Priority int    // For composite index ordering
}

// ConstraintDef represents a constraint definition parsed from tags
type ConstraintDef struct {
    Name       string         // Auto-generated or explicit
    Type       ConstraintType // UNIQUE, NODE_KEY, NOT_NULL
    Label      string         // Node label
    Properties []string       // Property names (composite for NODE_KEY)
}

// SchemaMeta holds all schema metadata for a registered type
type SchemaMeta struct {
    TypeName    string
    Labels      []string
    Indexes     []IndexDef
    Constraints []ConstraintDef
}
```

#### Integration with Existing Code

Current flow:
1. `parseNeo4jTag()` in `internal/codec/tags.go` → returns `(fieldName, []options)`
2. `parseFieldInfo()` → creates `FieldInfo` with processed options
3. `RegisterNode()` in `internal/registry.go` → stores in `RegisteredNode`

New flow additions:
1. Extend `FieldInfo` with schema metadata:
   ```go
   type FieldInfo struct {
       // ... existing fields ...
       Index      *IndexSpec      // nil if no index
       Constraint *ConstraintSpec // nil if no constraint
   }
   
   type IndexSpec struct {
       Name     string    // Empty = auto-generate
       Type     IndexType // Default = RANGE
       Priority int       // For composite (default 10)
       Options  map[string]string
   }
   
   type ConstraintSpec struct {
       Name string         // Empty = auto-generate
       Type ConstraintType
   }
   ```

2. Aggregate in `RegisteredNode`:
   ```go
   type RegisteredNode struct {
       // ... existing fields ...
       Schema *SchemaMeta // Aggregated indexes/constraints
   }
   ```

3. New `Schema` interface on Driver:
   ```go
   type Schema interface {
       // Generate CREATE INDEX/CONSTRAINT statements
       AutoMigrate(ctx context.Context, types ...any) error
       
       // Get current schema state
       GetIndexes(ctx context.Context) ([]IndexInfo, error)
       GetConstraints(ctx context.Context) ([]ConstraintInfo, error)
       
       // Check if migration needed
       NeedsMigration(ctx context.Context) (bool, error)
   }
   ```

---

### GORM Lessons Learned

Key insights from GORM's implementation:

1. **Two-Level Parsing**: Base parser (`ParseTagSetting`) + type-specific parser (`parseFieldIndexes`)

2. **Composite Index Merging**: Fields with same index name are aggregated, sorted by priority:
   ```go
   indexesByName[index.Name] = &Index{...}
   idx.Fields = append(idx.Fields, ...)
   sort.Slice(idx.Fields, func(i, j int) bool {
       return idx.Fields[i].Priority < idx.Fields[j].Priority
   })
   ```

3. **Naming Strategy**: Interface for customizable auto-naming:
   ```go
   type Namer interface {
       IndexName(label, property string) string  // "idx_person_email"
   }
   ```

4. **Settings Map**: Parse options into map, apply defaults:
   ```go
   settings := ParseTagSetting(tagSetting, ",")
   priority, _ := strconv.Atoi(settings["PRIORITY"])
   if err != nil { priority = 10 }
   ```

---

### Implementation Plan

#### Phase 1: Tag Parsing (extends existing code)

1. **Update `parseNeo4jTag()`** to recognize schema options:
   - `index`, `index:name`, `index:type`, `index:name:type`
   - `unique`, `unique:name`
   - `notNull`, `notNull:name`
   - `nodeKey:name`
   - `fulltext`, `fulltext:name`
   - `vector:dimensions`, `vector:dimensions:similarity`
   - `priority:N`

2. **Update `FieldInfo`** struct with `IndexSpec` and `ConstraintSpec`

3. **Add schema aggregation** in `RegisterNode()` to build `SchemaMeta`

#### Phase 2: Cypher Generation

1. **Add `GenerateIndexCypher()`** method:
   ```go
   func (idx IndexDef) GenerateCypher() string {
       // CREATE INDEX idx_person_email FOR (n:Person) ON (n.email)
       // CREATE FULLTEXT INDEX ft_movie_plot FOR (n:Movie) ON EACH [n.plot]
       // CREATE VECTOR INDEX vec_movie_embed FOR (n:Movie) ON (n.embedding) OPTIONS {...}
   }
   ```

2. **Add `GenerateConstraintCypher()`** method:
   ```go
   func (c ConstraintDef) GenerateCypher() string {
       // CREATE CONSTRAINT uniq_person_email FOR (n:Person) REQUIRE n.email IS UNIQUE
       // CREATE CONSTRAINT key_person_tenant FOR (n:Person) REQUIRE (n.tenant_id, n.person_id) IS NODE KEY
   }
   ```

#### Phase 3: Schema Interface

1. **Add `Driver.Schema()`** method returning `Schema` interface

2. **Implement `AutoMigrate()`**:
   - Get current indexes/constraints from DB
   - Generate missing ones
   - Execute CREATE statements

3. **Implement introspection**:
   ```cypher
   SHOW INDEXES YIELD name, type, labelsOrTypes, properties, options
   SHOW CONSTRAINTS YIELD name, type, labelsOrTypes, properties
   ```

#### Phase 4: Testing

1. Unit tests for tag parsing
2. Integration tests for Cypher generation
3. Integration tests with Neo4j for AutoMigrate

---

### Design Decisions

1. **Auto-index on Node.ID?** ✅ YES - The base `Node` type's `ID` field automatically gets a unique constraint.

2. **Relationship indexes?** ✅ YES - Support relationship property indexes:
   ```go
   type ActedIn struct {
       neogo.Relationship `neo4j:"ACTED_IN"`
       Role string `neo4j:"role,index"`  // Index on relationship property
   }
   ```

3. **Drop behavior?** ❌ NO - AutoMigrate should NOT drop indexes/constraints not in code. It's additive-only for safety. Users can manually drop if needed.

4. **Migration versioning?** Deferred - Not in initial implementation. Can add later if needed.

---

### Naming Conventions

#### Brainstorming

**Option A: Verbose/Descriptive (GORM-style)**
```
idx_person_email           # index
idx_person_first_name_last_name  # composite index
uniq_person_email          # unique constraint
nn_person_name             # not null
key_person_tenant_id_person_id   # node key
ft_movie_plot              # fulltext
vec_movie_embedding        # vector
txt_person_bio             # text index
pt_location_coords         # point index
```
- ✅ Clear what each is
- ✅ Matches GORM conventions
- ❌ Long names for composites
- ❌ Inconsistent prefixes (idx vs uniq vs nn vs key)

**Option B: Type Prefix Consistency**
```
idx_range_person_email
idx_text_person_bio
idx_point_location_coords
idx_fulltext_movie_plot
idx_vector_movie_embedding
con_unique_person_email
con_notnull_person_name
con_nodekey_person_tenant_id_person_id
```
- ✅ Very explicit about type
- ✅ Consistent idx_/con_ prefix
- ❌ Very verbose
- ❌ Neo4j has 252 char limit but still ugly

**Option C: Short Prefixes**
```
i_person_email             # range index (default)
it_person_bio              # text index
ip_location_coords         # point index
ift_movie_plot             # fulltext index
iv_movie_embedding         # vector index
cu_person_email            # constraint unique
cn_person_name             # constraint not null
ck_person_tenant_person    # constraint node key
```
- ✅ Short
- ❌ Cryptic, hard to remember
- ❌ `i_` vs `it_` confusing

**Option D: Neo4j Native Style**
Neo4j's auto-generated names use patterns like:
- `constraint_unique_Person_email`
- `index_Person_email`

Adapt to:
```
neogo_idx_Person_email
neogo_idx_Person_firstName_lastName
neogo_unique_Person_email
neogo_notnull_Person_name
neogo_nodekey_Person_tenantId_personId
neogo_fulltext_Movie_plot
neogo_vector_Movie_embedding
neogo_text_Person_bio
neogo_point_Location_coords
```
- ✅ Clear namespace (neogo_)
- ✅ Preserves case from Go (Person not person)
- ✅ Type is explicit
- ❌ Longer than Option A

**Option E: Hybrid (RECOMMENDED)**
```
# Indexes - prefix by type
idx_Person_email                    # range (default)
text_Person_bio                     # text
point_Location_coords               # point
fulltext_Movie_plot                 # fulltext
vector_Movie_embedding              # vector

# Constraints - prefix by constraint type
unique_Person_email                 # unique
notnull_Person_name                 # not null
nodekey_Person_tenantId_personId    # node key

# Composite - join with underscore
idx_Person_firstName_lastName
unique_Movie_title_year
```

**Reasoning for Option E:**
1. **Index prefix reflects what it IS** - When you see `text_Person_bio` you know it's a text index
2. **Constraint prefix reflects what it DOES** - `unique_Person_email` clearly enforces uniqueness
3. **CamelCase labels** - Matches Neo4j convention (Person not person)
4. **snake_case properties** - Matches our DB naming convention
5. **Composites are obvious** - Multiple properties joined with underscore

#### Final Convention (Option E)

| Schema Type | Pattern | Example |
|-------------|---------|---------|
| Range Index | `idx_{Label}_{prop}` | `idx_Person_email` |
| Text Index | `text_{Label}_{prop}` | `text_Person_bio` |
| Point Index | `point_{Label}_{prop}` | `point_Location_coords` |
| Fulltext Index | `fulltext_{Label}_{prop}` | `fulltext_Movie_plot` |
| Vector Index | `vector_{Label}_{prop}` | `vector_Movie_embedding` |
| Unique Constraint | `unique_{Label}_{prop}` | `unique_Person_email` |
| Not Null Constraint | `notnull_{Label}_{prop}` | `notnull_Person_name` |
| Node Key Constraint | `nodekey_{Label}_{props}` | `nodekey_Person_tenant_id_person_id` |
| Rel Range Index | `idx_{RelType}_{prop}` | `idx_ACTED_IN_role` |
| Rel Unique Constraint | `unique_{RelType}_{prop}` | `unique_WORKED_AT_employee_id` |

**Composite naming**: Properties joined with `_` in order of priority
- `idx_Person_first_name_last_name` (firstName priority:1, lastName priority:2)

**Special case - Node.ID**: 
- Auto-generated: `unique_{Label}_id` for each registered node type

---

### References

- Neo4j Indexes: https://neo4j.com/docs/cypher-manual/current/indexes/
- Neo4j Constraints: https://neo4j.com/docs/cypher-manual/current/constraints/
- GORM Schema: https://github.com/go-gorm/gorm/tree/master/schema
- ORM Feature Comparison: ./ORM_FEATURE_COMPARISON.md
