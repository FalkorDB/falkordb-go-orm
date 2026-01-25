package orm

import (
	"fmt"
	"reflect"
	"sync"

	"github.com/FalkorDB/falkordb-go/v2"
)

// LazyList provides lazy loading for a list of related entities
type LazyList[T any] struct {
	graph        *falkordb.Graph
	sourceID     int64
	relMeta      *RelationshipMetadata
	mapper       *Mapper
	queryBuilder *QueryBuilder
	loaded       bool
	items        []*T
	mu           sync.Mutex
}

// NewLazyList creates a new lazy list
func NewLazyList[T any](graph *falkordb.Graph, sourceID int64, relMeta *RelationshipMetadata) *LazyList[T] {
	return &LazyList[T]{
		graph:        graph,
		sourceID:     sourceID,
		relMeta:      relMeta,
		mapper:       NewMapper(),
		queryBuilder: &QueryBuilder{},
		loaded:       false,
		items:        make([]*T, 0),
	}
}

// Load loads the related entities from the database
func (l *LazyList[T]) Load() error {
	l.mu.Lock()
	defer l.mu.Unlock()

	if l.loaded {
		return nil
	}

	// Build query to load related entities
	cypher, params := l.queryBuilder.BuildRelationshipQuery(l.relMeta, l.sourceID)

	result, err := l.graph.Query(cypher, params, nil)
	if err != nil {
		return fmt.Errorf("failed to load relationships: %w", err)
	}

	// Map results to entities
	var zero T
	targetType := reflect.TypeOf(zero)
	if targetType.Kind() == reflect.Ptr {
		targetType = targetType.Elem()
	}

	for result.Next() {
		record := result.Record()
		entity, err := l.mapper.MapFromRecord(record, targetType, "target")
		if err != nil {
			return fmt.Errorf("failed to map related entity: %w", err)
		}
		l.items = append(l.items, entity.(*T))
	}

	l.loaded = true
	return nil
}

// Get returns the list of entities, loading them if necessary
func (l *LazyList[T]) Get() ([]*T, error) {
	if !l.loaded {
		if err := l.Load(); err != nil {
			return nil, err
		}
	}
	return l.items, nil
}

// Len returns the number of items (loads if necessary)
func (l *LazyList[T]) Len() (int, error) {
	if !l.loaded {
		if err := l.Load(); err != nil {
			return 0, err
		}
	}
	return len(l.items), nil
}

// IsLoaded returns whether the list has been loaded
func (l *LazyList[T]) IsLoaded() bool {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.loaded
}

// ForEach iterates over the items, loading them if necessary
func (l *LazyList[T]) ForEach(fn func(*T) error) error {
	items, err := l.Get()
	if err != nil {
		return err
	}

	for _, item := range items {
		if err := fn(item); err != nil {
			return err
		}
	}

	return nil
}

// Add adds an item to the list (marks as dirty for cascade save)
func (l *LazyList[T]) Add(item *T) {
	l.mu.Lock()
	defer l.mu.Unlock()

	l.items = append(l.items, item)
	l.loaded = true // Mark as loaded since we have modified it
}

// Clear clears the list
func (l *LazyList[T]) Clear() {
	l.mu.Lock()
	defer l.mu.Unlock()

	l.items = make([]*T, 0)
	l.loaded = true
}

// LazySingle provides lazy loading for a single related entity
type LazySingle[T any] struct {
	graph        *falkordb.Graph
	sourceID     int64
	relMeta      *RelationshipMetadata
	mapper       *Mapper
	queryBuilder *QueryBuilder
	loaded       bool
	item         *T
	mu           sync.Mutex
}

// NewLazySingle creates a new lazy single
func NewLazySingle[T any](graph *falkordb.Graph, sourceID int64, relMeta *RelationshipMetadata) *LazySingle[T] {
	return &LazySingle[T]{
		graph:        graph,
		sourceID:     sourceID,
		relMeta:      relMeta,
		mapper:       NewMapper(),
		queryBuilder: &QueryBuilder{},
		loaded:       false,
	}
}

// Load loads the related entity from the database
func (l *LazySingle[T]) Load() error {
	l.mu.Lock()
	defer l.mu.Unlock()

	if l.loaded {
		return nil
	}

	// Build query to load related entity
	cypher, params := l.queryBuilder.BuildRelationshipQuery(l.relMeta, l.sourceID)

	result, err := l.graph.Query(cypher, params, nil)
	if err != nil {
		return fmt.Errorf("failed to load relationship: %w", err)
	}

	// Map result to entity
	var zero T
	targetType := reflect.TypeOf(zero)
	if targetType.Kind() == reflect.Ptr {
		targetType = targetType.Elem()
	}

	if result.Next() {
		record := result.Record()
		entity, err := l.mapper.MapFromRecord(record, targetType, "target")
		if err != nil {
			return fmt.Errorf("failed to map related entity: %w", err)
		}
		l.item = entity.(*T)
	}

	l.loaded = true
	return nil
}

// Get returns the entity, loading it if necessary
func (l *LazySingle[T]) Get() (*T, error) {
	if !l.loaded {
		if err := l.Load(); err != nil {
			return nil, err
		}
	}
	return l.item, nil
}

// IsLoaded returns whether the entity has been loaded
func (l *LazySingle[T]) IsLoaded() bool {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.loaded
}

// Set sets the related entity (marks as dirty for cascade save)
func (l *LazySingle[T]) Set(item *T) {
	l.mu.Lock()
	defer l.mu.Unlock()

	l.item = item
	l.loaded = true
}

// RelationshipManager manages relationship operations
type RelationshipManager struct {
	graph        *falkordb.Graph
	mapper       *Mapper
	queryBuilder *QueryBuilder
}

// NewRelationshipManager creates a new relationship manager
func NewRelationshipManager(graph *falkordb.Graph) *RelationshipManager {
	return &RelationshipManager{
		graph:        graph,
		mapper:       NewMapper(),
		queryBuilder: &QueryBuilder{},
	}
}

// SaveRelationships saves relationships for an entity
func (m *RelationshipManager) SaveRelationships(source interface{}, sourceID int64, metadata *EntityMetadata, isUpdate bool) error {
	sourceValue := reflect.ValueOf(source)
	if sourceValue.Kind() == reflect.Ptr {
		sourceValue = sourceValue.Elem()
	}

	for _, relMeta := range metadata.Relationships {
		if !relMeta.Cascade {
			continue // Skip non-cascade relationships
		}

		fieldValue := sourceValue.FieldByName(relMeta.FieldName)
		if !fieldValue.IsValid() || fieldValue.IsNil() {
			continue
		}

		// Delete old relationships if updating
		if isUpdate {
			if err := m.DeleteRelationships(sourceID, relMeta); err != nil {
				return fmt.Errorf("failed to delete old relationships: %w", err)
			}
		}

		// Save new relationships
		if relMeta.IsList {
			// Handle list relationships
			if err := m.saveListRelationship(sourceID, fieldValue, relMeta); err != nil {
				return err
			}
		} else {
			// Handle single relationship
			if err := m.saveSingleRelationship(sourceID, fieldValue, relMeta); err != nil {
				return err
			}
		}
	}

	return nil
}

// saveListRelationship saves a list of related entities
func (m *RelationshipManager) saveListRelationship(sourceID int64, fieldValue reflect.Value, relMeta *RelationshipMetadata) error {
	// Handle LazyList[T]
	if fieldValue.Type().Name() == "LazyList" {
		// Get items from lazy list
		getMethod := fieldValue.MethodByName("Get")
		if !getMethod.IsValid() {
			return fmt.Errorf("LazyList does not have Get method")
		}

		results := getMethod.Call(nil)
		if len(results) != 2 {
			return fmt.Errorf("unexpected Get return values")
		}

		if !results[1].IsNil() {
			return results[1].Interface().(error)
		}

		fieldValue = results[0]
	}

	// Process slice/array
	if fieldValue.Kind() == reflect.Ptr {
		fieldValue = fieldValue.Elem()
	}

	for i := 0; i < fieldValue.Len(); i++ {
		targetEntity := fieldValue.Index(i).Interface()
		if err := m.createRelationship(sourceID, targetEntity, relMeta); err != nil {
			return err
		}
	}

	return nil
}

// saveSingleRelationship saves a single related entity
func (m *RelationshipManager) saveSingleRelationship(sourceID int64, fieldValue reflect.Value, relMeta *RelationshipMetadata) error {
	// Handle LazySingle[T]
	if fieldValue.Type().Name() == "LazySingle" {
		// Get item from lazy single
		getMethod := fieldValue.MethodByName("Get")
		if !getMethod.IsValid() {
			return fmt.Errorf("LazySingle does not have Get method")
		}

		results := getMethod.Call(nil)
		if len(results) != 2 {
			return fmt.Errorf("unexpected Get return values")
		}

		if !results[1].IsNil() {
			return results[1].Interface().(error)
		}

		fieldValue = results[0]
	}

	if fieldValue.IsNil() {
		return nil
	}

	targetEntity := fieldValue.Interface()
	return m.createRelationship(sourceID, targetEntity, relMeta)
}

// createRelationship creates a relationship between two entities
func (m *RelationshipManager) createRelationship(sourceID int64, targetEntity interface{}, relMeta *RelationshipMetadata) error {
	// Get target entity ID
	targetValue := reflect.ValueOf(targetEntity)
	if targetValue.Kind() == reflect.Ptr {
		targetValue = targetValue.Elem()
	}

	targetType := targetValue.Type()
	targetMetadata := GetMetadata(targetType)
	if targetMetadata == nil || targetMetadata.IDProperty == nil {
		return fmt.Errorf("target entity has no ID property")
	}

	idField := targetValue.FieldByName(targetMetadata.IDProperty.FieldName)
	if !idField.IsValid() {
		return fmt.Errorf("target entity ID field not found")
	}

	// Extract target ID
	var targetID int64
	if idField.Kind() == reflect.Ptr {
		if idField.IsNil() {
			return fmt.Errorf("target entity ID is nil")
		}
		targetID = idField.Elem().Int()
	} else {
		targetID = idField.Int()
	}

	// Build and execute relationship creation query
	cypher, params := m.queryBuilder.BuildCreateRelationship(sourceID, targetID, relMeta)
	_, err := m.graph.Query(cypher, params, nil)
	if err != nil {
		return fmt.Errorf("failed to create relationship: %w", err)
	}

	return nil
}

// DeleteRelationships deletes all relationships of a specific type for an entity
func (m *RelationshipManager) DeleteRelationships(sourceID int64, relMeta *RelationshipMetadata) error {
	cypher, params := m.queryBuilder.BuildDeleteRelationships(sourceID, relMeta)
	_, err := m.graph.Query(cypher, params, nil)
	if err != nil {
		return fmt.Errorf("failed to delete relationships: %w", err)
	}
	return nil
}

// LoadRelationships loads relationships for an entity
func (m *RelationshipManager) LoadRelationships(sourceID int64, relMeta *RelationshipMetadata, targetType reflect.Type) (interface{}, error) {
	cypher, params := m.queryBuilder.BuildRelationshipQuery(relMeta, sourceID)

	result, err := m.graph.Query(cypher, params, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to load relationships: %w", err)
	}

	if relMeta.IsList {
		// Load list of entities
		items := reflect.MakeSlice(reflect.SliceOf(reflect.PtrTo(targetType)), 0, 0)

		for result.Next() {
			record := result.Record()
			entity, err := m.mapper.MapFromRecord(record, targetType, "target")
			if err != nil {
				return nil, fmt.Errorf("failed to map related entity: %w", err)
			}
			items = reflect.Append(items, reflect.ValueOf(entity))
		}

		return items.Interface(), nil
	} else {
		// Load single entity
		if result.Next() {
			record := result.Record()
			entity, err := m.mapper.MapFromRecord(record, targetType, "target")
			if err != nil {
				return nil, fmt.Errorf("failed to map related entity: %w", err)
			}
			return entity, nil
		}
		return nil, nil
	}
}
