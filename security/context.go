package security

import (
	"fmt"
	"strings"
	"sync"

	falkordb "github.com/FalkorDB/falkordb-go/v2"
)

// SecurityContext manages security for a user session
type SecurityContext struct {
	user           *User
	roles          []string
	privileges     []*Privilege
	privilegeCache map[string]bool // "action:resource" -> allowed
	mu             sync.RWMutex
	graph          *falkordb.Graph
}

// NewSecurityContext creates a new security context for a user
func NewSecurityContext(graph *falkordb.Graph, user *User) (*SecurityContext, error) {
	ctx := &SecurityContext{
		graph:          graph,
		user:           user,
		roles:          []string{},
		privileges:     []*Privilege{},
		privilegeCache: make(map[string]bool),
	}

	// Load user's roles and privileges from graph
	if err := ctx.loadRolesAndPrivileges(); err != nil {
		return nil, err
	}

	return ctx, nil
}

// GetUser returns the current user
func (c *SecurityContext) GetUser() *User {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.user
}

// GetRoles returns the user's roles
func (c *SecurityContext) GetRoles() []string {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return append([]string{}, c.roles...)
}

// HasRole checks if user has a specific role
func (c *SecurityContext) HasRole(role string) bool {
	c.mu.RLock()
	defer c.mu.RUnlock()

	for _, r := range c.roles {
		if r == role {
			return true
		}
	}
	return false
}

// IsAdmin checks if user has admin role
func (c *SecurityContext) IsAdmin() bool {
	return c.HasRole("admin")
}

// Can checks if user can perform an action on a resource
func (c *SecurityContext) Can(action, resource string) bool {
	c.mu.RLock()
	cacheKey := fmt.Sprintf("%s:%s", action, resource)
	if allowed, cached := c.privilegeCache[cacheKey]; cached {
		c.mu.RUnlock()
		return allowed
	}
	c.mu.RUnlock()

	// Check privileges
	allowed := c.checkPrivileges(action, resource)

	// Cache result
	c.mu.Lock()
	c.privilegeCache[cacheKey] = allowed
	c.mu.Unlock()

	return allowed
}

// GetDeniedProperties returns properties denied for reading for a specific entity label
func (c *SecurityContext) GetDeniedProperties(entityLabel, action string) []string {
	c.mu.RLock()
	defer c.mu.RUnlock()

	denied := []string{}

	for _, priv := range c.privileges {
		if priv.GrantType == GrantTypeDeny &&
			priv.Action == action &&
			priv.ResourceType == ResourceTypeProperty &&
			(priv.ResourceLabel == entityLabel || priv.ResourceLabel == "*") {

			// Check if this privilege applies to any of the user's roles
			denied = append(denied, priv.Property)
		}
	}

	return denied
}

// HasAnyRole checks if user has any of the specified roles
func (c *SecurityContext) HasAnyRole(roles []string) bool {
	if len(roles) == 0 {
		return true // No roles required means accessible to all
	}

	c.mu.RLock()
	defer c.mu.RUnlock()

	for _, required := range roles {
		for _, userRole := range c.roles {
			if userRole == required || userRole == "admin" {
				return true
			}
		}
	}
	return false
}

// checkPrivileges checks if user has privilege for action on resource
func (c *SecurityContext) checkPrivileges(action, resource string) bool {
	c.mu.RLock()
	defer c.mu.RUnlock()

	// Admin can do everything
	if c.IsAdmin() {
		return true
	}

	// Check for explicit DENY first
	for _, priv := range c.privileges {
		if priv.GrantType == GrantTypeDeny &&
			priv.Action == action &&
			matchesResource(priv, resource) {
			return false
		}
	}

	// Check for GRANT
	for _, priv := range c.privileges {
		if priv.GrantType == GrantTypeGrant &&
			priv.Action == action &&
			matchesResource(priv, resource) {
			return true
		}
	}

	// No explicit privilege found - deny by default
	return false
}

// matchesResource checks if a privilege matches a resource
func matchesResource(priv *Privilege, resource string) bool {
	// Handle wildcards
	if priv.ResourceLabel == "*" {
		return true
	}

	// Exact match
	if priv.ResourceLabel == resource {
		return true
	}

	// Property-level match (e.g., "Person.ssn")
	if strings.Contains(resource, ".") {
		parts := strings.Split(resource, ".")
		if len(parts) == 2 {
			entityLabel := parts[0]
			property := parts[1]
			return priv.ResourceLabel == entityLabel && priv.Property == property
		}
	}

	return false
}

// loadRolesAndPrivileges loads the user's roles and privileges from the graph
func (c *SecurityContext) loadRolesAndPrivileges() error {
	if c.user == nil || c.user.ID == nil {
		return fmt.Errorf("user must have an ID")
	}

	// Query to get roles and privileges
	query := `
		MATCH (u:User)-[:HAS_ROLE]->(r:Role)
		WHERE id(u) = $user_id
		OPTIONAL MATCH (r)-[:HAS_PRIVILEGE]->(p:Privilege)
		RETURN r.name as role_name, collect(p) as privileges
	`

	params := map[string]interface{}{
		"user_id": *c.user.ID,
	}

	result, err := c.graph.Query(query, params, nil)
	if err != nil {
		return fmt.Errorf("failed to load roles and privileges: %w", err)
	}

	roles := []string{}
	privileges := []*Privilege{}

	for result.Next() {
		record := result.Record()

		// Get role name
		if roleName, ok := record.Get("role_name"); ok && roleName != nil {
			if name, ok := roleName.(string); ok {
				roles = append(roles, name)
			}
		}

		// Get privileges
		if privData, ok := record.Get("privileges"); ok && privData != nil {
			// TODO: Parse privilege nodes from graph result
			// This is simplified - in reality would parse the node data
			// For now, we'll handle this in the manager when privileges are created
			_ = privData // Silence unused warning
		}
	}

	c.mu.Lock()
	c.roles = roles
	c.privileges = privileges
	c.mu.Unlock()

	return nil
}

// Refresh reloads roles and privileges from database
func (c *SecurityContext) Refresh() error {
	c.mu.Lock()
	c.privilegeCache = make(map[string]bool) // Clear cache
	c.mu.Unlock()

	return c.loadRolesAndPrivileges()
}

// AddRole adds a role to the context (for testing/impersonation)
func (c *SecurityContext) AddRole(role string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	for _, r := range c.roles {
		if r == role {
			return // Already has role
		}
	}

	c.roles = append(c.roles, role)
	c.privilegeCache = make(map[string]bool) // Clear cache
}

// AddPrivilege adds a privilege to the context (for testing)
func (c *SecurityContext) AddPrivilege(priv *Privilege) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.privileges = append(c.privileges, priv)
	c.privilegeCache = make(map[string]bool) // Clear cache
}
