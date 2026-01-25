package orm

import (
	"fmt"
	"reflect"
	"strings"
	"sync"
)

// Direction represents relationship direction
type Direction string

const (
	DirectionOutgoing Direction = "OUTGOING"
	DirectionIncoming Direction = "INCOMING"
	DirectionBoth     Direction = "BOTH"
)

// EntityMetadata contains metadata about an entity type
type EntityMetadata struct {
	EntityType    reflect.Type
	Labels        []string
	PrimaryLabel  string
	Properties    []*PropertyMetadata
	Relationships []*RelationshipMetadata
	IDProperty    *PropertyMetadata
}

// PropertyMetadata contains metadata about a single property
type PropertyMetadata struct {
	FieldName   string
	GraphName   string
	Type        reflect.Type
	IsID        bool
	IDGenerator IDGenerator
	Converter   TypeConverter
	Required    bool
	Indexed     bool
	Unique      bool
	IndexType   string // RANGE, FULLTEXT, VECTOR
	Interned    bool
}

// RelationshipMetadata contains metadata about a relationship
type RelationshipMetadata struct {
	FieldName        string
	RelationshipType string
	Direction        Direction
	TargetType       reflect.Type
	Lazy             bool
	Cascade          bool
	IsList           bool // true for []*T, false for *T
}

// IDGenerator interface for custom ID generation
type IDGenerator interface {
	Generate() (interface{}, error)
}

// TypeConverter interface for custom type conversion
type TypeConverter interface {
	ToGraph(value interface{}) (interface{}, error)
	FromGraph(value interface{}, targetType reflect.Type) (interface{}, error)
}

// EntityLabeler interface for entities that define their own labels
type EntityLabeler interface {
	Labels() []string
}

// EntityRegistry maintains a cache of entity metadata
type EntityRegistry struct {
	mu       sync.RWMutex
	entities map[reflect.Type]*EntityMetadata
}

// NewEntityRegistry creates a new entity registry
func NewEntityRegistry() *EntityRegistry {
	return &EntityRegistry{
		entities: make(map[reflect.Type]*EntityMetadata),
	}
}

// Register registers an entity and extracts its metadata
func (r *EntityRegistry) Register(entity interface{}) error {
	entityType := reflect.TypeOf(entity)
	if entityType.Kind() == reflect.Ptr {
		entityType = entityType.Elem()
	}

	if entityType.Kind() != reflect.Struct {
		return fmt.Errorf("entity must be a struct, got %s", entityType.Kind())
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	// Check if already registered
	if _, exists := r.entities[entityType]; exists {
		return nil
	}

	metadata, err := extractMetadata(entityType, entity)
	if err != nil {
		return err
	}

	r.entities[entityType] = metadata
	return nil
}

// Get retrieves metadata for an entity type
func (r *EntityRegistry) Get(entityType reflect.Type) *EntityMetadata {
	if entityType.Kind() == reflect.Ptr {
		entityType = entityType.Elem()
	}

	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.entities[entityType]
}

// extractMetadata extracts metadata from a struct type
func extractMetadata(entityType reflect.Type, entity interface{}) (*EntityMetadata, error) {
	metadata := &EntityMetadata{
		EntityType:    entityType,
		Properties:    make([]*PropertyMetadata, 0),
		Relationships: make([]*RelationshipMetadata, 0),
	}

	// Extract labels
	if labeler, ok := entity.(EntityLabeler); ok {
		metadata.Labels = labeler.Labels()
		if len(metadata.Labels) > 0 {
			metadata.PrimaryLabel = metadata.Labels[0]
		}
	} else {
		// Default to struct name
		metadata.Labels = []string{entityType.Name()}
		metadata.PrimaryLabel = entityType.Name()
	}

	// Extract properties and relationships from struct fields
	for i := 0; i < entityType.NumField(); i++ {
		field := entityType.Field(i)
		tag := field.Tag.Get("falkor")

		if tag == "" || tag == "-" {
			continue
		}

		// Parse tag
		parts := strings.Split(tag, ",")
		name := parts[0]
		options := parts[1:]

		// Check if it's a relationship
		if isRelationship(options) {
			relMeta, err := parseRelationshipMetadata(field, name, options)
			if err != nil {
				return nil, fmt.Errorf("field %s: %w", field.Name, err)
			}
			metadata.Relationships = append(metadata.Relationships, relMeta)
		} else {
			// It's a property
			propMeta := parsePropertyMetadata(field, name, options)
			metadata.Properties = append(metadata.Properties, propMeta)

			if propMeta.IsID {
				metadata.IDProperty = propMeta
			}
		}
	}

	return metadata, nil
}

// parsePropertyMetadata parses property metadata from struct field
func parsePropertyMetadata(field reflect.StructField, name string, options []string) *PropertyMetadata {
	meta := &PropertyMetadata{
		FieldName: field.Name,
		GraphName: name,
		Type:      field.Type,
	}

	for _, opt := range options {
		switch opt {
		case "id":
			meta.IsID = true
		case "generated":
			meta.IsID = true
			// IDGenerator will be set later if needed
		case "required":
			meta.Required = true
		case "indexed":
			meta.Indexed = true
		case "unique":
			meta.Unique = true
		case "interned":
			meta.Interned = true
		default:
			// Check for index_type
			if strings.HasPrefix(opt, "index_type=") {
				meta.IndexType = strings.TrimPrefix(opt, "index_type=")
			}
		}
	}

	// Default index type
	if meta.Indexed && meta.IndexType == "" {
		meta.IndexType = "RANGE"
	}

	return meta
}

// parseRelationshipMetadata parses relationship metadata from struct field
func parseRelationshipMetadata(field reflect.StructField, name string, options []string) (*RelationshipMetadata, error) {
	meta := &RelationshipMetadata{
		FieldName: field.Name,
		Lazy:      true, // Default to lazy loading
	}

	// Determine if it's a list or single entity
	fieldType := field.Type
	if fieldType.Kind() == reflect.Ptr {
		// Check if it's a pointer to a slice/array
		elemType := fieldType.Elem()
		if elemType.Kind() == reflect.Slice || elemType.Kind() == reflect.Array {
			meta.IsList = true
			meta.TargetType = elemType.Elem()
		} else {
			meta.IsList = false
			meta.TargetType = elemType
		}
	} else if fieldType.Kind() == reflect.Slice || fieldType.Kind() == reflect.Array {
		meta.IsList = true
		meta.TargetType = fieldType.Elem()
	} else {
		meta.TargetType = fieldType
	}

	// Parse options
	for _, opt := range options {
		if strings.HasPrefix(opt, "rel=") {
			meta.RelationshipType = strings.TrimPrefix(opt, "rel=")
		} else if strings.HasPrefix(opt, "dir=") {
			dirStr := strings.TrimPrefix(opt, "dir=")
			switch strings.ToUpper(dirStr) {
			case "OUT", "OUTGOING":
				meta.Direction = DirectionOutgoing
			case "IN", "INCOMING":
				meta.Direction = DirectionIncoming
			case "BOTH":
				meta.Direction = DirectionBoth
			default:
				return nil, fmt.Errorf("invalid direction: %s", dirStr)
			}
		} else if opt == "lazy" {
			meta.Lazy = true
		} else if opt == "eager" {
			meta.Lazy = false
		} else if opt == "cascade" {
			meta.Cascade = true
		}
	}

	// Validate relationship type is set
	if meta.RelationshipType == "" {
		return nil, fmt.Errorf("relationship type (rel=) is required")
	}

	// Default direction
	if meta.Direction == "" {
		meta.Direction = DirectionOutgoing
	}

	return meta, nil
}

// isRelationship checks if the field is a relationship based on options
func isRelationship(options []string) bool {
	for _, opt := range options {
		if strings.HasPrefix(opt, "rel=") {
			return true
		}
	}
	return false
}

// GetPropertyByName finds a property by field name
func (m *EntityMetadata) GetPropertyByName(name string) *PropertyMetadata {
	for _, prop := range m.Properties {
		if prop.FieldName == name || prop.GraphName == name {
			return prop
		}
	}
	return nil
}

// GetRelationshipByName finds a relationship by field name
func (m *EntityMetadata) GetRelationshipByName(name string) *RelationshipMetadata {
	for _, rel := range m.Relationships {
		if rel.FieldName == name {
			return rel
		}
	}
	return nil
}

// IsRelationshipField checks if a field name is a relationship
func (m *EntityMetadata) IsRelationshipField(fieldName string) bool {
	return m.GetRelationshipByName(fieldName) != nil
}

// Global registry instance
var globalRegistry = NewEntityRegistry()

// Register registers an entity type globally
func Register(entity interface{}) error {
	return globalRegistry.Register(entity)
}

// GetMetadata retrieves metadata for an entity type
func GetMetadata(entityType reflect.Type) *EntityMetadata {
	return globalRegistry.Get(entityType)
}

// GetEntityRegistry returns the global entity registry
func GetEntityRegistry() *EntityRegistry {
	return globalRegistry
}
