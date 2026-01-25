package orm

import (
	"encoding/json"
	"fmt"
	"reflect"
	"sync"
	"time"

	"github.com/google/uuid"
)

// TypeRegistry manages type converters
type TypeRegistry struct {
	converters map[reflect.Type]TypeConverter
	mu         sync.RWMutex
}

// NewTypeRegistry creates a new type registry with built-in converters
func NewTypeRegistry() *TypeRegistry {
	registry := &TypeRegistry{
		converters: make(map[reflect.Type]TypeConverter),
	}

	// Register built-in converters
	registry.Register(reflect.TypeOf(time.Time{}), &TimeConverter{})
	registry.Register(reflect.TypeOf(uuid.UUID{}), &UUIDConverter{})

	return registry
}

// Register registers a type converter for a specific type
func (r *TypeRegistry) Register(typ reflect.Type, converter TypeConverter) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.converters[typ] = converter
}

// ConvertToGraph converts a Go value to a graph-compatible value
func (r *TypeRegistry) ConvertToGraph(value interface{}, targetType reflect.Type) (interface{}, error) {
	if value == nil {
		return nil, nil
	}

	valueType := reflect.TypeOf(value)

	// Check for custom converter
	r.mu.RLock()
	converter, exists := r.converters[valueType]
	r.mu.RUnlock()

	if exists {
		return converter.ToGraph(value)
	}

	// Handle basic types
	switch valueType.Kind() {
	case reflect.String, reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64,
		reflect.Float32, reflect.Float64, reflect.Bool:
		return value, nil

	case reflect.Slice, reflect.Array:
		return r.convertSliceToGraph(value)

	case reflect.Map:
		return r.convertMapToGraph(value)

	case reflect.Ptr:
		// Dereference pointer
		v := reflect.ValueOf(value)
		if v.IsNil() {
			return nil, nil
		}
		return r.ConvertToGraph(v.Elem().Interface(), targetType)

	case reflect.Struct:
		// Convert struct to JSON string for storage
		data, err := json.Marshal(value)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal struct to JSON: %w", err)
		}
		return string(data), nil

	default:
		return nil, fmt.Errorf("unsupported type for graph storage: %s", valueType)
	}
}

// ConvertFromGraph converts a graph value to a Go value
func (r *TypeRegistry) ConvertFromGraph(value interface{}, targetType reflect.Type) (interface{}, error) {
	if value == nil {
		return reflect.Zero(targetType).Interface(), nil
	}

	// Check for custom converter
	r.mu.RLock()
	converter, exists := r.converters[targetType]
	r.mu.RUnlock()

	if exists {
		return converter.FromGraph(value, targetType)
	}

	// Handle pointer types
	if targetType.Kind() == reflect.Ptr {
		elemType := targetType.Elem()
		elemValue, err := r.ConvertFromGraph(value, elemType)
		if err != nil {
			return nil, err
		}
		ptr := reflect.New(elemType)
		ptr.Elem().Set(reflect.ValueOf(elemValue))
		return ptr.Interface(), nil
	}

	// Direct type conversion for basic types
	valueType := reflect.TypeOf(value)
	if valueType.AssignableTo(targetType) {
		return value, nil
	}

	// Try conversion
	if valueType.ConvertibleTo(targetType) {
		return reflect.ValueOf(value).Convert(targetType).Interface(), nil
	}

	// Handle slice/array
	if targetType.Kind() == reflect.Slice || targetType.Kind() == reflect.Array {
		return r.convertSliceFromGraph(value, targetType)
	}

	// Handle map
	if targetType.Kind() == reflect.Map {
		return r.convertMapFromGraph(value, targetType)
	}

	// Handle struct - assume it's JSON encoded
	if targetType.Kind() == reflect.Struct {
		if str, ok := value.(string); ok {
			result := reflect.New(targetType).Interface()
			if err := json.Unmarshal([]byte(str), result); err != nil {
				return nil, fmt.Errorf("failed to unmarshal JSON to struct: %w", err)
			}
			return reflect.ValueOf(result).Elem().Interface(), nil
		}
	}

	return nil, fmt.Errorf("cannot convert %T to %s", value, targetType)
}

// convertSliceToGraph converts a slice to graph format
func (r *TypeRegistry) convertSliceToGraph(value interface{}) (interface{}, error) {
	v := reflect.ValueOf(value)
	result := make([]interface{}, v.Len())

	for i := 0; i < v.Len(); i++ {
		elem := v.Index(i).Interface()
		converted, err := r.ConvertToGraph(elem, reflect.TypeOf(elem))
		if err != nil {
			return nil, err
		}
		result[i] = converted
	}

	return result, nil
}

// convertSliceFromGraph converts a graph array to a Go slice
func (r *TypeRegistry) convertSliceFromGraph(value interface{}, targetType reflect.Type) (interface{}, error) {
	v := reflect.ValueOf(value)
	if v.Kind() != reflect.Slice && v.Kind() != reflect.Array {
		return nil, fmt.Errorf("expected slice/array, got %s", v.Kind())
	}

	elemType := targetType.Elem()
	result := reflect.MakeSlice(targetType, v.Len(), v.Len())

	for i := 0; i < v.Len(); i++ {
		elem := v.Index(i).Interface()
		converted, err := r.ConvertFromGraph(elem, elemType)
		if err != nil {
			return nil, err
		}
		result.Index(i).Set(reflect.ValueOf(converted))
	}

	return result.Interface(), nil
}

// convertMapToGraph converts a map to graph format
func (r *TypeRegistry) convertMapToGraph(value interface{}) (interface{}, error) {
	v := reflect.ValueOf(value)
	result := make(map[string]interface{})

	iter := v.MapRange()
	for iter.Next() {
		key := iter.Key().Interface()
		val := iter.Value().Interface()

		keyStr := fmt.Sprintf("%v", key)
		converted, err := r.ConvertToGraph(val, reflect.TypeOf(val))
		if err != nil {
			return nil, err
		}
		result[keyStr] = converted
	}

	return result, nil
}

// convertMapFromGraph converts a graph map to a Go map
func (r *TypeRegistry) convertMapFromGraph(value interface{}, targetType reflect.Type) (interface{}, error) {
	v := reflect.ValueOf(value)
	if v.Kind() != reflect.Map {
		return nil, fmt.Errorf("expected map, got %s", v.Kind())
	}

	result := reflect.MakeMap(targetType)
	keyType := targetType.Key()
	valueType := targetType.Elem()

	iter := v.MapRange()
	for iter.Next() {
		key := iter.Key().Interface()
		val := iter.Value().Interface()

		// Convert key
		convertedKey, err := r.ConvertFromGraph(key, keyType)
		if err != nil {
			return nil, err
		}

		// Convert value
		convertedVal, err := r.ConvertFromGraph(val, valueType)
		if err != nil {
			return nil, err
		}

		result.SetMapIndex(reflect.ValueOf(convertedKey), reflect.ValueOf(convertedVal))
	}

	return result.Interface(), nil
}

// TimeConverter converts time.Time to/from Unix timestamp
type TimeConverter struct{}

func (c *TimeConverter) ToGraph(value interface{}) (interface{}, error) {
	t, ok := value.(time.Time)
	if !ok {
		return nil, fmt.Errorf("expected time.Time, got %T", value)
	}
	return t.Unix(), nil
}

func (c *TimeConverter) FromGraph(value interface{}, targetType reflect.Type) (interface{}, error) {
	var timestamp int64

	switch v := value.(type) {
	case int64:
		timestamp = v
	case int:
		timestamp = int64(v)
	case float64:
		timestamp = int64(v)
	default:
		return nil, fmt.Errorf("expected numeric timestamp, got %T", value)
	}

	return time.Unix(timestamp, 0), nil
}

// UUIDConverter converts uuid.UUID to/from string
type UUIDConverter struct{}

func (c *UUIDConverter) ToGraph(value interface{}) (interface{}, error) {
	id, ok := value.(uuid.UUID)
	if !ok {
		return nil, fmt.Errorf("expected uuid.UUID, got %T", value)
	}
	return id.String(), nil
}

func (c *UUIDConverter) FromGraph(value interface{}, targetType reflect.Type) (interface{}, error) {
	str, ok := value.(string)
	if !ok {
		return nil, fmt.Errorf("expected string, got %T", value)
	}
	return uuid.Parse(str)
}

// Global type registry
var globalTypeRegistry = NewTypeRegistry()

// RegisterConverter registers a global type converter
func RegisterConverter(typ reflect.Type, converter TypeConverter) {
	globalTypeRegistry.Register(typ, converter)
}

// GetTypeRegistry returns the global type registry
func GetTypeRegistry() *TypeRegistry {
	return globalTypeRegistry
}
