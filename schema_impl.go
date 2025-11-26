package neogo

import (
	"context"
	"fmt"
	"reflect"

	"github.com/neo4j/neo4j-go-driver/v5/neo4j"

	"github.com/rlch/neogo/internal"
	"github.com/rlch/neogo/internal/codec"
)

// schemaImpl implements the Schema interface.
type schemaImpl struct {
	db       neo4j.DriverWithContext
	registry *internal.Registry
}

// newSchema creates a new Schema implementation.
func newSchema(db neo4j.DriverWithContext, registry *internal.Registry) Schema {
	return &schemaImpl{
		db:       db,
		registry: registry,
	}
}

// GetIndexes returns all indexes from the database.
func (s *schemaImpl) GetIndexes(ctx context.Context) ([]IndexInfo, error) {
	session := s.db.NewSession(ctx, neo4j.SessionConfig{AccessMode: neo4j.AccessModeRead})
	defer session.Close(ctx)

	result, err := session.Run(ctx,
		`SHOW INDEXES YIELD name, type, entityType, labelsOrTypes, properties, state, options
		 WHERE type <> 'LOOKUP'`,
		nil,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to query indexes: %w", err)
	}

	var indexes []IndexInfo
	for result.Next(ctx) {
		record := result.Record()

		name, _ := record.Get("name")
		typ, _ := record.Get("type")
		entityType, _ := record.Get("entityType")
		labelsOrTypes, _ := record.Get("labelsOrTypes")
		properties, _ := record.Get("properties")
		state, _ := record.Get("state")
		options, _ := record.Get("options")

		idx := IndexInfo{
			Name:       asString(name),
			Type:       asString(typ),
			EntityType: asString(entityType),
			State:      asString(state),
		}

		if labels, ok := labelsOrTypes.([]any); ok {
			idx.LabelsOrTypes = make([]string, len(labels))
			for i, l := range labels {
				idx.LabelsOrTypes[i] = asString(l)
			}
		}

		if props, ok := properties.([]any); ok {
			idx.Properties = make([]string, len(props))
			for i, p := range props {
				idx.Properties[i] = asString(p)
			}
		}

		if opts, ok := options.(map[string]any); ok {
			idx.Options = opts
		}

		indexes = append(indexes, idx)
	}

	if err := result.Err(); err != nil {
		return nil, fmt.Errorf("error iterating indexes: %w", err)
	}

	return indexes, nil
}

// GetConstraints returns all constraints from the database.
func (s *schemaImpl) GetConstraints(ctx context.Context) ([]ConstraintInfo, error) {
	session := s.db.NewSession(ctx, neo4j.SessionConfig{AccessMode: neo4j.AccessModeRead})
	defer session.Close(ctx)

	result, err := session.Run(ctx,
		`SHOW CONSTRAINTS YIELD name, type, entityType, labelsOrTypes, properties`,
		nil,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to query constraints: %w", err)
	}

	var constraints []ConstraintInfo
	for result.Next(ctx) {
		record := result.Record()

		name, _ := record.Get("name")
		typ, _ := record.Get("type")
		entityType, _ := record.Get("entityType")
		labelsOrTypes, _ := record.Get("labelsOrTypes")
		properties, _ := record.Get("properties")

		con := ConstraintInfo{
			Name:       asString(name),
			Type:       asString(typ),
			EntityType: asString(entityType),
		}

		if labels, ok := labelsOrTypes.([]any); ok {
			con.LabelsOrTypes = make([]string, len(labels))
			for i, l := range labels {
				con.LabelsOrTypes[i] = asString(l)
			}
		}

		if props, ok := properties.([]any); ok {
			con.Properties = make([]string, len(props))
			for i, p := range props {
				con.Properties[i] = asString(p)
			}
		}

		constraints = append(constraints, con)
	}

	if err := result.Err(); err != nil {
		return nil, fmt.Errorf("error iterating constraints: %w", err)
	}

	return constraints, nil
}

// AutoMigrate creates missing indexes and constraints for registered types.
func (s *schemaImpl) AutoMigrate(ctx context.Context, types ...any) ([]MigrationAction, error) {
	// Get pending actions
	needsMigration, actions, err := s.NeedsMigration(ctx, types...)
	if err != nil {
		return nil, err
	}

	if !needsMigration {
		return nil, nil
	}

	// Execute each action
	session := s.db.NewSession(ctx, neo4j.SessionConfig{AccessMode: neo4j.AccessModeWrite})
	defer session.Close(ctx)

	var executed []MigrationAction
	for _, action := range actions {
		_, err := session.Run(ctx, action.Cypher, nil)
		if err != nil {
			return executed, fmt.Errorf("failed to execute %s for %s: %w", action.Type, action.Name, err)
		}
		executed = append(executed, action)
	}

	return executed, nil
}

// NeedsMigration checks if any registered types need schema migration.
func (s *schemaImpl) NeedsMigration(ctx context.Context, types ...any) (bool, []MigrationAction, error) {
	// Get existing schema from database
	existingIndexes, err := s.GetIndexes(ctx)
	if err != nil {
		return false, nil, fmt.Errorf("failed to get existing indexes: %w", err)
	}

	existingConstraints, err := s.GetConstraints(ctx)
	if err != nil {
		return false, nil, fmt.Errorf("failed to get existing constraints: %w", err)
	}

	// Build lookup maps by name
	indexByName := make(map[string]struct{}, len(existingIndexes))
	for _, idx := range existingIndexes {
		indexByName[idx.Name] = struct{}{}
	}

	constraintByName := make(map[string]struct{}, len(existingConstraints))
	for _, con := range existingConstraints {
		constraintByName[con.Name] = struct{}{}
	}

	// Collect schemas to check
	var schemas []*codec.SchemaMeta

	if len(types) == 0 {
		// Check all registered types
		for _, node := range s.registry.Nodes {
			if schema := node.Schema(); schema != nil {
				schemas = append(schemas, schema)
			}
		}
		for _, rel := range s.registry.Relationships {
			if schema := rel.Schema(); schema != nil {
				schemas = append(schemas, schema)
			}
		}
	} else {
		// Check only specified types
		for _, t := range types {
			schema := s.getSchemaForType(t)
			if schema != nil {
				schemas = append(schemas, schema)
			}
		}
	}

	// Find missing indexes and constraints
	var actions []MigrationAction

	for _, schema := range schemas {
		// Check indexes
		for _, idx := range schema.Indexes {
			if _, exists := indexByName[idx.Name]; !exists {
				cypher := idx.GenerateIndexCypher()
				if cypher != "" {
					actions = append(actions, MigrationAction{
						Type:   MigrationCreateIndex,
						Name:   idx.Name,
						Cypher: cypher,
					})
				}
			}
		}

		// Check constraints
		for _, con := range schema.Constraints {
			if _, exists := constraintByName[con.Name]; !exists {
				cypher := con.GenerateConstraintCypher()
				if cypher != "" {
					actions = append(actions, MigrationAction{
						Type:   MigrationCreateConstraint,
						Name:   con.Name,
						Cypher: cypher,
					})
				}
			}
		}
	}

	return len(actions) > 0, actions, nil
}

// getSchemaForType returns the schema for a given type instance.
func (s *schemaImpl) getSchemaForType(t any) *codec.SchemaMeta {
	typ := reflect.TypeOf(t)
	if typ.Kind() == reflect.Ptr {
		typ = typ.Elem()
	}

	entity := s.registry.Get(typ)
	if entity == nil {
		return nil
	}

	// Try as node
	if node, ok := entity.(*internal.RegisteredNode); ok {
		return node.Schema()
	}

	// Try as abstract node
	if absNode, ok := entity.(*internal.RegisteredAbstractNode); ok {
		return absNode.Schema()
	}

	// Try as relationship
	if rel, ok := entity.(*internal.RegisteredRelationship); ok {
		return rel.Schema()
	}

	return nil
}

// asString safely converts any value to string.
func asString(v any) string {
	if v == nil {
		return ""
	}
	if s, ok := v.(string); ok {
		return s
	}
	return fmt.Sprintf("%v", v)
}
