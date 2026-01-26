package orm

import (
	"fmt"
	"reflect"
	"strings"

	falkordb "github.com/FalkorDB/falkordb-go/v2"
)

// IndexType represents the type of index
type IndexType string

const (
	IndexTypeRange    IndexType = "RANGE"
	IndexTypeFullText IndexType = "FULLTEXT"
	IndexTypeVector   IndexType = "VECTOR"
)

// IndexInfo contains information about an index
type IndexInfo struct {
	Label    string
	Property string
	Type     IndexType
	IsUnique bool
}

// IndexManager manages graph indexes
type IndexManager struct {
	graph *falkordb.Graph
}

// NewIndexManager creates a new index manager
func NewIndexManager(graph *falkordb.Graph) *IndexManager {
	return &IndexManager{
		graph: graph,
	}
}

// CreateIndexes creates indexes for an entity type based on metadata
func (m *IndexManager) CreateIndexes(entityType reflect.Type, ifNotExists bool) error {
	if entityType.Kind() == reflect.Ptr {
		entityType = entityType.Elem()
	}

	metadata := GetMetadata(entityType)
	if metadata == nil {
		return fmt.Errorf("entity type %s is not registered", entityType.Name())
	}

	var errors []error

	// Create indexes for indexed properties
	for _, prop := range metadata.Properties {
		if prop.Indexed || prop.Unique {
			indexType := IndexTypeRange
			if prop.IndexType != "" {
				indexType = IndexType(strings.ToUpper(prop.IndexType))
			}

			err := m.createIndex(metadata.PrimaryLabel, prop.GraphName, indexType, prop.Unique, ifNotExists)
			if err != nil {
				errors = append(errors, err)
			}
		}
	}

	if len(errors) > 0 {
		return fmt.Errorf("failed to create some indexes: %v", errors)
	}

	return nil
}

// CreateIndex creates a single index
func (m *IndexManager) CreateIndex(label, property string, indexType IndexType, unique bool) error {
	return m.createIndex(label, property, indexType, unique, false)
}

// createIndex creates an index with optional existence check
func (m *IndexManager) createIndex(label, property string, indexType IndexType, unique bool, ifNotExists bool) error {
	var query string

	// FalkorDB index creation syntax
	switch indexType {
	case IndexTypeRange:
		query = fmt.Sprintf("CREATE INDEX FOR (n:%s) ON (n.%s)", label, property)
	case IndexTypeFullText:
		query = fmt.Sprintf("CALL db.idx.fulltext.createNodeIndex('%s', '%s')", label, property)
	case IndexTypeVector:
		// Vector indexes require additional parameters (dimensions, similarity)
		// For now, just create a basic structure
		query = fmt.Sprintf("CREATE VECTOR INDEX FOR (n:%s) ON (n.%s)", label, property)
	default:
		return fmt.Errorf("unsupported index type: %s", indexType)
	}

	_, err := m.graph.Query(query, nil, nil)
	if err != nil {
		// Ignore "already exists" errors if ifNotExists is true
		if ifNotExists && strings.Contains(err.Error(), "already exists") {
			return nil
		}
		return fmt.Errorf("failed to create index on %s.%s: %w", label, property, err)
	}

	return nil
}

// DropIndexes drops all indexes for an entity type
func (m *IndexManager) DropIndexes(entityType reflect.Type) error {
	if entityType.Kind() == reflect.Ptr {
		entityType = entityType.Elem()
	}

	metadata := GetMetadata(entityType)
	if metadata == nil {
		return fmt.Errorf("entity type %s is not registered", entityType.Name())
	}

	var errors []error

	// Drop indexes for indexed properties
	for _, prop := range metadata.Properties {
		if prop.Indexed || prop.Unique {
			err := m.dropIndex(metadata.PrimaryLabel, prop.GraphName)
			if err != nil {
				errors = append(errors, err)
			}
		}
	}

	if len(errors) > 0 {
		return fmt.Errorf("failed to drop some indexes: %v", errors)
	}

	return nil
}

// DropIndex drops a single index
func (m *IndexManager) DropIndex(label, property string) error {
	return m.dropIndex(label, property)
}

// dropIndex drops an index
func (m *IndexManager) dropIndex(label, property string) error {
	query := fmt.Sprintf("DROP INDEX ON :%s(%s)", label, property)

	_, err := m.graph.Query(query, nil, nil)
	if err != nil {
		// Ignore "doesn't exist" errors
		if strings.Contains(err.Error(), "doesn't exist") || strings.Contains(err.Error(), "not found") {
			return nil
		}
		return fmt.Errorf("failed to drop index on %s.%s: %w", label, property, err)
	}

	return nil
}

// ListIndexes lists all indexes for a label
func (m *IndexManager) ListIndexes(label string) ([]IndexInfo, error) {
	// FalkorDB command to list indexes
	query := "CALL db.indexes()"

	result, err := m.graph.Query(query, nil, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to list indexes: %w", err)
	}

	indexes := []IndexInfo{}

	for result.Next() {
		record := result.Record()

		// Parse index information from result
		// Format varies by FalkorDB version, this is a simplified parser
		indexLabel, _ := record.GetByIndex(0)
		if indexLabel != nil {
			if labelStr, ok := indexLabel.(string); ok {
				// Check if this index is for our label
				if label == "" || strings.Contains(labelStr, label) {
					info := IndexInfo{
						Label:    label,
						Type:     IndexTypeRange,
						IsUnique: false,
					}
					indexes = append(indexes, info)
				}
			}
		}
	}

	return indexes, nil
}

// CreateIndexesForAll creates indexes for all registered entities
func (m *IndexManager) CreateIndexesForAll(ifNotExists bool) error {
	// This would need access to all registered entities
	// For now, return not implemented
	return fmt.Errorf("CreateIndexesForAll not yet implemented - use CreateIndexes per entity type")
}

// EnsureIndexes ensures indexes exist for an entity, creating them if needed
func (m *IndexManager) EnsureIndexes(entityType reflect.Type) error {
	return m.CreateIndexes(entityType, true)
}
