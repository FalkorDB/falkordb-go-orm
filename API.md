# FalkorDB Go ORM API Reference

Complete API documentation for the `orm` and `security` packages.

## Table of Contents

- [orm Package](#orm-package)
  - [EntityRegistry](#entityregistry)
  - [Repository[T]](#repositoryt)
  - [QueryBuilder](#querybuilder)
  - [Session](#session)
  - [RelationshipManager](#relationshipmanager)
  - [IndexManager](#indexmanager)
  - [SchemaManager](#schemamanager)
  - [Type Converters](#type-converters)
- [security Package](#security-package)
  - [SecurityContext](#securitycontext)
  - [SecureRepository[T]](#securerepository)
  - [RBACManager](#rbacmanager)

## orm Package

### EntityRegistry

Thread-safe registry for entity metadata.

#### NewEntityRegistry

```go
func NewEntityRegistry() *EntityRegistry
```

Creates a new entity registry.

**Example**:
```go
registry := orm.NewEntityRegistry()
```

#### Register

```go
func (r *EntityRegistry) Register(entity interface{}) error
```

Registers an entity type with the registry. The entity must have a `Labels() []string` method.

**Parameters**:
- `entity`: Pointer to an instance of the entity type

**Returns**: Error if registration fails

**Example**:
```go
type User struct {
    ID   uuid.UUID `falkor:"id,generated"`
    Name string    `falkor:"name"`
}

func (u User) Labels() []string {
    return []string{"User"}
}

err := registry.Register(&User{})
```

#### GetMetadata

```go
func (r *EntityRegistry) GetMetadata(entityType reflect.Type) (*EntityMetadata, error)
```

Retrieves metadata for a registered entity type.

**Parameters**:
- `entityType`: The reflect.Type of the entity

**Returns**: EntityMetadata and error

---

### Repository[T]

Generic, type-safe repository for CRUD operations.

#### NewRepository

```go
func NewRepository[T any](graph *falkordb.Graph, registry *EntityRegistry) *Repository[T]
```

Creates a new repository for type T.

**Type Parameters**:
- `T`: Entity type (must be pointer, e.g., `*User`)

**Parameters**:
- `graph`: FalkorDB graph connection
- `registry`: Entity registry with registered types

**Example**:
```go
repo := orm.NewRepository[*User](graph, registry)
```

#### Save

```go
func (r *Repository[T]) Save(entity T) error
```

Saves an entity. Creates new entity if ID is empty, updates if ID exists.

**Parameters**:
- `entity`: Entity instance to save

**Returns**: Error if save fails

**Example**:
```go
user := &User{Name: "Alice", Email: "alice@example.com"}
err := repo.Save(user)
// user.ID is now populated
```

#### SaveAll

```go
func (r *Repository[T]) SaveAll(entities []T) error
```

Saves multiple entities in batch.

**Parameters**:
- `entities`: Slice of entities to save

**Returns**: Error if any save fails

**Example**:
```go
users := []*User{
    {Name: "Alice"},
    {Name: "Bob"},
}
err := repo.SaveAll(users)
```

#### FindByID

```go
func (r *Repository[T]) FindByID(id uuid.UUID) (T, error)
```

Finds an entity by its ID.

**Parameters**:
- `id`: UUID of the entity

**Returns**: Entity instance and error

**Example**:
```go
user, err := repo.FindByID(userID)
if err != nil {
    // Handle not found or error
}
```

#### FindAll

```go
func (r *Repository[T]) FindAll() ([]T, error)
```

Finds all entities of type T.

**Returns**: Slice of entities and error

**Example**:
```go
users, err := repo.FindAll()
```

#### FindAllByID

```go
func (r *Repository[T]) FindAllByID(ids []uuid.UUID) ([]T, error)
```

Finds multiple entities by their IDs.

**Parameters**:
- `ids`: Slice of UUIDs

**Returns**: Slice of entities and error

#### ExistsByID

```go
func (r *Repository[T]) ExistsByID(id uuid.UUID) (bool, error)
```

Checks if an entity exists by ID.

**Parameters**:
- `id`: UUID of the entity

**Returns**: Boolean existence status and error

**Example**:
```go
exists, err := repo.ExistsByID(userID)
```

#### Count

```go
func (r *Repository[T]) Count() (int64, error)
```

Counts all entities of type T.

**Returns**: Count and error

**Example**:
```go
count, err := repo.Count()
fmt.Printf("Total users: %d\n", count)
```

#### Delete

```go
func (r *Repository[T]) Delete(entity T) error
```

Deletes an entity by its ID.

**Parameters**:
- `entity`: Entity instance to delete

**Returns**: Error if delete fails

#### DeleteByID

```go
func (r *Repository[T]) DeleteByID(id uuid.UUID) error
```

Deletes an entity by ID.

**Parameters**:
- `id`: UUID of entity to delete

**Returns**: Error if delete fails

#### DeleteAll

```go
func (r *Repository[T]) DeleteAll() error
```

Deletes all entities of type T.

**Returns**: Error if delete fails

**Warning**: This deletes ALL entities of the type!

#### Query

```go
func (r *Repository[T]) Query() *QueryBuilderChain[T]
```

Creates a query builder for complex queries.

**Returns**: QueryBuilderChain for method chaining

**Example**:
```go
users, err := repo.Query().
    Where("age > $age", map[string]interface{}{"age": 18}).
    OrderBy("name ASC").
    FindAll()
```

---

### QueryBuilder

Fluent API for building complex Cypher queries.

#### Where

```go
func (q *QueryBuilderChain[T]) Where(condition string, params map[string]interface{}) *QueryBuilderChain[T]
```

Adds a WHERE clause to the query.

**Parameters**:
- `condition`: Cypher WHERE condition (use $param for parameters)
- `params`: Parameter values

**Returns**: Self for chaining

**Example**:
```go
repo.Query().
    Where("age > $minAge AND city = $city", map[string]interface{}{
        "minAge": 21,
        "city":   "New York",
    })
```

#### OrderBy

```go
func (q *QueryBuilderChain[T]) OrderBy(order string) *QueryBuilderChain[T]
```

Adds ORDER BY clause.

**Parameters**:
- `order`: Order specification (e.g., "name ASC", "age DESC")

**Returns**: Self for chaining

**Example**:
```go
repo.Query().OrderBy("created_at DESC")
```

#### Limit

```go
func (q *QueryBuilderChain[T]) Limit(limit int) *QueryBuilderChain[T]
```

Adds LIMIT clause.

**Parameters**:
- `limit`: Maximum number of results

**Returns**: Self for chaining

#### Skip

```go
func (q *QueryBuilderChain[T]) Skip(skip int) *QueryBuilderChain[T]
```

Adds SKIP clause for pagination.

**Parameters**:
- `skip`: Number of results to skip

**Returns**: Self for chaining

#### Fetch

```go
func (q *QueryBuilderChain[T]) Fetch(relationships ...string) *QueryBuilderChain[T]
```

Eager loads relationships.

**Parameters**:
- `relationships`: Names of relationship fields to load

**Returns**: Self for chaining

**Example**:
```go
posts, err := repo.Query().
    Fetch("Author", "Tags", "Comments").
    FindAll()
// Author, Tags, Comments are preloaded
```

#### Paginate

```go
func (q *QueryBuilderChain[T]) Paginate(pageable Pageable) *QueryBuilderChain[T]
```

Adds pagination.

**Parameters**:
- `pageable`: Pageable instance with page number and size

**Returns**: Self for chaining

**Example**:
```go
pageable := orm.NewPageable(0, 20) // page 0, 20 items
page, err := repo.Query().Paginate(pageable).FindPage()
```

#### FindAll

```go
func (q *QueryBuilderChain[T]) FindAll() ([]T, error)
```

Executes query and returns all matching entities.

**Returns**: Slice of entities and error

#### FindPage

```go
func (q *QueryBuilderChain[T]) FindPage() (*Page[T], error)
```

Executes query and returns paginated results.

**Returns**: Page object and error

#### Count

```go
func (q *QueryBuilderChain[T]) Count() (int64, error)
```

Counts matching entities.

**Returns**: Count and error

---

### Session

Provides transaction support and identity map pattern.

#### NewSession

```go
func NewSession(graph *falkordb.Graph, registry *EntityRegistry) *Session
```

Creates a new session.

**Parameters**:
- `graph`: FalkorDB graph connection
- `registry`: Entity registry

**Example**:
```go
session := orm.NewSession(graph, registry)
```

#### Save

```go
func (s *Session) Save(entity interface{}) error
```

Saves an entity within the session. Tracks changes.

**Parameters**:
- `entity`: Entity to save

**Returns**: Error if save fails

#### FindByID

```go
func (s *Session) FindByID(entityType interface{}, id uuid.UUID) (interface{}, error)
```

Finds entity by ID. Checks identity map first.

**Parameters**:
- `entityType`: Example entity instance for type
- `id`: Entity UUID

**Returns**: Entity and error

#### Delete

```go
func (s *Session) Delete(entity interface{}) error
```

Marks entity for deletion.

**Parameters**:
- `entity`: Entity to delete

**Returns**: Error

#### BeginTransaction

```go
func (s *Session) BeginTransaction() error
```

Starts a database transaction.

**Returns**: Error if transaction start fails

#### Commit

```go
func (s *Session) Commit() error
```

Commits all tracked changes and the transaction.

**Returns**: Error if commit fails

#### Rollback

```go
func (s *Session) Rollback() error
```

Rolls back the transaction and clears tracked changes.

**Returns**: Error if rollback fails

#### WithSession

```go
func WithSession(session *Session, fn func(*Session) error) error
```

Executes function within session, automatically managing lifecycle.

**Parameters**:
- `session`: Session instance
- `fn`: Function to execute within session

**Returns**: Error from function or session operations

**Example**:
```go
err := orm.WithSession(session, func(s *orm.Session) error {
    user := &User{Name: "Alice"}
    s.Save(user)
    
    user.Age = 30
    s.Save(user) // Tracked as dirty
    
    return s.Commit()
})
```

---

### RelationshipManager

Manages entity relationships.

#### NewRelationshipManager

```go
func NewRelationshipManager(graph *falkordb.Graph, registry *EntityRegistry) *RelationshipManager
```

Creates a new relationship manager.

**Example**:
```go
relMgr := orm.NewRelationshipManager(graph, registry)
```

#### SaveRelationships

```go
func (rm *RelationshipManager) SaveRelationships(
    fromEntity interface{},
    fieldName string,
    toEntities []interface{},
) error
```

Creates relationships between entities.

**Parameters**:
- `fromEntity`: Source entity
- `fieldName`: Name of relationship field
- `toEntities`: Target entities

**Returns**: Error if creation fails

**Example**:
```go
err := relMgr.SaveRelationships(user, "Friends", []interface{}{friend1, friend2})
```

#### DeleteRelationships

```go
func (rm *RelationshipManager) DeleteRelationships(
    fromEntity interface{},
    fieldName string,
    toEntities []interface{},
) error
```

Deletes relationships between entities.

**Parameters**:
- `fromEntity`: Source entity
- `fieldName`: Name of relationship field
- `toEntities`: Target entities (nil for all)

**Returns**: Error if deletion fails

#### LoadRelationships

```go
func (rm *RelationshipManager) LoadRelationships(
    entity interface{},
    fieldName string,
) ([]interface{}, error)
```

Loads related entities.

**Parameters**:
- `entity`: Source entity
- `fieldName`: Name of relationship field

**Returns**: Slice of related entities and error

---

### IndexManager

Manages database indexes.

#### NewIndexManager

```go
func NewIndexManager(graph *falkordb.Graph) *IndexManager
```

Creates a new index manager.

**Example**:
```go
indexMgr := orm.NewIndexManager(graph)
```

#### CreateIndexes

```go
func (im *IndexManager) CreateIndexes(entity interface{}, indexType IndexType) error
```

Creates indexes for all indexed properties in an entity.

**Parameters**:
- `entity`: Entity instance
- `indexType`: RANGE, FULLTEXT, or VECTOR

**Returns**: Error if creation fails

**Example**:
```go
err := indexMgr.CreateIndexes(&User{}, orm.IndexTypeRange)
```

#### DropIndexes

```go
func (im *IndexManager) DropIndexes(entity interface{}) error
```

Drops all indexes for an entity.

**Parameters**:
- `entity`: Entity instance

**Returns**: Error if drop fails

#### ListIndexes

```go
func (im *IndexManager) ListIndexes(label string) ([]IndexInfo, error)
```

Lists indexes for a label.

**Parameters**:
- `label`: Node label

**Returns**: Slice of IndexInfo and error

#### EnsureIndexes

```go
func (im *IndexManager) EnsureIndexes(entity interface{}, indexType IndexType) error
```

Creates indexes if they don't exist (idempotent).

**Parameters**:
- `entity`: Entity instance
- `indexType`: Index type

**Returns**: Error if operation fails

---

### SchemaManager

Validates and syncs entity schemas.

#### NewSchemaManager

```go
func NewSchemaManager(registry *EntityRegistry, graph *falkordb.Graph) *SchemaManager
```

Creates a new schema manager.

**Example**:
```go
schemaMgr := orm.NewSchemaManager(registry, graph)
```

#### ValidateSchema

```go
func (sm *SchemaManager) ValidateSchema(entity interface{}) (*ValidationResult, error)
```

Validates an entity's schema definition.

**Parameters**:
- `entity`: Entity instance to validate

**Returns**: ValidationResult with errors and error

**Example**:
```go
result, err := schemaMgr.ValidateSchema(&User{})
if !result.IsValid {
    for _, e := range result.Errors {
        log.Println(e)
    }
}
```

#### SyncSchema

```go
func (sm *SchemaManager) SyncSchema(indexType IndexType) error
```

Syncs schemas for all registered entities (creates missing indexes).

**Parameters**:
- `indexType`: Index type to create

**Returns**: Error if sync fails

---

### Type Converters

#### GetTypeRegistry

```go
func GetTypeRegistry() *TypeRegistry
```

Returns the global type registry.

**Example**:
```go
registry := orm.GetTypeRegistry()
```

#### Register (TypeRegistry)

```go
func (tr *TypeRegistry) Register(goType reflect.Type, converter TypeConverter) error
```

Registers a custom type converter.

**Parameters**:
- `goType`: reflect.Type of Go type
- `converter`: TypeConverter implementation

**Returns**: Error if registration fails

**Example**:
```go
type PointConverter struct{}

func (c *PointConverter) ToGraph(value interface{}) (interface{}, error) {
    p := value.(Point)
    return map[string]interface{}{"lat": p.Lat, "lon": p.Lon}, nil
}

func (c *PointConverter) FromGraph(value interface{}, targetType reflect.Type) (interface{}, error) {
    m := value.(map[string]interface{})
    return Point{Lat: m["lat"].(float64), Lon: m["lon"].(float64)}, nil
}

orm.GetTypeRegistry().Register(reflect.TypeOf(Point{}), &PointConverter{})
```

---

### Pagination

#### NewPageable

```go
func NewPageable(page int, size int) Pageable
```

Creates a pageable for pagination.

**Parameters**:
- `page`: Page number (0-indexed)
- `size`: Page size

**Returns**: Pageable instance

**Example**:
```go
pageable := orm.NewPageable(0, 20)
```

#### Page[T]

```go
type Page[T any] struct {
    Content       []T
    TotalElements int64
    TotalPages    int
    Size          int
    Number        int
}
```

Represents a page of results.

**Example**:
```go
page, err := repo.Query().Paginate(pageable).FindPage()
fmt.Printf("Page %d/%d with %d items\n", 
    page.Number+1, page.TotalPages, len(page.Content))
```

---

## security Package

### SecurityContext

Manages security state for a user.

#### NewSecurityContext

```go
func NewSecurityContext(graph *falkordb.Graph, registry *orm.EntityRegistry, userID uuid.UUID) *SecurityContext
```

Creates a security context for a user.

**Parameters**:
- `graph`: FalkorDB graph connection
- `registry`: Entity registry
- `userID`: UUID of current user

**Returns**: SecurityContext instance

**Example**:
```go
ctx := security.NewSecurityContext(graph, registry, currentUserID)
```

#### HasPermission

```go
func (sc *SecurityContext) HasPermission(role string, action string) (bool, error)
```

Checks if user has permission.

**Parameters**:
- `role`: Required role
- `action`: Action to check (e.g., "read", "write")

**Returns**: Boolean permission status and error

#### GetUser

```go
func (sc *SecurityContext) GetUser() (*User, error)
```

Retrieves the current user entity.

**Returns**: User entity and error

---

### SecureRepository[T]

Security-wrapped repository with access control.

#### NewSecureRepository

```go
func NewSecureRepository[T any](
    repo *orm.Repository[T],
    context *SecurityContext,
    metadata *SecurityMetadata,
) *SecureRepository[T]
```

Creates a secure repository.

**Parameters**:
- `repo`: Underlying repository
- `context`: Security context
- `metadata`: Security metadata with permissions

**Returns**: SecureRepository instance

**Example**:
```go
metadata := security.NewSecurityMetadataBuilder().
    RequirePermission("user", "read").
    RestrictProperty("salary", "admin", "read").
    Build()

secureRepo := security.NewSecureRepository(repo, ctx, metadata)
```

#### Methods

SecureRepository implements the same interface as Repository[T]:
- `Save(entity T) error`
- `FindByID(id uuid.UUID) (T, error)`
- `FindAll() ([]T, error)`
- etc.

All methods enforce security policies automatically.

---

### RBACManager

Manages roles, privileges, and permissions.

#### NewRBACManager

```go
func NewRBACManager(graph *falkordb.Graph, registry *orm.EntityRegistry) *RBACManager
```

Creates an RBAC manager.

**Example**:
```go
rbacMgr := security.NewRBACManager(graph, registry)
```

#### CreateRole

```go
func (rm *RBACManager) CreateRole(name string, description string) (*Role, error)
```

Creates a new role.

**Parameters**:
- `name`: Role name
- `description`: Role description

**Returns**: Role entity and error

#### AssignRoleToUser

```go
func (rm *RBACManager) AssignRoleToUser(userID uuid.UUID, roleID uuid.UUID) error
```

Assigns a role to a user.

**Parameters**:
- `userID`: User UUID
- `roleID`: Role UUID

**Returns**: Error if assignment fails

#### GrantPrivilege

```go
func (rm *RBACManager) GrantPrivilege(roleID uuid.UUID, resource string, action string) error
```

Grants a privilege to a role.

**Parameters**:
- `roleID`: Role UUID
- `resource`: Resource name
- `action`: Action (e.g., "read", "write")

**Returns**: Error if grant fails

---

### SecurityMetadata

Defines security rules for entities.

#### NewSecurityMetadataBuilder

```go
func NewSecurityMetadataBuilder() *SecurityMetadataBuilder
```

Creates a security metadata builder.

**Example**:
```go
builder := security.NewSecurityMetadataBuilder()
```

#### RequirePermission

```go
func (b *SecurityMetadataBuilder) RequirePermission(role string, action string) *SecurityMetadataBuilder
```

Requires permission for entity access.

**Parameters**:
- `role`: Required role
- `action`: Required action

**Returns**: Self for chaining

#### RestrictProperty

```go
func (b *SecurityMetadataBuilder) RestrictProperty(
    property string,
    role string,
    action string,
) *SecurityMetadataBuilder
```

Restricts a property to specific role.

**Parameters**:
- `property`: Property name
- `role`: Required role
- `action`: Required action

**Returns**: Self for chaining

#### Build

```go
func (b *SecurityMetadataBuilder) Build() *SecurityMetadata
```

Builds the security metadata.

**Returns**: SecurityMetadata instance

**Example**:
```go
metadata := security.NewSecurityMetadataBuilder().
    RequirePermission("user", "read").
    RestrictProperty("salary", "admin", "read").
    RestrictProperty("ssn", "admin", "read").
    Build()
```

---

## Query Derivation Reference

### Method Name Patterns

Repositories support method-name-based query derivation:

**Pattern**: `{Action}By{Property}{Operator}[And{Property}{Operator}...]`

**Actions**:
- `FindBy`: Returns entities
- `CountBy`: Returns count
- `ExistsBy`: Returns boolean
- `DeleteBy`: Deletes entities

**Operators**:
- `Equal` (default if omitted)
- `NotEqual`
- `GreaterThan`, `GreaterThanOrEqual`
- `LessThan`, `LessThanOrEqual`
- `Between`
- `In`, `NotIn`
- `Like`, `NotLike`
- `Contains`, `NotContains`
- `StartsWith`, `EndsWith`
- `IsNull`, `IsNotNull`

**Examples**:
```go
// Find by exact match
users, err := repo.FindByName("Alice")

// Multiple conditions
users, err := repo.FindByNameAndAgeGreaterThan("Alice", 25)

// String operations
users, err := repo.FindByEmailEndsWith("@example.com")

// Null checks
users, err := repo.FindByBioIsNotNull()

// Counting
count, err := repo.CountByAgeGreaterThan(18)

// Existence check
exists, err := repo.ExistsByEmail("alice@example.com")

// Deletion
err := repo.DeleteByAgeIsNull()
```

---

## Error Handling

All methods that can fail return an error as the last return value. Check errors explicitly:

```go
user, err := repo.FindByID(id)
if err != nil {
    // Handle error
    return err
}
```

Common errors:
- Entity not found
- Connection errors
- Constraint violations
- Permission denied (security)

---

For architecture details, see [ARCHITECTURE.md](ARCHITECTURE.md).  
For usage examples, see [TUTORIAL.md](TUTORIAL.md).
