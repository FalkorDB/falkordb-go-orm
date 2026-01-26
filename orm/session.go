package orm

import (
	"fmt"
	"reflect"
	"sync"

	falkordb "github.com/FalkorDB/falkordb-go/v2"
)

// Session represents a unit of work with identity map and change tracking
type Session struct {
	graph        *falkordb.Graph
	mapper       *Mapper
	queryBuilder *QueryBuilder

	// Identity map: "Type:ID" -> entity instance
	identityMap map[string]interface{}

	// Change tracking
	new     map[interface{}]bool
	dirty   map[interface{}]bool
	deleted map[interface{}]bool

	// Original state for dirty checking: entity -> property map
	originalState map[interface{}]map[string]interface{}

	mu            sync.RWMutex
	inTransaction bool
	closed        bool
}

// NewSession creates a new session for the given graph
func NewSession(graph *falkordb.Graph) *Session {
	return &Session{
		graph:         graph,
		mapper:        NewMapper(),
		queryBuilder:  NewQueryBuilder(),
		identityMap:   make(map[string]interface{}),
		new:           make(map[interface{}]bool),
		dirty:         make(map[interface{}]bool),
		deleted:       make(map[interface{}]bool),
		originalState: make(map[interface{}]map[string]interface{}),
	}
}

// Add marks an entity as new to be persisted on flush
func (s *Session) Add(entity interface{}) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.closed {
		return fmt.Errorf("session is closed")
	}

	// Check if already in session
	if s.new[entity] || s.dirty[entity] || s.deleted[entity] {
		return fmt.Errorf("entity already in session")
	}

	s.new[entity] = true
	return nil
}

// Get retrieves an entity from the identity map or loads it from the database
func (s *Session) Get(entityType reflect.Type, id interface{}) (interface{}, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.closed {
		return nil, fmt.Errorf("session is closed")
	}

	// Build identity map key
	key := fmt.Sprintf("%s:%v", entityType.Name(), id)

	// Check identity map first
	if entity, exists := s.identityMap[key]; exists {
		return entity, nil
	}

	// Load from database
	metadata := globalRegistry.Get(entityType)
	if metadata == nil {
		return nil, fmt.Errorf("entity not registered")
	}

	// Build query
	query, params := s.queryBuilder.BuildMatchByID(metadata, id)

	// Execute query
	result, err := s.graph.Query(query, params, nil)
	if err != nil {
		return nil, fmt.Errorf("query failed: %w", err)
	}

	if !result.Next() {
		return nil, fmt.Errorf("entity not found")
	}

	// Map from record
	record := result.Record()
	entity, err := s.mapper.MapFromRecord(record, entityType, "n")
	if err != nil {
		return nil, fmt.Errorf("mapping failed: %w", err)
	}

	// Store in identity map
	s.identityMap[key] = entity

	// Store original state for dirty checking
	props, err := s.mapper.ExtractProperties(entity)
	if err != nil {
		return nil, fmt.Errorf("failed to extract properties: %w", err)
	}
	s.originalState[entity] = s.copyProperties(props)

	return entity, nil
}

// MarkDirty marks an entity as modified
func (s *Session) MarkDirty(entity interface{}) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.closed {
		return fmt.Errorf("session is closed")
	}

	// Don't mark new or deleted entities as dirty
	if s.new[entity] || s.deleted[entity] {
		return nil
	}

	s.dirty[entity] = true
	return nil
}

// Delete marks an entity for deletion
func (s *Session) Delete(entity interface{}) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.closed {
		return fmt.Errorf("session is closed")
	}

	// If entity is new, just remove it from new set
	if s.new[entity] {
		delete(s.new, entity)
		return nil
	}

	// Remove from dirty set if present
	delete(s.dirty, entity)

	// Mark for deletion
	s.deleted[entity] = true

	return nil
}

// Flush persists all pending changes to the database
func (s *Session) Flush() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.closed {
		return fmt.Errorf("session is closed")
	}

	var errs []error

	// 1. Process inserts (new entities)
	for entity := range s.new {
		if err := s.insertEntity(entity); err != nil {
			errs = append(errs, fmt.Errorf("insert failed: %w", err))
		}
	}

	// 2. Process updates (dirty entities)
	for entity := range s.dirty {
		if err := s.updateEntity(entity); err != nil {
			errs = append(errs, fmt.Errorf("update failed: %w", err))
		}
	}

	// 3. Process deletes
	for entity := range s.deleted {
		if err := s.deleteEntity(entity); err != nil {
			errs = append(errs, fmt.Errorf("delete failed: %w", err))
		}
	}

	if len(errs) > 0 {
		return fmt.Errorf("flush failed with %d errors: %v", len(errs), errs)
	}

	// Clear tracking sets
	s.new = make(map[interface{}]bool)
	s.dirty = make(map[interface{}]bool)
	s.deleted = make(map[interface{}]bool)

	return nil
}

// Commit commits the transaction and flushes changes
func (s *Session) Commit() error {
	s.mu.Lock()
	if s.closed {
		s.mu.Unlock()
		return fmt.Errorf("session is closed")
	}

	if !s.inTransaction {
		s.mu.Unlock()
		return fmt.Errorf("no active transaction")
	}
	s.mu.Unlock()

	// Flush changes first
	if err := s.Flush(); err != nil {
		return fmt.Errorf("flush before commit failed: %w", err)
	}

	s.mu.Lock()
	s.inTransaction = false
	s.mu.Unlock()

	return nil
}

// Rollback rolls back the transaction and discards changes
func (s *Session) Rollback() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.closed {
		return fmt.Errorf("session is closed")
	}

	if !s.inTransaction {
		return fmt.Errorf("no active transaction")
	}

	// Clear all pending changes
	s.new = make(map[interface{}]bool)
	s.dirty = make(map[interface{}]bool)
	s.deleted = make(map[interface{}]bool)
	s.originalState = make(map[interface{}]map[string]interface{})

	s.inTransaction = false

	return nil
}

// BeginTransaction starts a new transaction
func (s *Session) BeginTransaction() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.closed {
		return fmt.Errorf("session is closed")
	}

	if s.inTransaction {
		return fmt.Errorf("transaction already active")
	}

	s.inTransaction = true
	return nil
}

// Close closes the session
func (s *Session) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.closed {
		return nil
	}

	// Rollback if transaction is active
	if s.inTransaction {
		// Clear without error since we're closing
		s.new = make(map[interface{}]bool)
		s.dirty = make(map[interface{}]bool)
		s.deleted = make(map[interface{}]bool)
		s.inTransaction = false
	}

	s.closed = true
	return nil
}

// IsInTransaction returns whether the session has an active transaction
func (s *Session) IsInTransaction() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.inTransaction
}

// IsClosed returns whether the session is closed
func (s *Session) IsClosed() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.closed
}

// WithSession executes a function within a session context
// Automatically commits on success, rolls back on error
func WithSession(graph *falkordb.Graph, fn func(*Session) error) error {
	session := NewSession(graph)
	defer session.Close()

	// Begin transaction
	if err := session.BeginTransaction(); err != nil {
		return err
	}

	// Execute function
	err := fn(session)

	if err != nil {
		// Rollback on error
		if rbErr := session.Rollback(); rbErr != nil {
			return fmt.Errorf("rollback failed: %v (original error: %w)", rbErr, err)
		}
		return err
	}

	// Commit on success
	if err := session.Commit(); err != nil {
		return fmt.Errorf("commit failed: %w", err)
	}

	return nil
}

// Helper methods

func (s *Session) insertEntity(entity interface{}) error {
	entityType := reflect.TypeOf(entity)
	if entityType.Kind() == reflect.Ptr {
		entityType = entityType.Elem()
	}

	metadata := globalRegistry.Get(entityType)
	if metadata == nil {
		return fmt.Errorf("entity not registered")
	}

	// Build CREATE query using Mapper
	query, params, err := s.mapper.MapToCreate(entity)
	if err != nil {
		return err
	}

	// Execute query
	result, err := s.graph.Query(query, params, nil)
	if err != nil {
		return err
	}

	if result.Next() {
		record := result.Record()

		// Update entity with generated ID if present
		if metadata.IDProperty != nil {
			// Extract ID from record using GetByIndex or Get
			if idVal, ok := record.Get("node_id"); ok {
				if id, ok := idVal.(int64); ok {
					// Set ID on entity
					entityVal := reflect.ValueOf(entity)
					if entityVal.Kind() == reflect.Ptr {
						entityVal = entityVal.Elem()
					}

					idField := entityVal.FieldByName(metadata.IDProperty.FieldName)
					if idField.IsValid() && idField.CanSet() {
						if idField.Kind() == reflect.Ptr {
							idPtr := reflect.New(idField.Type().Elem())
							idPtr.Elem().SetInt(id)
							idField.Set(idPtr)
						} else {
							idField.SetInt(id)
						}
					}

					// Add to identity map
					key := fmt.Sprintf("%s:%v", entityType.Name(), id)
					s.identityMap[key] = entity
				}
			}
		}
	}

	return nil
}

func (s *Session) updateEntity(entity interface{}) error {
	entityType := reflect.TypeOf(entity)
	if entityType.Kind() == reflect.Ptr {
		entityType = entityType.Elem()
	}

	metadata := globalRegistry.Get(entityType)
	if metadata == nil {
		return fmt.Errorf("entity not registered")
	}

	// Get entity ID
	id, err := s.getEntityID(entity, metadata)
	if err != nil {
		return err
	}

	// Check if actually dirty by extracting current properties
	props, err := s.mapper.ExtractProperties(entity)
	if err != nil {
		return err
	}

	original, hasOriginal := s.originalState[entity]
	if hasOriginal && !s.propertiesChanged(original, props) {
		// Not actually dirty, skip
		return nil
	}

	// Build MERGE query (acts as update) using Mapper
	query, params, err := s.mapper.MapToMerge(entity)
	if err != nil {
		return err
	}

	// Execute query
	_, err = s.graph.Query(query, params, nil)
	if err != nil {
		return err
	}

	// Update original state
	s.originalState[entity] = s.copyProperties(props)

	// Update identity map
	key := fmt.Sprintf("%s:%v", entityType.Name(), id)
	s.identityMap[key] = entity

	return nil
}

func (s *Session) deleteEntity(entity interface{}) error {
	entityType := reflect.TypeOf(entity)
	if entityType.Kind() == reflect.Ptr {
		entityType = entityType.Elem()
	}

	metadata := globalRegistry.Get(entityType)
	if metadata == nil {
		return fmt.Errorf("entity not registered")
	}

	// Get entity ID
	id, err := s.getEntityID(entity, metadata)
	if err != nil {
		return err
	}

	// Build DELETE query
	labels := metadata.PrimaryLabel
	query := fmt.Sprintf("MATCH (n:%s) WHERE id(n) = $id DELETE n", labels)
	params := map[string]interface{}{"id": id}

	// Execute query
	_, err = s.graph.Query(query, params, nil)
	if err != nil {
		return err
	}

	// Remove from identity map
	key := fmt.Sprintf("%s:%v", entityType.Name(), id)
	delete(s.identityMap, key)
	delete(s.originalState, entity)

	return nil
}

func (s *Session) getEntityID(entity interface{}, metadata *EntityMetadata) (interface{}, error) {
	if metadata.IDProperty == nil {
		return nil, fmt.Errorf("entity has no ID property")
	}

	entityVal := reflect.ValueOf(entity)
	if entityVal.Kind() == reflect.Ptr {
		entityVal = entityVal.Elem()
	}

	idField := entityVal.FieldByName(metadata.IDProperty.FieldName)
	if !idField.IsValid() {
		return nil, fmt.Errorf("ID field not found")
	}

	// Handle pointer to ID
	if idField.Kind() == reflect.Ptr {
		if idField.IsNil() {
			return nil, fmt.Errorf("entity ID is nil")
		}
		return idField.Elem().Interface(), nil
	}

	return idField.Interface(), nil
}

func (s *Session) copyProperties(props map[string]interface{}) map[string]interface{} {
	copy := make(map[string]interface{}, len(props))
	for k, v := range props {
		copy[k] = v
	}
	return copy
}

func (s *Session) propertiesChanged(original, current map[string]interface{}) bool {
	if len(original) != len(current) {
		return true
	}

	for k, v1 := range original {
		v2, exists := current[k]
		if !exists {
			return true
		}

		// Simple equality check
		if !reflect.DeepEqual(v1, v2) {
			return true
		}
	}

	return false
}
