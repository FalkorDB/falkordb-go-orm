package security

import (
	"reflect"
	"sync"
)

// SecurityMetadata defines access control rules for an entity type
type SecurityMetadata struct {
	// Entity-level permissions (roles required)
	ReadRoles   []string
	WriteRoles  []string
	CreateRoles []string
	DeleteRoles []string

	// Property-level denial (property -> roles denied)
	DenyReadProperties  map[string][]string
	DenyWriteProperties map[string][]string

	// Row-level security
	RowLevelSecurity *RowLevelSecurityRule
}

// RowLevelSecurityRule defines row-level security constraints
type RowLevelSecurityRule struct {
	// Cypher WHERE clause to filter results
	Condition string
	// Roles this rule applies to
	Roles []string
}

// SecurityMetadataRegistry manages security metadata for entity types
type SecurityMetadataRegistry struct {
	mu       sync.RWMutex
	metadata map[reflect.Type]*SecurityMetadata
}

// NewSecurityMetadataRegistry creates a new registry
func NewSecurityMetadataRegistry() *SecurityMetadataRegistry {
	return &SecurityMetadataRegistry{
		metadata: make(map[reflect.Type]*SecurityMetadata),
	}
}

// Register registers security metadata for an entity type
func (r *SecurityMetadataRegistry) Register(entityType reflect.Type, meta *SecurityMetadata) {
	if entityType.Kind() == reflect.Ptr {
		entityType = entityType.Elem()
	}

	r.mu.Lock()
	defer r.mu.Unlock()
	r.metadata[entityType] = meta
}

// Get retrieves security metadata for an entity type
func (r *SecurityMetadataRegistry) Get(entityType reflect.Type) *SecurityMetadata {
	if entityType.Kind() == reflect.Ptr {
		entityType = entityType.Elem()
	}

	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.metadata[entityType]
}

// Global security metadata registry
var globalSecurityRegistry = NewSecurityMetadataRegistry()

// RegisterSecurityMetadata registers security metadata globally
func RegisterSecurityMetadata(entityType reflect.Type, meta *SecurityMetadata) {
	globalSecurityRegistry.Register(entityType, meta)
}

// GetSecurityMetadata retrieves security metadata for an entity type
func GetSecurityMetadata(entityType reflect.Type) *SecurityMetadata {
	return globalSecurityRegistry.Get(entityType)
}

// Helper functions for creating security metadata

// NewSecurityMetadata creates a new SecurityMetadata with default values
func NewSecurityMetadata() *SecurityMetadata {
	return &SecurityMetadata{
		ReadRoles:           []string{},
		WriteRoles:          []string{},
		CreateRoles:         []string{},
		DeleteRoles:         []string{},
		DenyReadProperties:  make(map[string][]string),
		DenyWriteProperties: make(map[string][]string),
	}
}

// WithReadRoles sets the read roles
func (m *SecurityMetadata) WithReadRoles(roles ...string) *SecurityMetadata {
	m.ReadRoles = roles
	return m
}

// WithWriteRoles sets the write roles
func (m *SecurityMetadata) WithWriteRoles(roles ...string) *SecurityMetadata {
	m.WriteRoles = roles
	return m
}

// WithCreateRoles sets the create roles
func (m *SecurityMetadata) WithCreateRoles(roles ...string) *SecurityMetadata {
	m.CreateRoles = roles
	return m
}

// WithDeleteRoles sets the delete roles
func (m *SecurityMetadata) WithDeleteRoles(roles ...string) *SecurityMetadata {
	m.DeleteRoles = roles
	return m
}

// DenyReadProperty denies read access to a property for specific roles
func (m *SecurityMetadata) DenyReadProperty(property string, roles ...string) *SecurityMetadata {
	m.DenyReadProperties[property] = roles
	return m
}

// DenyWriteProperty denies write access to a property for specific roles
func (m *SecurityMetadata) DenyWriteProperty(property string, roles ...string) *SecurityMetadata {
	m.DenyWriteProperties[property] = roles
	return m
}

// WithRowLevelSecurity sets row-level security rule
func (m *SecurityMetadata) WithRowLevelSecurity(condition string, roles ...string) *SecurityMetadata {
	m.RowLevelSecurity = &RowLevelSecurityRule{
		Condition: condition,
		Roles:     roles,
	}
	return m
}
