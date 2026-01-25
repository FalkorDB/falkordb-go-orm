# FalkorDB Go ORM Architecture

This document provides a deep dive into the architecture, design decisions, and implementation details of FalkorDB Go ORM.

## Table of Contents

- [Overview](#overview)
- [Core Architecture](#core-architecture)
- [Component Details](#component-details)
- [Design Decisions](#design-decisions)
- [Performance Considerations](#performance-considerations)
- [Extension Points](#extension-points)

## Overview

FalkorDB Go ORM is designed as a type-safe, high-performance Object-Graph Mapping library for Go. It leverages Go 1.21+ generics to provide compile-time type safety while maintaining flexibility and extensibility.

### Design Principles

1. **Type Safety**: Use Go generics for compile-time guarantees
2. **Idiomatic Go**: Follow Go conventions and patterns
3. **Performance**: Minimize reflection, optimize hot paths
4. **Composability**: Small, focused components that compose well
5. **Extensibility**: Allow customization at key extension points

### Architecture Diagram

```
┌─────────────────────────────────────────────────────────────────┐
│                        Application Layer                         │
│                     (User's Go Application)                      │
└────────────────────────────┬────────────────────────────────────┘
                             │
┌────────────────────────────┴────────────────────────────────────┐
│                         ORM Layer                                │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐         │
│  │ Repository[T]│  │   Session    │  │  Security    │         │
│  │  (CRUD API)  │  │ (Txn, Cache) │  │  (RBAC)      │         │
│  └──────┬───────┘  └──────┬───────┘  └──────┬───────┘         │
│         │                  │                  │                  │
│  ┌──────┴──────────────────┴──────────────────┴───────┐         │
│  │         Query Builder & Parser                      │         │
│  │  (Cypher Generation, Method Derivation)            │         │
│  └──────────────────────┬──────────────────────────────┘         │
│                         │                                         │
│  ┌──────────────────────┴──────────────────────────────┐         │
│  │           Entity Registry & Mapper                   │         │
│  │  (Metadata, Type Conversion, Struct ↔ Cypher)      │         │
│  └──────────────────────┬──────────────────────────────┘         │
└─────────────────────────┴─────────────────────────────────────────┘
                          │
┌─────────────────────────┴─────────────────────────────────────────┐
│                    falkordb-go Client                              │
│              (Connection Pool, Protocol, Redis)                    │
└────────────────────────────────────────────────────────────────────┘
```

## Core Architecture

### 1. Entity Metadata System

**File**: `orm/entity.go`

The foundation of the ORM is the entity metadata system, which provides runtime information about entity structures.

```go
type EntityMetadata struct {
    Type           reflect.Type
    Labels         []string
    IDField        *PropertyMetadata
    Properties     []*PropertyMetadata
    Relationships  []*RelationshipMetadata
}
```

**Key Components**:

- **EntityRegistry**: Thread-safe cache of entity metadata using `sync.Map`
- **PropertyMetadata**: Field-level metadata extracted from struct tags
- **Struct Tag Parsing**: Parses `falkor:"name,options"` tags at registration time

**Design Choice**: Metadata is computed once at registration and cached, avoiding repeated reflection overhead.

### 2. Type Conversion System

**File**: `orm/types.go`

Handles bidirectional conversion between Go types and FalkorDB (Cypher) types.

```go
type TypeConverter interface {
    ToGraph(value interface{}) (interface{}, error)
    FromGraph(value interface{}, targetType reflect.Type) (interface{}, error)
}
```

**Built-in Converters**:
- Primitives (string, int, float, bool) → Direct mapping
- `time.Time` → Unix timestamp (int64)
- `uuid.UUID` → String representation
- Structs → JSON encoding
- Slices/Maps → Graph arrays/maps

**TypeRegistry**: Global registry supporting custom converter registration via `Register()` method.

**Design Choice**: Use interface-based converters for extensibility while providing sensible defaults.

### 3. Mapper

**File**: `orm/mapper.go`

Converts between Go structs and Cypher queries/results.

**Core Functions**:
- `MapToCreate()`: Generate CREATE statement from entity
- `MapToNode()`: Convert struct to Cypher node properties
- `MapFromResult()`: Populate struct from query result
- `UpdateEntityID()`: Set generated ID after creation

**Process Flow**:
```
Go Struct → Extract Properties → Apply Type Converters → Cypher Map
Cypher Result → Parse Values → Apply Type Converters → Go Struct
```

**Design Choice**: Separate mapping logic from business logic for testability and reusability.

### 4. Repository Pattern

**File**: `orm/repository.go`

Type-safe repository using Go generics for CRUD operations.

```go
type Repository[T any] struct {
    graph    *falkordb.Graph
    registry *EntityRegistry
}
```

**CRUD Operations**:
- `Save(entity T)`: CREATE or MERGE based on ID presence
- `FindByID(id uuid.UUID)`: MATCH by ID
- `FindAll()`: MATCH all entities
- `Delete(entity T)`: DELETE by ID
- `Count()`: COUNT entities

**Method Naming**: Follows Spring Data conventions (FindBy*, DeleteBy*, CountBy*, ExistsBy*).

**Design Choice**: Use generics for type safety and eliminate casting, improving developer experience.

### 5. Query Builder

**File**: `orm/query_builder.go`

Fluent API for building complex Cypher queries.

```go
type QueryBuilderChain[T any] struct {
    repo      *Repository[T]
    wherePart string
    params    map[string]interface{}
    orderBy   string
    limit     int
    skip      int
    fetch     []string
}
```

**Fluent Methods**:
- `Where(condition, params)`: Add WHERE clause
- `OrderBy(order)`: Add ORDER BY
- `Limit(n)`: Add LIMIT
- `Skip(n)`: Add SKIP (for pagination)
- `Fetch(relationships...)`: Eager load relationships
- `FindAll()`, `FindPage()`, `Count()`: Terminal operations

**Design Choice**: Method chaining provides readable, composable queries while building Cypher incrementally.

### 6. Query Derivation

**File**: `orm/query_parser.go`

Parses method names into Cypher queries using a mini-parser.

**Supported Patterns**:
- `FindBy{Property}{Operator}`
- `CountBy{Property}{Operator}`
- `ExistsBy{Property}{Operator}`
- `DeleteBy{Property}{Operator}`

**Operators** (17 total):
- Comparison: Equal, NotEqual, GreaterThan, LessThan, etc.
- String: Like, Contains, StartsWith, EndsWith
- Collection: In, NotIn, Between
- Null: IsNull, IsNotNull

**Parsing Algorithm**:
```
1. Parse method prefix (Find, Count, Exists, Delete)
2. Extract property names and operators
3. Build WHERE clause with proper Cypher syntax
4. Generate complete query based on prefix
```

**Caching**: Parsed queries cached in `sync.Map` to avoid repeated parsing.

**Design Choice**: Inspired by Spring Data but adapted for Cypher syntax and Go conventions.

### 7. Relationship Management

**File**: `orm/relationships.go`

Handles entity relationships with lazy/eager loading support.

**Core Types**:
```go
type LazyList[T any] struct {
    loader func() []T
    loaded bool
    items  []T
}

type LazySingle[T any] struct {
    loader func() []T
    loaded bool
    item   T
}
```

**RelationshipManager**:
- `SaveRelationships()`: CREATE relationships
- `DeleteRelationships()`: DELETE relationships
- `LoadRelationships()`: Load related entities

**Loading Strategies**:
- **Lazy**: Load on first `Get()` call
- **Eager**: Preload via `Fetch()` in query

**Design Choice**: Lazy loading by default to avoid N+1 problems, with explicit eager loading when needed.

### 8. Session & Transactions

**File**: `orm/session.go`

Provides transaction support and identity map pattern.

```go
type Session struct {
    graph      *falkordb.Graph
    registry   *EntityRegistry
    identityMap map[string]interface{}  // "Type:ID" → entity
    newEntities    map[string]interface{}
    dirtyEntities  map[string]interface{}
    deletedEntities map[string]interface{}
    transaction    *falkordb.Transaction
}
```

**Identity Map**: Ensures single instance per entity within session using "Type:ID" keys.

**Change Tracking**:
- **New**: Entities without ID
- **Dirty**: Modified entities
- **Deleted**: Marked for deletion

**Transaction Flow**:
```
BeginTransaction() → Track Changes → Commit() / Rollback()
```

**Design Choice**: Implement Unit of Work pattern for consistent transaction boundaries and caching.

### 9. Security & RBAC

**Files**: `security/*.go`

Fine-grained access control with property-level restrictions.

**Components**:

**SecurityContext**:
- Manages current user permissions
- Caches role/privilege lookups
- Validates access at runtime

**SecurityMetadata**:
- Entity-level permission requirements
- Property-level restrictions
- Fluent builder API

**SecureRepository[T]**:
- Wraps Repository[T]
- Filters results based on permissions
- Removes restricted properties via reflection

**Design Choice**: Security as a wrapper allows optional use while providing comprehensive RBAC when needed.

### 10. Index & Schema Management

**Files**: `orm/indexes.go`, `orm/schema.go`

**IndexManager**:
- Creates indexes for tagged properties
- Supports RANGE, FULLTEXT, VECTOR types
- Idempotent operations via `EnsureIndexes()`

**SchemaManager**:
- Validates entity definitions
- Syncs indexes across registered entities
- Reports validation errors

**Design Choice**: Separate index management from core operations for flexibility in deployment strategies.

## Design Decisions

### Why Generics?

**Decision**: Use Go 1.21+ generics for Repository[T]

**Rationale**:
- Type safety at compile time
- Eliminates type assertions
- Better IDE support
- Cleaner API

**Trade-off**: Requires Go 1.21+, but provides significantly better developer experience.

### Why Struct Tags?

**Decision**: Use `falkor:"..."` struct tags for metadata

**Rationale**:
- Idiomatic Go (similar to `json:`, `db:` tags)
- Declarative and readable
- Compile-time checked by linters
- Single source of truth

**Alternative Considered**: Builder pattern or decorators (rejected for verbosity).

### Why Labels() Method?

**Decision**: Require `Labels() []string` method instead of struct tag

**Rationale**:
- Dynamic label generation possible
- Multiple labels support
- Interface-based approach
- Consistent with entity definition

**Alternative Considered**: `@node` decorator like Python ORM (not possible in Go).

### Why Identity Map?

**Decision**: Implement identity map in Session

**Rationale**:
- Prevents duplicate entity instances
- Enables change tracking
- Improves performance (caching)
- Essential for transaction correctness

**Trade-off**: Memory overhead within session scope (acceptable for typical use cases).

### Why Lazy Loading?

**Decision**: Default to lazy loading for relationships

**Rationale**:
- Avoids N+1 query problem by default
- Explicit eager loading when needed
- Reduces initial query complexity
- Better performance for common cases

**Trade-off**: Can lead to many queries if not carefully managed (mitigated by Fetch()).

## Performance Considerations

### Hot Path Optimizations

1. **Metadata Caching**: Entity metadata computed once at registration
2. **Query Caching**: Parsed derived queries cached in sync.Map
3. **Type Conversion**: Direct type switches for common types
4. **Generics**: Avoid reflection in Repository methods

### Reflection Usage

**Where Used**:
- Entity registration (one-time)
- Struct tag parsing (one-time)
- Mapping results to structs (per query)
- Security property filtering (per query)

**Minimization Strategy**:
- Cache all metadata
- Use type switches before reflect.Value
- Batch reflection operations

### Concurrency

**Thread-Safe Components**:
- EntityRegistry (sync.Map)
- TypeRegistry (sync.Map)
- QueryParser cache (sync.Map)
- Session (sync.RWMutex)

**Non-Thread-Safe** (by design):
- Repository[T] (stateless, safe to share)
- Entities (user responsibility)

## Extension Points

### 1. Custom Type Converters

```go
type CustomConverter struct{}

func (c *CustomConverter) ToGraph(value interface{}) (interface{}, error) {
    // Custom conversion logic
}

func (c *CustomConverter) FromGraph(value interface{}, targetType reflect.Type) (interface{}, error) {
    // Custom conversion logic
}

// Register
orm.GetTypeRegistry().Register(reflect.TypeOf(MyType{}), &CustomConverter{})
```

### 2. Custom Query Logic

Extend Repository[T] via composition:

```go
type UserRepository struct {
    *orm.Repository[*User]
}

func (r *UserRepository) FindActiveUsers() ([]*User, error) {
    return r.Query().
        Where("active = $active", map[string]interface{}{"active": true}).
        FindAll()
}
```

### 3. Custom Security Rules

```go
metadata := security.NewSecurityMetadataBuilder().
    RequirePermission("custom_role", "read").
    AddCustomValidator(func(ctx *security.SecurityContext, entity interface{}) bool {
        // Custom validation logic
        return true
    }).
    Build()
```

### 4. Middleware Pattern

Wrap Repository methods for cross-cutting concerns:

```go
type LoggingRepository[T any] struct {
    repo *orm.Repository[T]
}

func (r *LoggingRepository[T]) Save(entity T) error {
    log.Printf("Saving entity: %+v", entity)
    err := r.repo.Save(entity)
    log.Printf("Save result: %v", err)
    return err
}
```

## Implementation Details

### ID Generation

**Strategy**: Use `uuid.UUID` for entity IDs

**Rationale**:
- Universally unique
- No coordination needed
- Suitable for distributed systems
- Standard library support

**Alternative**: Could use FalkorDB internal IDs, but UUIDs provide better application-level control.

### Cypher Generation

**Approach**: String building with parameterized queries

**Security**: All user values passed as parameters (prevents injection)

**Example**:
```go
// Instead of: "MATCH (n:User {name: 'Alice'})"
// Generate: "MATCH (n:User {name: $name})"
// With params: {"name": "Alice"}
```

### Error Handling

**Strategy**: Wrap errors with context using `fmt.Errorf("%w")`

**Benefits**:
- Error chains preserve original errors
- Context helps debugging
- Supports `errors.Is()` and `errors.As()`

### Testing Strategy

**Levels**:
1. **Unit Tests**: Pure logic (parsers, converters)
2. **Integration Tests**: With live FalkorDB
3. **Example Tests**: Full scenarios in examples/

**Test Data**: Each test cleans up after itself (MATCH (n) DETACH DELETE n).

## Future Considerations

### Potential Enhancements

1. **Query Result Streaming**: For large result sets
2. **Bulk Operations**: Optimized batch inserts
3. **Migration Tools**: Schema migration support
4. **Graph Algorithms**: Native support for graph algorithms
5. **Connection Pooling**: Enhanced pool management
6. **Distributed Tracing**: OpenTelemetry integration

### Backward Compatibility

**Commitment**: Semantic versioning (SemVer)
- Major version for breaking changes
- Minor version for new features
- Patch version for bug fixes

## Conclusion

FalkorDB Go ORM is architected for type safety, performance, and extensibility. The use of Go generics provides compile-time guarantees while maintaining the flexibility needed for complex graph database operations. The modular design allows users to adopt features incrementally, from simple CRUD to advanced security and transaction management.

For specific API details, see [API.md](API.md).  
For usage examples, see [TUTORIAL.md](TUTORIAL.md).
