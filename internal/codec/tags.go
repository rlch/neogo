package codec

import (
	"reflect"
	"strings"
)

// FieldInfo contains parsed information about a struct field
type FieldInfo struct {
	Name     string    // Go field name
	DBName   string    // Database field name (from neo4j tag or derived)
	Meta     FieldMeta // Pre-computed field metadata
	Options  []string  // Additional options from neo4j tag
	IsSkip   bool      // Field should be skipped (neo4j:"-")
	IsCustom bool      // Has custom codec
	IsEmbed  bool      // Embedded struct
	Codec    string    // Name of custom codec
}

// parseNeo4jTag parses a neo4j struct tag
// Format: `neo4j:"field_name,option1,option2:value,option3"`
// Returns field name and options
func parseNeo4jTag(tag string) (string, []string) {
	if tag == "" {
		return "", nil
	}

	parts := strings.Split(tag, ",")
	if len(parts) == 0 {
		return "", nil
	}

	fieldName := strings.TrimSpace(parts[0])
	if fieldName == "-" {
		return "", []string{"skip"}
	}

	var options []string
	if len(parts) > 1 {
		for _, part := range parts[1:] {
			option := strings.TrimSpace(part)
			if option != "" {
				options = append(options, option)
			}
		}
	}

	return fieldName, options
}

// parseFieldInfo extracts field information from a reflect.StructField
func parseFieldInfo(field reflect.StructField) *FieldInfo {
	info := &FieldInfo{
		Name: field.Name,
		Meta: extractFieldMeta(field),
	}

	// Parse neo4j tag
	neo4jTag := field.Tag.Get("neo4j")
	dbName, options := parseNeo4jTag(neo4jTag)

	if dbName == "" {
		// If no neo4j tag, use field name converted to snake_case
		dbName = toSnakeCase(field.Name)
	}

	info.DBName = dbName
	info.Options = options

	// Process options
	for _, option := range options {
		switch {
		case option == "skip":
			info.IsSkip = true
		case option == "embed":
			info.IsEmbed = true
		case strings.HasPrefix(option, "codec:"):
			info.IsCustom = true
			info.Codec = strings.TrimPrefix(option, "codec:")
		}
	}

	// Anonymous (embedded) struct fields should be treated as embedded
	// even without explicit ",embed" tag. This handles cases like:
	//   type Human struct { BaseOrganism `neo4j:"Human"` }
	// where the tag specifies a Neo4j label but the field is still embedded in Go.
	if field.Anonymous && UnwindType(field.Type).Kind() == reflect.Struct {
		info.IsEmbed = true
	}

	return info
}

// toSnakeCase converts CamelCase to snake_case
func toSnakeCase(s string) string {
	var result strings.Builder

	for i, r := range s {
		if i > 0 && (r >= 'A' && r <= 'Z') {
			result.WriteByte('_')
		}
		if r >= 'A' && r <= 'Z' {
			result.WriteRune(r - 'A' + 'a')
		} else {
			result.WriteRune(r)
		}
	}

	return result.String()
}
