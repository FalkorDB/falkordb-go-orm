package orm

import (
	"fmt"
	"reflect"
	"strings"

	falkordb "github.com/FalkorDB/falkordb-go/v2"
)

// ValidationError represents a schema validation error
type ValidationError struct {
	EntityType string
	FieldName  string
	Message    string
}

func (e ValidationError) Error() string {
	return fmt.Sprintf("%s.%s: %s", e.EntityType, e.FieldName, e.Message)
}

// ValidationResult contains schema validation results
type ValidationResult struct {
	IsValid bool
	Errors  []ValidationError
}

// SchemaManager manages schema validation and synchronization
type SchemaManager struct {
	graph    *falkordb.Graph
	registry *EntityRegistry
}

// NewSchemaManager creates a new schema manager
func NewSchemaManager(graph *falkordb.Graph) *SchemaManager {
	return &SchemaManager{
		graph:    graph,
		registry: GetEntityRegistry(),
	}
}

// ValidateSchema validates entity schemas
func (m *SchemaManager) ValidateSchema(entityTypes []reflect.Type) *ValidationResult {
	result := &ValidationResult{
		IsValid: true,
		Errors:  []ValidationError{},
	}

	for _, entityType := range entityTypes {
		if entityType.Kind() == reflect.Ptr {
			entityType = entityType.Elem()
		}

		metadata := m.registry.Get(entityType)
		if metadata == nil {
			result.IsValid = false
			result.Errors = append(result.Errors, ValidationError{
				EntityType: entityType.Name(),
				Message:    "entity not registered",
			})
			continue
		}

		// Validate entity has labels
		if len(metadata.Labels) == 0 {
			result.IsValid = false
			result.Errors = append(result.Errors, ValidationError{
				EntityType: entityType.Name(),
				Message:    "entity must have at least one label",
			})
		}

		// Validate required properties
		for _, prop := range metadata.Properties {
			if prop.Required {
				// Check if field exists
				if prop.FieldName == "" {
					result.IsValid = false
					result.Errors = append(result.Errors, ValidationError{
						EntityType: entityType.Name(),
						FieldName:  prop.GraphName,
						Message:    "required property has no field name",
					})
				}
			}

			// Validate unique properties are indexed
			if prop.Unique && !prop.Indexed {
				// Automatically mark as indexed
				prop.Indexed = true
			}

			// Validate index type
			if prop.Indexed && prop.IndexType != "" {
				validTypes := []string{"RANGE", "FULLTEXT", "VECTOR", ""}
				valid := false
				for _, vt := range validTypes {
					if strings.ToUpper(prop.IndexType) == vt {
						valid = true
						break
					}
				}
				if !valid {
					result.IsValid = false
					result.Errors = append(result.Errors, ValidationError{
						EntityType: entityType.Name(),
						FieldName:  prop.FieldName,
						Message:    fmt.Sprintf("invalid index type: %s", prop.IndexType),
					})
				}
			}
		}

		// Validate relationships
		for _, rel := range metadata.Relationships {
			// Check relationship type is set
			if rel.RelationshipType == "" {
				result.IsValid = false
				result.Errors = append(result.Errors, ValidationError{
					EntityType: entityType.Name(),
					FieldName:  rel.FieldName,
					Message:    "relationship must have a type",
				})
			}

			// Check direction is valid
			if rel.Direction != DirectionOutgoing &&
				rel.Direction != DirectionIncoming &&
				rel.Direction != DirectionBoth {
				result.IsValid = false
				result.Errors = append(result.Errors, ValidationError{
					EntityType: entityType.Name(),
					FieldName:  rel.FieldName,
					Message:    fmt.Sprintf("invalid direction: %s", rel.Direction),
				})
			}
		}
	}

	return result
}

// SyncSchema synchronizes schema by creating missing indexes
func (m *SchemaManager) SyncSchema(entityTypes []reflect.Type) error {
	// First validate
	validation := m.ValidateSchema(entityTypes)
	if !validation.IsValid {
		return fmt.Errorf("schema validation failed: %v", validation.Errors)
	}

	// Create indexes
	indexManager := NewIndexManager(m.graph)

	var errors []error
	for _, entityType := range entityTypes {
		if err := indexManager.EnsureIndexes(entityType); err != nil {
			errors = append(errors, err)
		}
	}

	if len(errors) > 0 {
		return fmt.Errorf("schema sync encountered errors: %v", errors)
	}

	return nil
}

// ValidateEntity validates a single entity type
func (m *SchemaManager) ValidateEntity(entityType reflect.Type) *ValidationResult {
	return m.ValidateSchema([]reflect.Type{entityType})
}

// GetSchemaInfo returns schema information for all registered entities
func (m *SchemaManager) GetSchemaInfo() map[string]*EntityMetadata {
	// This would need to iterate all registered entities
	// For simplicity, return empty for now
	return make(map[string]*EntityMetadata)
}
