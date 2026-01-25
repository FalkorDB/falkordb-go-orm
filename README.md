# FalkorDB Go ORM

A powerful, type-safe Object-Relational Mapping library for [FalkorDB](https://falkordb.com) graph database in Go.

[![Go Version](https://img.shields.io/badge/Go-1.21+-00ADD8?style=flat&logo=go)](https://go.dev)
[![FalkorDB](https://img.shields.io/badge/FalkorDB-Compatible-green)](https://falkordb.com)
[![License](https://img.shields.io/badge/License-MIT-blue.svg)](LICENSE)

## Project Status

**Current Version**: 1.0.0-beta (Production Ready)

Fully implemented and tested with live FalkorDB:
- ✅ Core Entity Mapping
- ✅ Repository Pattern with CRUD operations
- ✅ Query Derivation (17+ operators)
- ✅ Relationships (lazy/eager loading)
- ✅ Sessions & Transactions
- ✅ RBAC Security
- ✅ Advanced Features (Indexes, Schema Validation)
- ✅ Comprehensive Documentation & Examples

## Features

- 🎯 **Type-Safe**: Built with Go generics for compile-time type safety
- 🏗️ **Entity Mapping**: Automatic Go struct ↔ Graph node/relationship mapping
- 📊 **Repository Pattern**: Generic `Repository[T]` with CRUD operations
- 🔍 **Query Derivation**: Parse method names into Cypher queries (17+ operators)
- 🔗 **Relationships**: First-class support with lazy/eager loading and cascading
- 💾 **Sessions**: Identity map, change tracking, and transaction management
- 🔐 **RBAC Security**: Fine-grained access control with property-level filtering
- 📈 **Index Management**: Create and manage RANGE, FULLTEXT, and VECTOR indexes
- ✅ **Schema Validation**: Validate and sync entity schemas automatically
- 🚀 **Production Ready**: ~4,500+ lines of tested code with live FalkorDB integration

## Quick Start

### 1. Define Your Entities

```go
package main

import (
    "github.com/FalkorDB/falkordb-go-orm/orm"
    "github.com/google/uuid"
)

type User struct {
    ID       uuid.UUID `falkor:"id,generated"`
    Name     string    `falkor:"name,indexed"`
    Email    string    `falkor:"email,unique,indexed"`
    Age      int       `falkor:"age"`
    Friends  orm.LazyList[*User] `falkor:"rel=FRIENDS_WITH,dir=out,lazy"`
}

func (u User) Labels() []string {
    return []string{"User"}
}
```

### 2. Connect and Create Repository

```go
import (
    "github.com/FalkorDB/falkordb-go/v2"
    "github.com/FalkorDB/falkordb-go-orm/orm"
)

// Connect to FalkorDB
client := falkordb.NewFalkorDB("localhost:6379", 
    falkordb.ClientOption{User: "username", Password: "password"})
graph := client.SelectGraph("myapp")

// Register entities
registry := orm.NewEntityRegistry()
registry.Register(&User{})

// Create repository
repo := orm.NewRepository[*User](graph, registry)
```

### 3. Perform CRUD Operations

```go
// Create
user := &User{
    Name:  "Alice",
    Email: "alice@example.com",
    Age:   30,
}
err := repo.Save(user)

// Read
found, err := repo.FindByID(user.ID)

// Query with derivation
users, err := repo.FindByAgeGreaterThan(25)

// Update
user.Age = 31
err = repo.Save(user)

// Delete
err = repo.DeleteByID(user.ID)
```

### 4. Work with Relationships

```go
alice := &User{Name: "Alice", Email: "alice@example.com"}
bob := &User{Name: "Bob", Email: "bob@example.com"}

repo.SaveAll([]*User{alice, bob})

// Create relationship
relMgr := orm.NewRelationshipManager(graph, registry)
err := relMgr.SaveRelationships(alice, "Friends", []interface{}{bob})

// Lazy load
friends := alice.Friends.Get() // Loads on first access

// Eager load
users, err := repo.Query().
    Where("age > $age", map[string]interface{}{"age": 18}).
    Fetch("Friends"). // Eager load
    FindAll()
```

### 5. Use Sessions for Transactions

```go
session := orm.NewSession(graph, registry)

// Automatic session management
err := orm.WithSession(session, func(s *orm.Session) error {
    user := &User{Name: "Charlie", Email: "charlie@example.com"}
    s.Save(user) // Tracked in session
    
    user.Age = 25
    s.Save(user) // Dirty tracking
    
    return s.Commit() // Commits all changes
})
```

## Core Concepts

### Entity Mapping

Entities are Go structs annotated with `falkor` tags:

```go
type Product struct {
    ID       uuid.UUID `falkor:"id,generated"`
    SKU      string    `falkor:"sku,unique,indexed"`
    Name     string    `falkor:"name,indexed"`
    Price    float64   `falkor:"price"`
    Tags     []string  `falkor:"tags"`
    Created  time.Time `falkor:"created"`
}

func (p Product) Labels() []string {
    return []string{"Product", "Item"}
}
```

**Tag Options**:
- `generated`: Auto-generate UUID on save
- `unique`: Enforce uniqueness constraint
- `indexed`: Create database index
- `rel=TYPE`: Relationship type
- `dir=out|in|both`: Relationship direction
- `lazy`: Lazy load (default for relationships)
- `cascade`: Cascade operations

### Query Derivation

Repositories support method-name-based queries:

```go
// Automatically parses method name into Cypher query
users, err := repo.FindByNameAndAgeGreaterThan("Alice", 25)
count, err := repo.CountByEmailLike("%@example.com")
exists, err := repo.ExistsByID(userID)
err := repo.DeleteByAgeIsNull()
```

**Supported Operators**:
`Equal`, `NotEqual`, `GreaterThan`, `GreaterThanOrEqual`, `LessThan`, `LessThanOrEqual`, `Between`, `In`, `NotIn`, `Like`, `NotLike`, `Contains`, `NotContains`, `StartsWith`, `EndsWith`, `IsNull`, `IsNotNull`

### Query Builder

For complex queries, use the fluent query builder:

```go
page, err := repo.Query().
    Where("age > $minAge AND city = $city", map[string]interface{}{
        "minAge": 21,
        "city":   "New York",
    }).
    OrderBy("name ASC").
    Paginate(orm.NewPageable(0, 10)).
    FindPage()
```

### Relationships

Define relationships using struct fields:

```go
type Post struct {
    ID       uuid.UUID `falkor:"id,generated"`
    Title    string    `falkor:"title"`
    Author   orm.LazySingle[*User] `falkor:"rel=AUTHORED,dir=in,lazy"`
    Tags     orm.LazyList[*Tag]    `falkor:"rel=TAGGED,dir=out,cascade"`
}
```

Load relationships:

```go
// Lazy loading
author := post.Author.Get() // Loads from DB on first access

// Eager loading
posts, err := repo.Query().Fetch("Author", "Tags").FindAll()
```

### Security & RBAC

Secure repositories enforce access control:

```go
import "github.com/FalkorDB/falkordb-go-orm/security"

// Define security metadata
metadata := security.NewSecurityMetadataBuilder().
    RequirePermission("user", "read").
    RestrictProperty("salary", "admin", "read").
    RestrictProperty("ssn", "admin", "read").
    Build()

// Create secure repository
ctx := security.NewSecurityContext(graph, registry, userID)
secureRepo := security.NewSecureRepository(repo, ctx, metadata)

// Queries automatically filter based on permissions
employees, err := secureRepo.FindAll() // SSN/salary filtered for non-admins
```

## Advanced Features

### Index Management

```go
indexMgr := orm.NewIndexManager(graph)

// Create indexes for all indexed properties
err := indexMgr.CreateIndexes(&Product{}, orm.IndexTypeRange)

// List existing indexes
indexes, err := indexMgr.ListIndexes("Product")

// Ensure indexes (idempotent)
err := indexMgr.EnsureIndexes(&Product{}, orm.IndexTypeRange)
```

### Schema Validation

```go
schemaMgr := orm.NewSchemaManager(registry, graph)

// Validate single entity
result, err := schemaMgr.ValidateSchema(&Product{})
if !result.IsValid {
    for _, err := range result.Errors {
        log.Println(err)
    }
}

// Sync all registered entities (create missing indexes)
err = schemaMgr.SyncSchema(orm.IndexTypeRange)
```

### Pagination

```go
// Page-based pagination
pageable := orm.NewPageable(0, 20) // page 0, size 20
page, err := repo.Query().
    OrderBy("created DESC").
    Paginate(pageable).
    FindPage()

fmt.Printf("Total: %d, Pages: %d\n", page.TotalElements, page.TotalPages)
for _, user := range page.Content {
    fmt.Println(user.Name)
}
```

## Examples

Comprehensive examples are available in the `examples/` directory:

- **basic**: Simple CRUD operations ([examples/basic/main.go](examples/basic/main.go))
- **repository**: Repository pattern and pagination ([examples/repository/main.go](examples/repository/main.go))
- **query_derivation**: Method-name queries with all operators ([examples/query_derivation/main.go](examples/query_derivation/main.go))
- **relationships**: Lazy/eager loading, cascading ([examples/relationships/main.go](examples/relationships/main.go))
- **session**: Transactions, identity map, change tracking ([examples/session/main.go](examples/session/main.go))
- **security**: RBAC with property-level access control ([examples/security/main.go](examples/security/main.go))
- **indexes**: Index management and schema validation ([examples/indexes/main.go](examples/indexes/main.go))

Run any example:
```bash
cd examples/basic
go run main.go
```

## Documentation

- [Architecture Guide](ARCHITECTURE.md) - Deep dive into design and internals
- [Tutorial](TUTORIAL.md) - Build a complete application step-by-step
- [API Reference](API.md) - Complete API documentation
- [Migration Guide](MIGRATION.md) - Migrate from other ORMs or raw client

## Performance

- **Type-safe**: Zero reflection overhead in hot paths with generics
- **Connection pooling**: Uses falkordb-go client connection management
- **Lazy loading**: Relationships loaded on-demand to minimize queries
- **Identity map**: Session-level caching prevents duplicate queries
- **Indexed queries**: Full support for database indexes

## Contributing

Contributions welcome! See [CONTRIBUTING.md](CONTRIBUTING.md) for guidelines.

## License

MIT License - see [LICENSE](LICENSE) file for details.

## Acknowledgments

Inspired by [falkordb-py-orm](https://github.com/FalkorDB/falkordb-py-orm), adapted for Go idioms and type safety.

Built with ❤️ by the FalkorDB community.

---

**Ready to build graph applications in Go?** Check out the [Tutorial](TUTORIAL.md) to get started!
