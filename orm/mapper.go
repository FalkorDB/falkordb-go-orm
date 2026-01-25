package orm

import (
	"fmt"
	"reflect"
	"strings"

	falkordb "github.com/FalkorDB/falkordb-go/v2"
)

// Mapper handles bidirectional conversion between Go structs and FalkorDB graph structures
type Mapper struct {
	registry     *EntityRegistry
	typeRegistry *TypeRegistry
}

// NewMapper creates a new mapper instance
func NewMapper() *Mapper {
	return &Mapper{
		registry:     globalRegistry,
		typeRegistry: globalTypeRegistry,
	}
}

// ExtractProperties extracts property values from an entity
func (m *Mapper) ExtractProperties(entity interface{}) (map[string]interface{}, error) {
	entityValue := reflect.ValueOf(entity)
	if entityValue.Kind() == reflect.Ptr {
		entityValue = entityValue.Elem()
	}

	entityType := entityValue.Type()
	metadata := m.registry.Get(entityType)
	if metadata == nil {
		return nil, fmt.Errorf("entity type %s is not registered", entityType.Name())
	}

	properties := make(map[string]interface{})

	for _, prop := range metadata.Properties {
		// Skip relationship fields
		if metadata.IsRelationshipField(prop.FieldName) {
			continue
		}

		// Get field value
		field := entityValue.FieldByName(prop.FieldName)
		if !field.IsValid() {
			continue
		}

		value := field.Interface()

		// Skip nil pointers and auto-generated IDs with nil value
		if field.Kind() == reflect.Ptr && field.IsNil() {
			if prop.IsID && prop.IDGenerator != nil {
				continue // Skip auto-generated IDs that are nil
			}
		}

		// Convert value to graph type
		var graphValue interface{}
		var err error

		if prop.Converter != nil {
			graphValue, err = prop.Converter.ToGraph(value)
		} else {
			graphValue, err = m.typeRegistry.ConvertToGraph(value, prop.Type)
		}

		if err != nil {
			return nil, fmt.Errorf("failed to convert property %s: %w", prop.FieldName, err)
		}

		if graphValue != nil {
			properties[prop.GraphName] = graphValue
		}
	}

	return properties, nil
}

// MapToCreate generates a Cypher CREATE statement for an entity
func (m *Mapper) MapToCreate(entity interface{}) (string, map[string]interface{}, error) {
	entityValue := reflect.ValueOf(entity)
	if entityValue.Kind() == reflect.Ptr {
		entityValue = entityValue.Elem()
	}

	entityType := entityValue.Type()
	metadata := m.registry.Get(entityType)
	if metadata == nil {
		return "", nil, fmt.Errorf("entity type %s is not registered", entityType.Name())
	}

	properties, err := m.ExtractProperties(entity)
	if err != nil {
		return "", nil, err
	}

	// Build labels string
	labelsStr := strings.Join(metadata.Labels, ":")

	// Build SET clauses for properties
	setClauses := make([]string, 0, len(properties))
	params := make(map[string]interface{})

	for _, prop := range metadata.Properties {
		if value, exists := properties[prop.GraphName]; exists {
			paramName := fmt.Sprintf("prop_%s", prop.GraphName)

			if prop.Interned && isStringValue(value) {
				// Use intern() function for interned string properties
				setClauses = append(setClauses, fmt.Sprintf("n.%s = intern($%s)", prop.GraphName, paramName))
			} else {
				setClauses = append(setClauses, fmt.Sprintf("n.%s = $%s", prop.GraphName, paramName))
			}

			params[paramName] = value
		}
	}

	// Generate Cypher query
	var cypher string
	if len(setClauses) > 0 {
		setStatement := strings.Join(setClauses, ", ")
		cypher = fmt.Sprintf("CREATE (n:%s) SET %s RETURN n, id(n) as node_id", labelsStr, setStatement)
	} else {
		cypher = fmt.Sprintf("CREATE (n:%s) RETURN n, id(n) as node_id", labelsStr)
	}

	return cypher, params, nil
}

// MapToMerge generates a Cypher MERGE statement for an entity (update or create)
func (m *Mapper) MapToMerge(entity interface{}) (string, map[string]interface{}, error) {
	entityValue := reflect.ValueOf(entity)
	if entityValue.Kind() == reflect.Ptr {
		entityValue = entityValue.Elem()
	}

	entityType := entityValue.Type()
	metadata := m.registry.Get(entityType)
	if metadata == nil {
		return "", nil, fmt.Errorf("entity type %s is not registered", entityType.Name())
	}

	if metadata.IDProperty == nil {
		return "", nil, fmt.Errorf("entity %s has no ID property", entityType.Name())
	}

	// Get entity ID
	idField := entityValue.FieldByName(metadata.IDProperty.FieldName)
	if !idField.IsValid() {
		return "", nil, fmt.Errorf("ID field %s not found", metadata.IDProperty.FieldName)
	}

	var entityID interface{}
	if idField.Kind() == reflect.Ptr {
		if idField.IsNil() {
			return "", nil, fmt.Errorf("entity ID is nil")
		}
		entityID = idField.Elem().Interface()
	} else {
		entityID = idField.Interface()
	}

	properties, err := m.ExtractProperties(entity)
	if err != nil {
		return "", nil, err
	}

	// Build labels string
	labelsStr := strings.Join(metadata.Labels, ":")

	// Build SET clauses (excluding ID)
	setClauses := make([]string, 0)
	params := map[string]interface{}{"id": entityID}

	for _, prop := range metadata.Properties {
		if prop.IsID {
			continue // Skip ID in SET clause
		}

		if value, exists := properties[prop.GraphName]; exists {
			paramName := fmt.Sprintf("prop_%s", prop.GraphName)

			if prop.Interned && isStringValue(value) {
				setClauses = append(setClauses, fmt.Sprintf("n.%s = intern($%s)", prop.GraphName, paramName))
			} else {
				setClauses = append(setClauses, fmt.Sprintf("n.%s = $%s", prop.GraphName, paramName))
			}

			params[paramName] = value
		}
	}

	// Generate Cypher query
	var cypher string
	if len(setClauses) > 0 {
		setStatement := strings.Join(setClauses, ", ")
		cypher = fmt.Sprintf("MERGE (n:%s {id: $id}) SET %s RETURN n, id(n) as node_id", labelsStr, setStatement)
	} else {
		cypher = fmt.Sprintf("MERGE (n:%s {id: $id}) RETURN n, id(n) as node_id", labelsStr)
	}

	return cypher, params, nil
}

// MapFromRecord converts a FalkorDB record to an entity instance
func (m *Mapper) MapFromRecord(record *falkordb.Record, entityType reflect.Type, varName string) (interface{}, error) {
	if entityType.Kind() == reflect.Ptr {
		entityType = entityType.Elem()
	}

	metadata := m.registry.Get(entityType)
	if metadata == nil {
		return nil, fmt.Errorf("entity type %s is not registered", entityType.Name())
	}

	// Get the node from record by index or variable name
	var node *falkordb.Node
	var nodeData interface{}
	var ok bool

	if varName != "" {
		nodeData, ok = record.Get(varName)
		if !ok {
			// Try index 0 as fallback
			nodeData, _ = record.GetByIndex(0)
		}
	} else {
		nodeData, _ = record.GetByIndex(0)
	}

	if nodeData == nil {
		return nil, fmt.Errorf("no node data in record")
	}

	if n, ok := nodeData.(*falkordb.Node); ok {
		node = n
	} else {
		return nil, fmt.Errorf("expected Node, got %T", nodeData)
	}

	// Create new instance of entity
	entityPtr := reflect.New(entityType)
	entityValue := entityPtr.Elem()

	// Map properties from node to entity
	for _, prop := range metadata.Properties {
		graphValue, exists := node.Properties[prop.GraphName]
		if !exists {
			continue
		}

		// Convert from graph type to Go type
		var goValue interface{}
		var err error

		if prop.Converter != nil {
			goValue, err = prop.Converter.FromGraph(graphValue, prop.Type)
		} else {
			goValue, err = m.typeRegistry.ConvertFromGraph(graphValue, prop.Type)
		}

		if err != nil {
			return nil, fmt.Errorf("failed to convert property %s: %w", prop.FieldName, err)
		}

		// Set field value
		field := entityValue.FieldByName(prop.FieldName)
		if field.IsValid() && field.CanSet() {
			m.setFieldValue(field, goValue)
		}
	}

	// Set node ID if entity has ID property
	if metadata.IDProperty != nil {
		idField := entityValue.FieldByName(metadata.IDProperty.FieldName)
		if idField.IsValid() && idField.CanSet() {
			m.setFieldValue(idField, int64(node.ID))
		}
	}

	return entityPtr.Interface(), nil
}

// UpdateEntityID updates the ID field of an entity after creation
func (m *Mapper) UpdateEntityID(entity interface{}, nodeID interface{}) error {
	entityValue := reflect.ValueOf(entity)
	if entityValue.Kind() == reflect.Ptr {
		entityValue = entityValue.Elem()
	}

	entityType := entityValue.Type()
	metadata := m.registry.Get(entityType)
	if metadata == nil || metadata.IDProperty == nil {
		return fmt.Errorf("entity has no ID property")
	}

	idField := entityValue.FieldByName(metadata.IDProperty.FieldName)
	if !idField.IsValid() || !idField.CanSet() {
		return fmt.Errorf("cannot set ID field")
	}

	m.setFieldValue(idField, nodeID)
	return nil
}

// setFieldValue sets a field value with proper type handling
func (m *Mapper) setFieldValue(field reflect.Value, value interface{}) {
	if value == nil {
		return
	}

	valueReflect := reflect.ValueOf(value)

	// Handle pointer fields
	if field.Kind() == reflect.Ptr {
		if valueReflect.Kind() != reflect.Ptr {
			// Create pointer to value
			ptr := reflect.New(field.Type().Elem())
			if valueReflect.Type().AssignableTo(field.Type().Elem()) {
				ptr.Elem().Set(valueReflect)
			} else if valueReflect.Type().ConvertibleTo(field.Type().Elem()) {
				ptr.Elem().Set(valueReflect.Convert(field.Type().Elem()))
			}
			field.Set(ptr)
		} else {
			field.Set(valueReflect)
		}
	} else {
		// Direct assignment or conversion
		if valueReflect.Type().AssignableTo(field.Type()) {
			field.Set(valueReflect)
		} else if valueReflect.Type().ConvertibleTo(field.Type()) {
			field.Set(valueReflect.Convert(field.Type()))
		} else if valueReflect.Kind() == reflect.Ptr && !valueReflect.IsNil() {
			// Dereference pointer value
			m.setFieldValue(field, valueReflect.Elem().Interface())
		}
	}
}

// isStringValue checks if a value is a string or pointer to string
func isStringValue(value interface{}) bool {
	if value == nil {
		return false
	}
	v := reflect.ValueOf(value)
	if v.Kind() == reflect.Ptr {
		if v.IsNil() {
			return false
		}
		v = v.Elem()
	}
	return v.Kind() == reflect.String
}
