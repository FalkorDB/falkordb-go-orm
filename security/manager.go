package security

import (
	"fmt"
	"time"

	falkordb "github.com/FalkorDB/falkordb-go/v2"
	"github.com/falkordb/falkordb-go-orm/orm"
)

// RBACManager provides administrative operations for RBAC
type RBACManager struct {
	graph         *falkordb.Graph
	userRepo      *orm.Repository[User]
	roleRepo      *orm.Repository[Role]
	privilegeRepo *orm.Repository[Privilege]
	auditRepo     *orm.Repository[AuditLog]
}

// NewRBACManager creates a new RBAC manager
func NewRBACManager(graph *falkordb.Graph) *RBACManager {
	// Register entities
	orm.GetEntityRegistry().Register(&User{})
	orm.GetEntityRegistry().Register(&Role{})
	orm.GetEntityRegistry().Register(&Privilege{})
	orm.GetEntityRegistry().Register(&AuditLog{})

	return &RBACManager{
		graph:         graph,
		userRepo:      orm.NewRepository[User](graph),
		roleRepo:      orm.NewRepository[Role](graph),
		privilegeRepo: orm.NewRepository[Privilege](graph),
		auditRepo:     orm.NewRepository[AuditLog](graph),
	}
}

// CreateUser creates a new user
func (m *RBACManager) CreateUser(username, email string) (*User, error) {
	user := &User{
		Username:  username,
		Email:     email,
		IsActive:  true,
		CreatedAt: time.Now(),
	}

	return m.userRepo.Save(user)
}

// CreateRole creates a new role
func (m *RBACManager) CreateRole(name, description string, immutable bool) (*Role, error) {
	role := &Role{
		Name:        name,
		Description: description,
		Immutable:   immutable,
		CreatedAt:   time.Now(),
	}

	return m.roleRepo.Save(role)
}

// AssignRole assigns a role to a user
func (m *RBACManager) AssignRole(username, roleName string) error {
	// Find user by username
	users, err := m.userRepo.FindAll()
	if err != nil {
		return err
	}

	var user *User
	for _, u := range users {
		if u.Username == username {
			user = u
			break
		}
	}

	if user == nil {
		return fmt.Errorf("user not found: %s", username)
	}

	// Find role by name
	roles, err := m.roleRepo.FindAll()
	if err != nil {
		return err
	}

	var role *Role
	for _, r := range roles {
		if r.Name == roleName {
			role = r
			break
		}
	}

	if role == nil {
		return fmt.Errorf("role not found: %s", roleName)
	}

	// Create HAS_ROLE relationship
	query := `
		MATCH (u:User) WHERE id(u) = $user_id
		MATCH (r:Role) WHERE id(r) = $role_id
		MERGE (u)-[:HAS_ROLE]->(r)
	`

	params := map[string]interface{}{
		"user_id": *user.ID,
		"role_id": *role.ID,
	}

	_, err = m.graph.Query(query, params, nil)
	return err
}

// GrantPrivilege grants a privilege to a role
func (m *RBACManager) GrantPrivilege(roleName, action, resourceType, resourceLabel string) error {
	return m.createPrivilege(roleName, action, resourceType, resourceLabel, "", GrantTypeGrant)
}

// DenyPrivilege denies a privilege for a role
func (m *RBACManager) DenyPrivilege(roleName, action, resourceType, resourceLabel string) error {
	return m.createPrivilege(roleName, action, resourceType, resourceLabel, "", GrantTypeDeny)
}

// GrantPropertyPrivilege grants access to a specific property
func (m *RBACManager) GrantPropertyPrivilege(roleName, action, resourceLabel, property string) error {
	return m.createPrivilege(roleName, action, ResourceTypeProperty, resourceLabel, property, GrantTypeGrant)
}

// DenyPropertyPrivilege denies access to a specific property
func (m *RBACManager) DenyPropertyPrivilege(roleName, action, resourceLabel, property string) error {
	return m.createPrivilege(roleName, action, ResourceTypeProperty, resourceLabel, property, GrantTypeDeny)
}

// createPrivilege creates a privilege and assigns it to a role
func (m *RBACManager) createPrivilege(roleName, action, resourceType, resourceLabel, property, grantType string) error {
	// Find role
	roles, err := m.roleRepo.FindAll()
	if err != nil {
		return err
	}

	var role *Role
	for _, r := range roles {
		if r.Name == roleName {
			role = r
			break
		}
	}

	if role == nil {
		return fmt.Errorf("role not found: %s", roleName)
	}

	// Create privilege
	privilege := &Privilege{
		Action:        action,
		ResourceType:  resourceType,
		ResourceLabel: resourceLabel,
		Property:      property,
		GrantType:     grantType,
		CreatedAt:     time.Now(),
	}

	savedPriv, err := m.privilegeRepo.Save(privilege)
	if err != nil {
		return err
	}

	// Create HAS_PRIVILEGE relationship
	query := `
		MATCH (r:Role) WHERE id(r) = $role_id
		MATCH (p:Privilege) WHERE id(p) = $privilege_id
		MERGE (r)-[:HAS_PRIVILEGE]->(p)
	`

	params := map[string]interface{}{
		"role_id":      *role.ID,
		"privilege_id": *savedPriv.ID,
	}

	_, err = m.graph.Query(query, params, nil)
	return err
}

// LogAccess logs an access attempt
func (m *RBACManager) LogAccess(username, action, resource string, success bool, details string) error {
	log := &AuditLog{
		Username:  username,
		Action:    action,
		Resource:  resource,
		Timestamp: time.Now(),
		Success:   success,
		Details:   details,
	}

	_, err := m.auditRepo.Save(log)
	return err
}

// GetUserRoles returns roles for a user
func (m *RBACManager) GetUserRoles(username string) ([]string, error) {
	users, err := m.userRepo.FindAll()
	if err != nil {
		return nil, err
	}

	var user *User
	for _, u := range users {
		if u.Username == username {
			user = u
			break
		}
	}

	if user == nil {
		return nil, fmt.Errorf("user not found: %s", username)
	}

	query := `
		MATCH (u:User)-[:HAS_ROLE]->(r:Role)
		WHERE id(u) = $user_id
		RETURN r.name as role_name
	`

	params := map[string]interface{}{
		"user_id": *user.ID,
	}

	result, err := m.graph.Query(query, params, nil)
	if err != nil {
		return nil, err
	}

	roles := []string{}
	for result.Next() {
		record := result.Record()
		if roleName, ok := record.Get("role_name"); ok && roleName != nil {
			if name, ok := roleName.(string); ok {
				roles = append(roles, name)
			}
		}
	}

	return roles, nil
}
