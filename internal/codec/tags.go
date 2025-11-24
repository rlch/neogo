package codec

import (
	"reflect"
	"strings"
)

// FieldInfo contains parsed information about a struct field
type FieldInfo struct {
	Name     string    // Go field name
	DBName   string    // Database field name (from db tag or derived)
	Meta     FieldMeta // Pre-computed field metadata
	Options  []string  // Additional options from db tag
	IsSkip   bool      // Field should be skipped (db:"-")
	IsCustom bool      // Has custom codec
	IsEmbed  bool      // Embedded struct
	Codec    string    // Name of custom codec
}

// parseDBTag parses a db struct tag
// Format: `db:"field_name,option1,option2:value,option3"`
// Returns field name and options
func parseDBTag(tag string) (string, []string) {
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

	// Parse db tag
	dbTag := field.Tag.Get("db")
	dbName, options := parseDBTag(dbTag)

	if dbName == "" {
		// If no db tag, use field name converted to snake_case
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
