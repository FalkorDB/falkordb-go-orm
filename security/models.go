package security

import "time"

// User represents a user in the system
type User struct {
	ID        *int64    `falkor:"id,generated"`
	Username  string    `falkor:"username,unique,required"`
	Email     string    `falkor:"email,unique,required"`
	IsActive  bool      `falkor:"is_active"`
	CreatedAt time.Time `falkor:"created_at"`
}

func (*User) Labels() []string {
	return []string{"User"}
}

// Role represents a role that can be assigned to users
type Role struct {
	ID          *int64    `falkor:"id,generated"`
	Name        string    `falkor:"name,unique,required"`
	Description string    `falkor:"description"`
	Immutable   bool      `falkor:"immutable"`
	CreatedAt   time.Time `falkor:"created_at"`
}

func (*Role) Labels() []string {
	return []string{"Role"}
}

// Privilege represents a permission for a role
type Privilege struct {
	ID            *int64    `falkor:"id,generated"`
	Action        string    `falkor:"action,required"`        // READ, WRITE, CREATE, DELETE, TRAVERSE
	ResourceType  string    `falkor:"resource_type,required"` // NODE, RELATIONSHIP, PROPERTY
	ResourceLabel string    `falkor:"resource_label"`         // Person, KNOWS, etc.
	Property      string    `falkor:"property"`               // For property-level access
	GrantType     string    `falkor:"grant_type,required"`    // GRANT, DENY
	CreatedAt     time.Time `falkor:"created_at"`
}

func (*Privilege) Labels() []string {
	return []string{"Privilege"}
}

// AuditLog represents an audit log entry
type AuditLog struct {
	ID        *int64    `falkor:"id,generated"`
	UserID    int64     `falkor:"user_id"`
	Username  string    `falkor:"username,required"`
	Action    string    `falkor:"action,required"`
	Resource  string    `falkor:"resource,required"`
	Timestamp time.Time `falkor:"timestamp,required"`
	Success   bool      `falkor:"success"`
	Details   string    `falkor:"details"`
}

func (*AuditLog) Labels() []string {
	return []string{"AuditLog"}
}

// Action constants
const (
	ActionRead     = "READ"
	ActionWrite    = "WRITE"
	ActionCreate   = "CREATE"
	ActionDelete   = "DELETE"
	ActionTraverse = "TRAVERSE"
)

// ResourceType constants
const (
	ResourceTypeNode         = "NODE"
	ResourceTypeRelationship = "RELATIONSHIP"
	ResourceTypeProperty     = "PROPERTY"
)

// GrantType constants
const (
	GrantTypeGrant = "GRANT"
	GrantTypeDeny  = "DENY"
)
