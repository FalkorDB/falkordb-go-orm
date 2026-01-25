package orm

import (
	"fmt"
	"reflect"

	falkordb "github.com/FalkorDB/falkordb-go/v2"
)

// Repository provides CRUD operations for an entity type
type Repository[T any] struct {
	graph        *falkordb.Graph
	entityType   reflect.Type
	metadata     *EntityMetadata
	mapper       *Mapper
	queryBuilder *QueryBuilder
	queryParser  *QueryParser
	relManager   *RelationshipManager
}

// NewRepository creates a new repository for the given entity type
func NewRepository[T any](graph *falkordb.Graph) *Repository[T] {
	var zero T
	entityType := reflect.TypeOf(zero)
	if entityType.Kind() == reflect.Ptr {
		entityType = entityType.Elem()
	}

	// Get metadata from registry
	metadata := GetMetadata(entityType)
	if metadata == nil {
		// Try to register if not already registered
		Register(new(T))
		metadata = GetMetadata(entityType)
	}

	return &Repository[T]{
		graph:        graph,
		entityType:   entityType,
		metadata:     metadata,
		mapper:       NewMapper(),
		queryBuilder: &QueryBuilder{},
		queryParser:  NewQueryParser(),
		relManager:   NewRelationshipManager(graph),
	}
}

// Save saves or updates an entity
func (r *Repository[T]) Save(entity *T) (*T, error) {
	if entity == nil {
		return nil, fmt.Errorf("entity cannot be nil")
	}

	// Check if entity has an ID
	hasID := false
	if r.metadata.IDProperty != nil {
		entityValue := reflect.ValueOf(entity).Elem()
		idField := entityValue.FieldByName(r.metadata.IDProperty.FieldName)
		if idField.IsValid() {
			if idField.Kind() == reflect.Ptr {
				hasID = !idField.IsNil()
			} else {
				hasID = !idField.IsZero()
			}
		}
	}

	var cypher string
	var params map[string]interface{}
	var err error

	if hasID {
		// Update existing entity using MERGE
		cypher, params, err = r.mapper.MapToMerge(entity)
	} else {
		// Create new entity
		cypher, params, err = r.mapper.MapToCreate(entity)
	}

	if err != nil {
		return nil, fmt.Errorf("failed to map entity: %w", err)
	}

	// Execute query
	result, err := r.graph.Query(cypher, params, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to execute query: %w", err)
	}

	// Get entity ID
	var entityID int64
	if !hasID && result.Next() {
		record := result.Record()
		nodeID, _ := record.GetByIndex(1) // node_id from RETURN clause
		r.mapper.UpdateEntityID(entity, nodeID)
		entityID = nodeID.(int64)
	} else if hasID {
		// Extract existing ID
		entityValue := reflect.ValueOf(entity).Elem()
		idField := entityValue.FieldByName(r.metadata.IDProperty.FieldName)
		if idField.Kind() == reflect.Ptr {
			entityID = idField.Elem().Int()
		} else {
			entityID = idField.Int()
		}
	}

	// Save cascade relationships if configured
	if err := r.relManager.SaveRelationships(entity, entityID, r.metadata, hasID); err != nil {
		return nil, fmt.Errorf("failed to save relationships: %w", err)
	}

	return entity, nil
}

// SaveAll saves multiple entities
func (r *Repository[T]) SaveAll(entities []*T) ([]*T, error) {
	saved := make([]*T, 0, len(entities))
	for _, entity := range entities {
		savedEntity, err := r.Save(entity)
		if err != nil {
			return saved, err
		}
		saved = append(saved, savedEntity)
	}
	return saved, nil
}

// FindByID finds an entity by its ID with optional eager loading
func (r *Repository[T]) FindByID(id interface{}, fetch ...string) (*T, error) {
	var cypher string
	var params map[string]interface{}

	if len(fetch) > 0 {
		// Build eager loading query
		cypher, params = r.queryBuilder.BuildEagerLoading(r.metadata, id, fetch)
	} else {
		// Build simple query
		cypher, params = r.queryBuilder.BuildMatchByID(r.metadata, id)
	}

	result, err := r.graph.Query(cypher, params, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to execute query: %w", err)
	}

	if !result.Next() {
		return nil, nil // Not found
	}

	record := result.Record()
	entity, err := r.mapper.MapFromRecord(record, r.entityType, "n")
	if err != nil {
		return nil, fmt.Errorf("failed to map record: %w", err)
	}

	// Load eager relationships if requested
	if len(fetch) > 0 {
		if err := r.loadEagerRelationships(entity.(*T), record, fetch); err != nil {
			return nil, fmt.Errorf("failed to load relationships: %w", err)
		}
	}

	return entity.(*T), nil
}

// FindAll finds all entities with optional eager loading
func (r *Repository[T]) FindAll(fetch ...string) ([]*T, error) {
	cypher, params := r.queryBuilder.BuildMatchAll(r.metadata)

	result, err := r.graph.Query(cypher, params, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to execute query: %w", err)
	}

	entities := make([]*T, 0)
	for result.Next() {
		record := result.Record()
		entity, err := r.mapper.MapFromRecord(record, r.entityType, "n")
		if err != nil {
			return nil, fmt.Errorf("failed to map record: %w", err)
		}

		// Load eager relationships if requested (would need individual queries per entity)
		// For true eager loading with FindAll, consider using a custom query
		if len(fetch) > 0 {
			// Get entity ID for loading relationships
			entityValue := reflect.ValueOf(entity).Elem()
			if r.metadata.IDProperty != nil {
				idField := entityValue.FieldByName(r.metadata.IDProperty.FieldName)
				if idField.IsValid() {
					var entityID int64
					if idField.Kind() == reflect.Ptr {
						entityID = idField.Elem().Int()
					} else {
						entityID = idField.Int()
					}

					// Load each requested relationship
					for _, relName := range fetch {
						if err := r.loadRelationship(entity.(*T), entityID, relName); err != nil {
							return nil, fmt.Errorf("failed to load relationship %s: %w", relName, err)
						}
					}
				}
			}
		}

		entities = append(entities, entity.(*T))
	}

	return entities, nil
}

// FindAllByID finds entities by multiple IDs
func (r *Repository[T]) FindAllByID(ids []interface{}) ([]*T, error) {
	if len(ids) == 0 {
		return []*T{}, nil
	}

	cypher, params := r.queryBuilder.BuildMatchByIDs(r.metadata, ids)

	result, err := r.graph.Query(cypher, params, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to execute query: %w", err)
	}

	entities := make([]*T, 0)
	for result.Next() {
		record := result.Record()
		entity, err := r.mapper.MapFromRecord(record, r.entityType, "n")
		if err != nil {
			return nil, fmt.Errorf("failed to map record: %w", err)
		}
		entities = append(entities, entity.(*T))
	}

	return entities, nil
}

// ExistsByID checks if an entity exists by ID
func (r *Repository[T]) ExistsByID(id interface{}) (bool, error) {
	cypher, params := r.queryBuilder.BuildExistsByID(r.metadata, id)

	result, err := r.graph.Query(cypher, params, nil)
	if err != nil {
		return false, fmt.Errorf("failed to execute query: %w", err)
	}

	if result.Next() {
		record := result.Record()
		exists, _ := record.GetByIndex(0)
		if existsBool, ok := exists.(bool); ok {
			return existsBool, nil
		}
		// FalkorDB might return int64 instead of bool
		if existsInt, ok := exists.(int64); ok {
			return existsInt > 0, nil
		}
	}

	return false, nil
}

// ExecuteDerivedQuery executes a derived query from a method name pattern
// Example:
//   results, err := repo.ExecuteDerivedQuery("FindByAgeGreaterThan", 25)
//   results, err := repo.ExecuteDerivedQuery("FindByNameAndAge", "Alice", 30)
//   count, err := repo.ExecuteDerivedQuery("CountByAge", 25)
func (r *Repository[T]) ExecuteDerivedQuery(methodName string, args ...interface{}) (interface{}, error) {
	// Parse method name into QuerySpec
	spec, err := r.queryParser.Parse(methodName)
	if err != nil {
		return nil, fmt.Errorf("failed to parse method name: %v", err)
	}

	// Validate argument count
	expectedArgs := 0
	for _, condition := range spec.Conditions {
		expectedArgs += condition.Operator.ParamCount()
	}
	if len(args) != expectedArgs {
		return nil, fmt.Errorf("expected %d arguments but got %d", expectedArgs, len(args))
	}

	// Build query
	cypher, params := r.queryBuilder.BuildFromQuerySpec(spec, r.metadata, args)

	// Execute query based on operation type
	switch spec.Operation {
	case OpFind:
		// Return []*T
		resultSet, err := r.graph.Query(cypher, params, nil)
		if err != nil {
			return nil, fmt.Errorf("query execution failed: %v", err)
		}

		var results []*T
		for resultSet.Next() {
			record := resultSet.Record()
			entity, err := r.mapper.MapFromRecord(record, r.entityType, "n")
			if err != nil {
				return nil, fmt.Errorf("failed to map entity: %v", err)
			}
			results = append(results, entity.(*T))
		}
		return results, nil

	case OpCount:
		// Return int64
		resultSet, err := r.graph.Query(cypher, params, nil)
		if err != nil {
			return nil, fmt.Errorf("query execution failed: %v", err)
		}

		if resultSet.Next() {
			record := resultSet.Record()
			count := record.Values()[0].(int64)
			return count, nil
		}
		return int64(0), nil

	case OpExists:
		// Return bool
		resultSet, err := r.graph.Query(cypher, params, nil)
		if err != nil {
			return nil, fmt.Errorf("query execution failed: %v", err)
		}

		if resultSet.Next() {
			record := resultSet.Record()
			exists := record.Values()[0].(bool)
			return exists, nil
		}
		return false, nil

	case OpDelete:
		// Return error only
		_, err := r.graph.Query(cypher, params, nil)
		if err != nil {
			return nil, fmt.Errorf("delete execution failed: %v", err)
		}
		return nil, nil

	default:
		return nil, fmt.Errorf("unsupported operation: %v", spec.Operation)
	}
}

// loadEagerRelationships loads eagerly fetched relationships from a record
func (r *Repository[T]) loadEagerRelationships(entity *T, record *falkordb.Record, fetch []string) error {
	entityValue := reflect.ValueOf(entity).Elem()

	for i, relName := range fetch {
		relMeta := r.metadata.GetRelationshipByName(relName)
		if relMeta == nil {
			continue
		}

		// Get relationship field
		relField := entityValue.FieldByName(relName)
		if !relField.IsValid() || !relField.CanSet() {
			continue
		}

		// Get relationship data from record (index starts at 1, 0 is the main entity)
		relData, err := record.GetByIndex(i + 1)
		if err != nil || relData == nil {
			continue
		}

		// Map relationship data based on type
		if relMeta.IsList {
			// Handle list relationships
			if relList, ok := relData.([]interface{}); ok {
				items := make([]interface{}, 0, len(relList))
				for _, relItem := range relList {
					if relItem != nil {
						items = append(items, relItem)
					}
				}
				// Set the loaded items to the relationship field
				// This would need proper type conversion based on field type
			}
		} else {
			// Handle single relationship
			// This would need proper type conversion based on field type
		}
	}

	return nil
}

// loadRelationship loads a single relationship for an entity
func (r *Repository[T]) loadRelationship(entity *T, entityID int64, relName string) error {
	relMeta := r.metadata.GetRelationshipByName(relName)
	if relMeta == nil {
		return fmt.Errorf("relationship %s not found", relName)
	}

	// Get target type
	targetType := relMeta.TargetType
	if targetType.Kind() == reflect.Ptr {
		targetType = targetType.Elem()
	}

	// Load relationships using relationship manager
	relatedData, err := r.relManager.LoadRelationships(entityID, relMeta, targetType)
	if err != nil {
		return err
	}

	// Set the loaded data to the entity field
	entityValue := reflect.ValueOf(entity).Elem()
	relField := entityValue.FieldByName(relName)
	if !relField.IsValid() || !relField.CanSet() {
		return fmt.Errorf("cannot set relationship field %s", relName)
	}

	// Convert and set the relationship data
	if relatedData != nil {
		relField.Set(reflect.ValueOf(relatedData))
	}

	return nil
}

// Count returns the total number of entities
func (r *Repository[T]) Count() (int64, error) {
	cypher, params := r.queryBuilder.BuildCount(r.metadata)

	result, err := r.graph.Query(cypher, params, nil)
	if err != nil {
		return 0, fmt.Errorf("failed to execute query: %w", err)
	}

	if result.Next() {
		record := result.Record()
		count, _ := record.GetByIndex(0)
		if countInt, ok := count.(int64); ok {
			return countInt, nil
		}
		if countInt, ok := count.(int); ok {
			return int64(countInt), nil
		}
	}

	return 0, nil
}

// Delete deletes an entity
func (r *Repository[T]) Delete(entity *T) error {
	if entity == nil {
		return fmt.Errorf("entity cannot be nil")
	}

	if r.metadata.IDProperty == nil {
		return fmt.Errorf("entity has no ID property")
	}

	// Get entity ID
	entityValue := reflect.ValueOf(entity).Elem()
	idField := entityValue.FieldByName(r.metadata.IDProperty.FieldName)
	if !idField.IsValid() {
		return fmt.Errorf("ID field not found")
	}

	var id interface{}
	if idField.Kind() == reflect.Ptr {
		if idField.IsNil() {
			return fmt.Errorf("entity ID is nil")
		}
		id = idField.Elem().Interface()
	} else {
		id = idField.Interface()
	}

	return r.DeleteByID(id)
}

// DeleteByID deletes an entity by ID
func (r *Repository[T]) DeleteByID(id interface{}) error {
	cypher, params := r.queryBuilder.BuildDeleteByID(r.metadata, id)

	_, err := r.graph.Query(cypher, params, nil)
	if err != nil {
		return fmt.Errorf("failed to execute delete: %w", err)
	}

	return nil
}

// DeleteAll deletes multiple entities
func (r *Repository[T]) DeleteAll(entities []*T) error {
	for _, entity := range entities {
		if err := r.Delete(entity); err != nil {
			return err
		}
	}
	return nil
}

// DeleteAllEntities deletes all entities of this type
func (r *Repository[T]) DeleteAllEntities() error {
	cypher, params := r.queryBuilder.BuildDeleteAll(r.metadata)

	_, err := r.graph.Query(cypher, params, nil)
	if err != nil {
		return fmt.Errorf("failed to execute delete all: %w", err)
	}

	return nil
}

// FindAllPaginated returns a page of entities
func (r *Repository[T]) FindAllPaginated(pageable *Pageable) (*Page[T], error) {
	// First, get total count
	totalCount, err := r.Count()
	if err != nil {
		return nil, fmt.Errorf("failed to count entities: %w", err)
	}

	// Build paginated query
	cypher, params := r.queryBuilder.BuildMatchWithPagination(
		r.metadata,
		pageable.Page,
		pageable.Size,
		pageable.SortBy,
		pageable.Direction,
	)

	result, err := r.graph.Query(cypher, params, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to execute query: %w", err)
	}

	entities := make([]*T, 0, pageable.Size)
	for result.Next() {
		record := result.Record()
		entity, err := r.mapper.MapFromRecord(record, r.entityType, "n")
		if err != nil {
			return nil, fmt.Errorf("failed to map record: %w", err)
		}
		entities = append(entities, entity.(*T))
	}

	page := NewPage(entities, pageable, totalCount)
	return page, nil
}

// Query returns a query builder chain for advanced queries
func (r *Repository[T]) Query() *QueryBuilderChain {
	return NewQueryBuilderChain(r.metadata)
}

// ExecuteQuery executes a query built with QueryBuilderChain
func (r *Repository[T]) ExecuteQuery(chain *QueryBuilderChain) ([]*T, error) {
	cypher, params := chain.Build()

	result, err := r.graph.Query(cypher, params, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to execute query: %w", err)
	}

	entities := make([]*T, 0)
	for result.Next() {
		record := result.Record()
		entity, err := r.mapper.MapFromRecord(record, r.entityType, "n")
		if err != nil {
			return nil, fmt.Errorf("failed to map record: %w", err)
		}
		entities = append(entities, entity.(*T))
	}

	return entities, nil
}
