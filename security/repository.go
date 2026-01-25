package security

import (
	"errors"
	"reflect"

	falkordb "github.com/FalkorDB/falkordb-go/v2"
	"github.com/falkordb/falkordb-go-orm/orm"
)

// ErrAccessDenied is returned when access is denied
var ErrAccessDenied = errors.New("access denied")

// SecureRepository wraps Repository with security checks
type SecureRepository[T any] struct {
	*orm.Repository[T]
	securityContext *SecurityContext
	securityMeta    *SecurityMetadata
	entityType      reflect.Type
	primaryLabel    string
}

// NewSecureRepository creates a new secure repository
func NewSecureRepository[T any](graph *falkordb.Graph, ctx *SecurityContext) *SecureRepository[T] {
	var zero T
	entityType := reflect.TypeOf(zero)
	if entityType.Kind() == reflect.Ptr {
		entityType = entityType.Elem()
	}

	// Get security metadata
	securityMeta := GetSecurityMetadata(entityType)

	// Get entity metadata for primary label
	entityMeta := orm.GetMetadata(entityType)
	primaryLabel := ""
	if entityMeta != nil {
		primaryLabel = entityMeta.PrimaryLabel
	}

	return &SecureRepository[T]{
		Repository:      orm.NewRepository[T](graph),
		securityContext: ctx,
		securityMeta:    securityMeta,
		entityType:      entityType,
		primaryLabel:    primaryLabel,
	}
}

// FindByID retrieves an entity by ID with security checks
func (r *SecureRepository[T]) FindByID(id interface{}, fetch ...string) (*T, error) {
	// Check READ permission
	if !r.canRead() {
		return nil, ErrAccessDenied
	}

	// Call base repository
	entity, err := r.Repository.FindByID(id, fetch...)
	if err != nil {
		return nil, err
	}

	// Filter denied properties
	r.filterDeniedProperties(entity)

	return entity, nil
}

// FindAll retrieves all entities with security checks
func (r *SecureRepository[T]) FindAll(fetch ...string) ([]*T, error) {
	// Check READ permission
	if !r.canRead() {
		return nil, ErrAccessDenied
	}

	// Call base repository
	entities, err := r.Repository.FindAll(fetch...)
	if err != nil {
		return nil, err
	}

	// Filter denied properties for each entity
	for _, entity := range entities {
		r.filterDeniedProperties(entity)
	}

	return entities, nil
}

// Save creates or updates an entity with security checks
func (r *SecureRepository[T]) Save(entity *T) (*T, error) {
	// Check if it's create or update
	isCreate := r.isNewEntity(entity)

	if isCreate {
		if !r.canCreate() {
			return nil, ErrAccessDenied
		}
	} else {
		if !r.canWrite() {
			return nil, ErrAccessDenied
		}
	}

	// Filter denied properties before save
	r.filterDeniedWriteProperties(entity)

	// Call base repository
	return r.Repository.Save(entity)
}

// Delete deletes an entity with security checks
func (r *SecureRepository[T]) Delete(entity *T) error {
	if !r.canDelete() {
		return ErrAccessDenied
	}

	return r.Repository.Delete(entity)
}

// DeleteByID deletes an entity by ID with security checks
func (r *SecureRepository[T]) DeleteByID(id interface{}) error {
	if !r.canDelete() {
		return ErrAccessDenied
	}

	return r.Repository.DeleteByID(id)
}

// Permission check helpers

func (r *SecureRepository[T]) canRead() bool {
	if r.securityMeta == nil {
		return true // No security metadata means no restrictions
	}

	return r.securityContext.HasAnyRole(r.securityMeta.ReadRoles) ||
		r.securityContext.Can(ActionRead, r.primaryLabel)
}

func (r *SecureRepository[T]) canWrite() bool {
	if r.securityMeta == nil {
		return true
	}

	return r.securityContext.HasAnyRole(r.securityMeta.WriteRoles) ||
		r.securityContext.Can(ActionWrite, r.primaryLabel)
}

func (r *SecureRepository[T]) canCreate() bool {
	if r.securityMeta == nil {
		return true
	}

	return r.securityContext.HasAnyRole(r.securityMeta.CreateRoles) ||
		r.securityContext.Can(ActionCreate, r.primaryLabel)
}

func (r *SecureRepository[T]) canDelete() bool {
	if r.securityMeta == nil {
		return true
	}

	return r.securityContext.HasAnyRole(r.securityMeta.DeleteRoles) ||
		r.securityContext.Can(ActionDelete, r.primaryLabel)
}

// filterDeniedProperties filters out properties the user cannot read
func (r *SecureRepository[T]) filterDeniedProperties(entity *T) {
	if r.securityMeta == nil || entity == nil {
		return
	}

	userRoles := r.securityContext.GetRoles()
	entityValue := reflect.ValueOf(entity)
	if entityValue.Kind() == reflect.Ptr {
		entityValue = entityValue.Elem()
	}

	// Get entity metadata
	entityMeta := orm.GetMetadata(r.entityType)
	if entityMeta == nil {
		return
	}

	// Check each property
	for _, prop := range entityMeta.Properties {
		// Check if property is denied for any of user's roles
		if deniedRoles, exists := r.securityMeta.DenyReadProperties[prop.GraphName]; exists {
			if r.isRoleDenied(userRoles, deniedRoles) {
				// Set field to zero value
				field := entityValue.FieldByName(prop.FieldName)
				if field.IsValid() && field.CanSet() {
					field.Set(reflect.Zero(field.Type()))
				}
			}
		}
	}
}

// filterDeniedWriteProperties filters out properties the user cannot write
func (r *SecureRepository[T]) filterDeniedWriteProperties(entity *T) {
	if r.securityMeta == nil || entity == nil {
		return
	}

	userRoles := r.securityContext.GetRoles()
	entityValue := reflect.ValueOf(entity)
	if entityValue.Kind() == reflect.Ptr {
		entityValue = entityValue.Elem()
	}

	// Get entity metadata
	entityMeta := orm.GetMetadata(r.entityType)
	if entityMeta == nil {
		return
	}

	// Check each property
	for _, prop := range entityMeta.Properties {
		// Check if property is denied for any of user's roles
		if deniedRoles, exists := r.securityMeta.DenyWriteProperties[prop.GraphName]; exists {
			if r.isRoleDenied(userRoles, deniedRoles) {
				// Set field to zero value (prevent write)
				field := entityValue.FieldByName(prop.FieldName)
				if field.IsValid() && field.CanSet() {
					field.Set(reflect.Zero(field.Type()))
				}
			}
		}
	}
}

// isRoleDenied checks if any user role is in denied roles
func (r *SecureRepository[T]) isRoleDenied(userRoles, deniedRoles []string) bool {
	for _, denied := range deniedRoles {
		if denied == "*" {
			return true // Denied for everyone
		}

		for _, userRole := range userRoles {
			if userRole == denied {
				return true
			}
		}
	}
	return false
}

// isNewEntity checks if entity is new (no ID)
func (r *SecureRepository[T]) isNewEntity(entity *T) bool {
	entityValue := reflect.ValueOf(entity)
	if entityValue.Kind() == reflect.Ptr {
		entityValue = entityValue.Elem()
	}

	entityMeta := orm.GetMetadata(r.entityType)
	if entityMeta == nil || entityMeta.IDProperty == nil {
		return false
	}

	idField := entityValue.FieldByName(entityMeta.IDProperty.FieldName)
	if !idField.IsValid() {
		return true
	}

	// Check if ID is nil (for pointer types)
	if idField.Kind() == reflect.Ptr {
		return idField.IsNil()
	}

	// Check if ID is zero value
	return idField.Interface() == reflect.Zero(idField.Type()).Interface()
}

// GetSecurityContext returns the security context
func (r *SecureRepository[T]) GetSecurityContext() *SecurityContext {
	return r.securityContext
}
