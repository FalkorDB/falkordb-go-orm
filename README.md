# falkordb-go-orm

Type-safe, idiomatic Go client for working with [FalkorDB](https://falkordb.com) using an ORM-style API.

This project sits on top of the official `falkordb-go` client and adds:

- **Object mapping**: map Go structs to FalkorDB nodes/relationships using `falkor:"..."` tags.
- **Query ergonomics**: a fluent **query builder** and Spring-Data-like **query derivation** (method-name parsing).
- **RBAC protection**: optional `security` package that can enforce entity/action permissions and apply **data-level filtering** (row-level rules) and **property-level filtering**.

[![Go Version](https://img.shields.io/badge/Go-1.21+-00ADD8?style=flat&logo=go)](https://go.dev)
[![FalkorDB](https://img.shields.io/badge/FalkorDB-Compatible-green)](https://falkordb.com)
[![License](https://img.shields.io/badge/License-MIT-blue.svg)](LICENSE)

## What you get

### Object mapping (Go structs ↔ graph)
Define entities with `falkor` tags and (optionally) a `Labels() []string` method:

```go
package main

type User struct {
	ID    *int64  `falkor:"id,generated"`
	Name  string  `falkor:"name,indexed"`
	Email string  `falkor:"email,unique,indexed"`
	Age   int     `falkor:"age"`
}

func (*User) Labels() []string { return []string{"User"} }
```

### Repository API + query builder
Work through a generic repository and build Cypher safely with parameters:

```go
// Connect using falkordb-go (see examples/ for full setup)
graph := client.SelectGraph("myapp")

// Register entity metadata once
orm.Register(&User{})

repo := orm.NewRepository[User](graph)

// Fluent query builder
users, err := repo.Query().
	Where("n.age > $minAge", map[string]any{"minAge": 21}).
	OrderBy("n.name ASC").
	Limit(50).
	FindAll()
```

### Query derivation (method-name parsing)
For common cases, derive Cypher from method names (17+ operators):

```go
// FindByNameAndAgeGreaterThan -> MATCH ... WHERE ...
users, err := repo.ExecuteDerivedQuery("FindByNameAndAgeGreaterThan", "Alice", 25)

count, err := repo.ExecuteDerivedQuery("CountByEmailEndsWith", "@example.com")
exists, err := repo.ExecuteDerivedQuery("ExistsByEmail", "alice@example.com")
```

## RBAC + data-level filtering
The optional `security` package wraps repositories to enforce permissions and apply filtering:

- **Entity/action permissions**: allow/deny operations like READ/WRITE/CREATE/DELETE.
- **Property-level filtering**: automatically zero-out fields the caller is not allowed to read/write.
- **Data-level filtering (row-level rules)**: express tenant/user scoping as a Cypher predicate (e.g. `n.tenant_id = $tenantID`).

Minimal setup (conceptual):

```go
// 1) Register security metadata for an entity type
meta := security.NewSecurityMetadata().
	WithReadRoles("user", "admin").
	DenyReadProperty("ssn", "*").
	WithRowLevelSecurity("n.tenant_id = $tenantID", "user")

security.RegisterSecurityMetadata(reflect.TypeOf(User{}), meta)

// 2) Wrap repository with a user-bound SecurityContext
ctx, _ := security.NewSecurityContext(graph, currentUser)
secureRepo := security.NewSecureRepository[User](graph, ctx)

// 3) Use secureRepo instead of repo
users, err := secureRepo.FindAll()
```

Note: property-level filtering is enforced by `SecureRepository` today; row-level rules are represented in metadata so you can apply consistent data-level predicates in your queries.

## Documentation

- **API specifications**: `API.md`
- Tutorial / guided walkthrough: `TUTORIAL.md`
- Architecture / internals: `ARCHITECTURE.md`
- Migration notes: `MIGRATION.md`
- Contributing guide: `CONTRIBUTING.md`
- Additional materials: `docs/`

## Examples

Examples live under `examples/`.

## License

MIT — see `LICENSE`.
