package codec

import (
	"fmt"
	"sort"
	"strconv"
	"strings"
)

// IndexType represents Neo4j index types
type IndexType string

const (
	IndexTypeRange    IndexType = "RANGE"
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

// IndexSpec contains index specification parsed from a field tag
type IndexSpec struct {
	Name     string            // Empty = auto-generate
	Type     IndexType         // Default = RANGE
	Priority int               // For composite indexes (default 10)
	Options  map[string]string // Type-specific options (e.g., dimensions, similarity)
}

// ConstraintSpec contains constraint specification parsed from a field tag
type ConstraintSpec struct {
	Name     string         // Empty = auto-generate
	Type     ConstraintType // UNIQUE, NODE_KEY, NOT_NULL
	Priority int            // For composite constraints (e.g., nodeKey)
}

// IndexDef represents a complete index definition (aggregated from fields)
type IndexDef struct {
	Name       string            // Auto-generated or explicit
	Type       IndexType         // RANGE, TEXT, POINT, FULLTEXT, VECTOR
	Label      string            // Node label or relationship type
	IsNode     bool              // true for node index, false for relationship index
	Properties []IndexProperty   // Ordered by priority
	Options    map[string]string // Type-specific options
}

// IndexProperty represents a property within an index
type IndexProperty struct {
	Name     string // DB property name
	Priority int    // For composite index ordering
}

// ConstraintDef represents a complete constraint definition (aggregated from fields)
type ConstraintDef struct {
	Name       string         // Auto-generated or explicit
	Type       ConstraintType // UNIQUE, NODE_KEY, NOT_NULL
	Label      string         // Node label or relationship type
	IsNode     bool           // true for node constraint, false for relationship constraint
	Properties []string       // Property names (ordered by priority for composites)
}

// SchemaMeta holds all schema metadata for a registered type
type SchemaMeta struct {
	TypeName    string          // Go type name
	Labels      []string        // Neo4j labels (for nodes)
	RelType     string          // Relationship type (for relationships)
	IsNode      bool            // true for node, false for relationship
	Indexes     []IndexDef      // Aggregated indexes
	Constraints []ConstraintDef // Aggregated constraints
}

// parseIndexSpec parses index-related options from a tag option string.
// Formats supported:
//   - "index"                    -> Range index, auto-name
//   - "index:name"               -> Range index with explicit name
//   - "index:text"               -> Text index, auto-name
//   - "index:name:text"          -> Text index with explicit name
//   - "fulltext"                 -> Fulltext index, auto-name
//   - "fulltext:name"            -> Fulltext index with explicit name
//   - "vector:1536"              -> Vector index with dimensions
//   - "vector:1536:cosine"       -> Vector index with dimensions and similarity
//   - "vector:name:1536:cosine"  -> Vector index with name, dimensions, similarity
func parseIndexSpec(option string) *IndexSpec {
	spec := &IndexSpec{
		Type:     IndexTypeRange,
		Priority: 10, // Default priority
		Options:  make(map[string]string),
	}

	// Handle different index tag formats
	switch {
	case option == "index":
		// Simple range index
		return spec

	case strings.HasPrefix(option, "index:"):
		parts := strings.Split(strings.TrimPrefix(option, "index:"), ":")
		return parseIndexParts(spec, parts)

	case option == "fulltext":
		spec.Type = IndexTypeFulltext
		return spec

	case strings.HasPrefix(option, "fulltext:"):
		spec.Type = IndexTypeFulltext
		spec.Name = strings.TrimPrefix(option, "fulltext:")
		return spec

	case strings.HasPrefix(option, "vector:"):
		spec.Type = IndexTypeVector
		parts := strings.Split(strings.TrimPrefix(option, "vector:"), ":")
		return parseVectorParts(spec, parts)

	case strings.HasPrefix(option, "text:"):
		spec.Type = IndexTypeText
		spec.Name = strings.TrimPrefix(option, "text:")
		return spec

	case option == "text":
		spec.Type = IndexTypeText
		return spec

	case strings.HasPrefix(option, "point:"):
		spec.Type = IndexTypePoint
		spec.Name = strings.TrimPrefix(option, "point:")
		return spec

	case option == "point":
		spec.Type = IndexTypePoint
		return spec
	}

	return nil
}

// parseIndexParts handles index:name or index:name:type or index:type formats
func parseIndexParts(spec *IndexSpec, parts []string) *IndexSpec {
	if len(parts) == 0 {
		return spec
	}

	// Check if first part is a type
	if idxType := parseIndexType(parts[0]); idxType != "" {
		spec.Type = idxType
		if len(parts) > 1 {
			spec.Name = parts[1]
		}
		return spec
	}

	// First part is a name
	spec.Name = parts[0]
	if len(parts) > 1 {
		if idxType := parseIndexType(parts[1]); idxType != "" {
			spec.Type = idxType
		}
	}

	return spec
}

// parseIndexType converts a string to IndexType if valid
func parseIndexType(s string) IndexType {
	switch strings.ToLower(s) {
	case "range":
		return IndexTypeRange
	case "text":
		return IndexTypeText
	case "point":
		return IndexTypePoint
	case "fulltext":
		return IndexTypeFulltext
	case "vector":
		return IndexTypeVector
	}
	return ""
}

// parseVectorParts handles vector:dims or vector:dims:similarity or vector:name:dims:similarity
func parseVectorParts(spec *IndexSpec, parts []string) *IndexSpec {
	if len(parts) == 0 {
		return spec
	}

	// Try to parse first part as dimensions
	if dims, err := strconv.Atoi(parts[0]); err == nil {
		spec.Options["dimensions"] = strconv.Itoa(dims)
		if len(parts) > 1 {
			spec.Options["similarity"] = parts[1]
		}
		return spec
	}

	// First part is a name
	spec.Name = parts[0]
	if len(parts) > 1 {
		if dims, err := strconv.Atoi(parts[1]); err == nil {
			spec.Options["dimensions"] = strconv.Itoa(dims)
		}
	}
	if len(parts) > 2 {
		spec.Options["similarity"] = parts[2]
	}

	return spec
}

// parseConstraintSpec parses constraint-related options from a tag option string.
// Formats supported:
//   - "unique"           -> Unique constraint, auto-name
//   - "unique:name"      -> Unique constraint with explicit name
//   - "notNull"          -> Not null constraint, auto-name
//   - "notNull:name"     -> Not null constraint with explicit name
//   - "nodeKey:name"     -> Node key constraint (must have name for composite)
func parseConstraintSpec(option string) *ConstraintSpec {
	spec := &ConstraintSpec{
		Priority: 10, // Default priority
	}

	switch {
	case option == "unique":
		spec.Type = ConstraintTypeUnique
		return spec

	case strings.HasPrefix(option, "unique:"):
		spec.Type = ConstraintTypeUnique
		spec.Name = strings.TrimPrefix(option, "unique:")
		return spec

	case option == "notNull" || option == "notnull" || option == "not_null":
		spec.Type = ConstraintTypeNotNull
		return spec

	case strings.HasPrefix(option, "notNull:") || strings.HasPrefix(option, "notnull:") || strings.HasPrefix(option, "not_null:"):
		spec.Type = ConstraintTypeNotNull
		// Handle all variants
		for _, prefix := range []string{"notNull:", "notnull:", "not_null:"} {
			if strings.HasPrefix(option, prefix) {
				spec.Name = strings.TrimPrefix(option, prefix)
				break
			}
		}
		return spec

	case strings.HasPrefix(option, "nodeKey:") || strings.HasPrefix(option, "nodekey:") || strings.HasPrefix(option, "node_key:"):
		spec.Type = ConstraintTypeNodeKey
		// Handle all variants
		for _, prefix := range []string{"nodeKey:", "nodekey:", "node_key:"} {
			if strings.HasPrefix(option, prefix) {
				spec.Name = strings.TrimPrefix(option, prefix)
				break
			}
		}
		return spec
	}

	return nil
}

// parsePriority extracts priority value from a "priority:N" option
func parsePriority(option string) (int, bool) {
	if strings.HasPrefix(option, "priority:") {
		if p, err := strconv.Atoi(strings.TrimPrefix(option, "priority:")); err == nil {
			return p, true
		}
	}
	return 0, false
}

// GenerateIndexName generates an auto-name for an index
func GenerateIndexName(idxType IndexType, labelOrType string, properties []string) string {
	propStr := strings.Join(properties, "_")

	switch idxType {
	case IndexTypeRange:
		return fmt.Sprintf("idx_%s_%s", labelOrType, propStr)
	case IndexTypeText:
		return fmt.Sprintf("text_%s_%s", labelOrType, propStr)
	case IndexTypePoint:
		return fmt.Sprintf("point_%s_%s", labelOrType, propStr)
	case IndexTypeFulltext:
		return fmt.Sprintf("fulltext_%s_%s", labelOrType, propStr)
	case IndexTypeVector:
		return fmt.Sprintf("vector_%s_%s", labelOrType, propStr)
	default:
		return fmt.Sprintf("idx_%s_%s", labelOrType, propStr)
	}
}

// GenerateConstraintName generates an auto-name for a constraint
func GenerateConstraintName(conType ConstraintType, labelOrType string, properties []string) string {
	propStr := strings.Join(properties, "_")

	switch conType {
	case ConstraintTypeUnique:
		return fmt.Sprintf("unique_%s_%s", labelOrType, propStr)
	case ConstraintTypeNotNull:
		return fmt.Sprintf("notnull_%s_%s", labelOrType, propStr)
	case ConstraintTypeNodeKey:
		return fmt.Sprintf("nodekey_%s_%s", labelOrType, propStr)
	default:
		return fmt.Sprintf("constraint_%s_%s", labelOrType, propStr)
	}
}

// GenerateIndexCypher generates the CREATE INDEX Cypher statement
func (idx *IndexDef) GenerateIndexCypher() string {
	if len(idx.Properties) == 0 {
		return ""
	}

	// Build property list
	var props []string
	for _, p := range idx.Properties {
		if idx.IsNode {
			props = append(props, fmt.Sprintf("n.%s", p.Name))
		} else {
			props = append(props, fmt.Sprintf("r.%s", p.Name))
		}
	}

	var pattern string
	if idx.IsNode {
		pattern = fmt.Sprintf("(n:%s)", idx.Label)
	} else {
		pattern = fmt.Sprintf("()-[r:%s]-()", idx.Label)
	}

	switch idx.Type {
	case IndexTypeRange:
		return fmt.Sprintf("CREATE INDEX %s IF NOT EXISTS FOR %s ON (%s)",
			idx.Name, pattern, strings.Join(props, ", "))

	case IndexTypeText:
		return fmt.Sprintf("CREATE TEXT INDEX %s IF NOT EXISTS FOR %s ON (%s)",
			idx.Name, pattern, strings.Join(props, ", "))

	case IndexTypePoint:
		return fmt.Sprintf("CREATE POINT INDEX %s IF NOT EXISTS FOR %s ON (%s)",
			idx.Name, pattern, strings.Join(props, ", "))

	case IndexTypeFulltext:
		// Fulltext uses different syntax: ON EACH [props]
		var eachProps []string
		for _, p := range idx.Properties {
			if idx.IsNode {
				eachProps = append(eachProps, fmt.Sprintf("n.%s", p.Name))
			} else {
				eachProps = append(eachProps, fmt.Sprintf("r.%s", p.Name))
			}
		}
		return fmt.Sprintf("CREATE FULLTEXT INDEX %s IF NOT EXISTS FOR %s ON EACH [%s]",
			idx.Name, pattern, strings.Join(eachProps, ", "))

	case IndexTypeVector:
		// Vector index requires OPTIONS
		dims := idx.Options["dimensions"]
		if dims == "" {
			dims = "1536" // Default
		}
		similarity := idx.Options["similarity"]
		if similarity == "" {
			similarity = "cosine" // Default
		}
		return fmt.Sprintf("CREATE VECTOR INDEX %s IF NOT EXISTS FOR %s ON (%s) OPTIONS {indexConfig: {`vector.dimensions`: %s, `vector.similarity_function`: '%s'}}",
			idx.Name, pattern, props[0], dims, similarity)
	}

	return ""
}

// GenerateConstraintCypher generates the CREATE CONSTRAINT Cypher statement
func (c *ConstraintDef) GenerateConstraintCypher() string {
	if len(c.Properties) == 0 {
		return ""
	}

	var pattern string
	var propRefs []string

	if c.IsNode {
		pattern = fmt.Sprintf("(n:%s)", c.Label)
		for _, p := range c.Properties {
			propRefs = append(propRefs, fmt.Sprintf("n.%s", p))
		}
	} else {
		pattern = fmt.Sprintf("()-[r:%s]-()", c.Label)
		for _, p := range c.Properties {
			propRefs = append(propRefs, fmt.Sprintf("r.%s", p))
		}
	}

	switch c.Type {
	case ConstraintTypeUnique:
		if len(c.Properties) == 1 {
			return fmt.Sprintf("CREATE CONSTRAINT %s IF NOT EXISTS FOR %s REQUIRE %s IS UNIQUE",
				c.Name, pattern, propRefs[0])
		}
		// Composite unique
		return fmt.Sprintf("CREATE CONSTRAINT %s IF NOT EXISTS FOR %s REQUIRE (%s) IS UNIQUE",
			c.Name, pattern, strings.Join(propRefs, ", "))

	case ConstraintTypeNotNull:
		// NOT NULL is per-property, so generate one for each (though typically single)
		if len(c.Properties) == 1 {
			return fmt.Sprintf("CREATE CONSTRAINT %s IF NOT EXISTS FOR %s REQUIRE %s IS NOT NULL",
				c.Name, pattern, propRefs[0])
		}
		// Multiple properties - generate combined (requires all)
		return fmt.Sprintf("CREATE CONSTRAINT %s IF NOT EXISTS FOR %s REQUIRE (%s) IS NOT NULL",
			c.Name, pattern, strings.Join(propRefs, ", "))

	case ConstraintTypeNodeKey:
		// Node key is always wrapped in parentheses
		return fmt.Sprintf("CREATE CONSTRAINT %s IF NOT EXISTS FOR %s REQUIRE (%s) IS NODE KEY",
			c.Name, pattern, strings.Join(propRefs, ", "))
	}

	return ""
}

// AggregateIndexes collects field-level index specs into complete IndexDefs.
// Groups by index name for composite indexes and sorts properties by priority.
func AggregateIndexes(labelOrType string, isNode bool, fields []FieldSchemaInfo) []IndexDef {
	indexesByName := make(map[string]*IndexDef)
	var indexOrder []string // Preserve insertion order

	for _, f := range fields {
		if f.Index == nil {
			continue
		}

		spec := f.Index
		name := spec.Name
		if name == "" {
			// Auto-generate name for non-composite indexes
			name = GenerateIndexName(spec.Type, labelOrType, []string{f.DBName})
		}

		existing, found := indexesByName[name]
		if !found {
			existing = &IndexDef{
				Name:       name,
				Type:       spec.Type,
				Label:      labelOrType,
				IsNode:     isNode,
				Properties: []IndexProperty{},
				Options:    spec.Options,
			}
			indexesByName[name] = existing
			indexOrder = append(indexOrder, name)
		}

		// Add property with priority
		existing.Properties = append(existing.Properties, IndexProperty{
			Name:     f.DBName,
			Priority: spec.Priority,
		})
	}

	// Sort properties by priority within each index
	for _, idx := range indexesByName {
		sort.Slice(idx.Properties, func(i, j int) bool {
			return idx.Properties[i].Priority < idx.Properties[j].Priority
		})
	}

	// Build result in insertion order
	result := make([]IndexDef, 0, len(indexOrder))
	for _, name := range indexOrder {
		result = append(result, *indexesByName[name])
	}

	return result
}

// AggregateConstraints collects field-level constraint specs into complete ConstraintDefs.
// Groups by constraint name for composite constraints (like nodeKey) and sorts by priority.
func AggregateConstraints(labelOrType string, isNode bool, fields []FieldSchemaInfo) []ConstraintDef {
	constraintsByName := make(map[string]*constraintBuilder)
	var constraintOrder []string // Preserve insertion order

	for _, f := range fields {
		if f.Constraint == nil {
			continue
		}

		spec := f.Constraint
		name := spec.Name
		if name == "" {
			// Auto-generate name for non-composite constraints
			name = GenerateConstraintName(spec.Type, labelOrType, []string{f.DBName})
		}

		existing, found := constraintsByName[name]
		if !found {
			existing = &constraintBuilder{
				def: ConstraintDef{
					Name:   name,
					Type:   spec.Type,
					Label:  labelOrType,
					IsNode: isNode,
				},
			}
			constraintsByName[name] = existing
			constraintOrder = append(constraintOrder, name)
		}

		// Add property with priority
		existing.props = append(existing.props, propWithPriority{
			name:     f.DBName,
			priority: spec.Priority,
		})
	}

	// Sort properties by priority and build result
	result := make([]ConstraintDef, 0, len(constraintOrder))
	for _, name := range constraintOrder {
		builder := constraintsByName[name]
		sort.Slice(builder.props, func(i, j int) bool {
			return builder.props[i].priority < builder.props[j].priority
		})

		// Extract property names in sorted order
		for _, p := range builder.props {
			builder.def.Properties = append(builder.def.Properties, p.name)
		}
		result = append(result, builder.def)
	}

	return result
}

// Helper types for aggregation
type constraintBuilder struct {
	def   ConstraintDef
	props []propWithPriority
}

type propWithPriority struct {
	name     string
	priority int
}

// FieldSchemaInfo is a minimal struct for aggregation input
type FieldSchemaInfo struct {
	DBName     string
	Index      *IndexSpec
	Constraint *ConstraintSpec
}
