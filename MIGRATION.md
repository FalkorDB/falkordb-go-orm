# Migration Guide

This guide helps you migrate to FalkorDB Go ORM from various sources.

## Table of Contents

- [From Raw falkordb-go Client](#from-raw-falkordb-go-client)
- [From falkordb-py-orm (Python)](#from-falkordb-py-orm-python)
- [From Other Go ORMs](#from-other-go-orms)
- [Common Migration Patterns](#common-migration-patterns)

## From Raw falkordb-go Client

If you're currently using the raw `falkordb-go` client, migrating to the ORM provides type safety, reduced boilerplate, and automatic query generation.

### Before: Raw Client

```go
import "github.com/FalkorDB/falkordb-go/v2"

// Define struct (no metadata)
type User struct {
    ID    string
    Name  string
    Email string
    Age   int
}

// Connect
client := falkordb.NewFalkorDB("localhost:6379", falkordb.ClientOption{})
graph := client.SelectGraph("myapp")

// Create entity (manual Cypher)
query := "CREATE (u:User {id: $id, name: $name, email: $email, age: $age}) RETURN u"
params := map[string]interface{}{
    "id":    uuid.New().String(),
    "name":  "Alice",
    "email": "alice@example.com",
    "age":   30,
}
result, err := graph.Query(query, params, nil)

// Find by ID (manual Cypher)
query = "MATCH (u:User {id: $id}) RETURN u"
params = map[string]interface{}{"id": userID}
result, err = graph.Query(query, params, nil)

// Parse results manually
if len(result.ResultSet) > 0 {
    node := result.ResultSet[0][0].(*falkordb.Node)
    user := &User{
        ID:    node.Properties["id"].(string),
        Name:  node.Properties["name"].(string),
        Email: node.Properties["email"].(string),
        Age:   int(node.Properties["age"].(int64)),
    }
}
```

### After: With ORM

```go
import (
    "github.com/FalkorDB/falkordb-go/v2"
    "github.com/FalkorDB/falkordb-go-orm/orm"
    "github.com/google/uuid"
)

// Define entity with metadata
type User struct {
    ID    uuid.UUID `falkor:"id,generated"`
    Name  string    `falkor:"name"`
    Email string    `falkor:"email,unique,indexed"`
    Age   int       `falkor:"age"`
}

func (u User) Labels() []string {
    return []string{"User"}
}

// Connect
client := falkordb.NewFalkorDB("localhost:6379", falkordb.ClientOption{})
graph := client.SelectGraph("myapp")

// Setup ORM
registry := orm.NewEntityRegistry()
registry.Register(&User{})
repo := orm.NewRepository[*User](graph, registry)

// Create entity (automatic Cypher generation)
user := &User{
    Name:  "Alice",
    Email: "alice@example.com",
    Age:   30,
}
err := repo.Save(user)
// user.ID is automatically populated

// Find by ID (type-safe, no manual parsing)
found, err := repo.FindByID(user.ID)
```

### Key Benefits

| Aspect | Raw Client | ORM |
|--------|-----------|-----|
| **Type Safety** | Manual casting | Compile-time checks with generics |
| **Cypher Queries** | Write manually | Auto-generated |
| **Result Parsing** | Manual mapping | Automatic |
| **Code Lines** | ~20-30 for CRUD | ~5-10 for CRUD |
| **Query Errors** | Runtime | Compile-time (for typed methods) |
| **Relationships** | Manual MATCH | Automatic lazy/eager loading |

### Migration Steps

1. **Add struct tags** to your existing structs:
   ```go
   // Before
   type User struct {
       ID   string
       Name string
   }
   
   // After
   type User struct {
       ID   uuid.UUID `falkor:"id,generated"`
       Name string    `falkor:"name"`
   }
   
   func (u User) Labels() []string {
       return []string{"User"}
   }
   ```

2. **Register entities**:
   ```go
   registry := orm.NewEntityRegistry()
   registry.Register(&User{})
   ```

3. **Create repositories**:
   ```go
   repo := orm.NewRepository[*User](graph, registry)
   ```

4. **Replace manual queries** with repository methods:
   ```go
   // Before
   result, err := graph.Query("MATCH (u:User {email: $email}) RETURN u", params, nil)
   
   // After
   users, err := repo.FindByEmail("alice@example.com")
   ```

### Complex Query Migration

**Before** (manual Cypher):
```go
query := `
MATCH (u:User)
WHERE u.age > $minAge AND u.city = $city
RETURN u
ORDER BY u.name ASC
LIMIT 10
`
params := map[string]interface{}{
    "minAge": 21,
    "city":   "New York",
}
result, err := graph.Query(query, params, nil)
// Manual parsing...
```

**After** (query builder):
```go
users, err := repo.Query().
    Where("age > $minAge AND city = $city", map[string]interface{}{
        "minAge": 21,
        "city":   "New York",
    }).
    OrderBy("name ASC").
    Limit(10).
    FindAll()
```

---

## From falkordb-py-orm (Python)

Migrating from the Python ORM to Go ORM is straightforward due to similar design philosophy.

### Entity Definition

**Python**:
```python
from falkordb_py_orm import node, property

@node
class User:
    id = property(generated_id=True)
    name = property(indexed=True)
    email = property(unique=True, indexed=True)
    age = property()
```

**Go**:
```go
type User struct {
    ID    uuid.UUID `falkor:"id,generated"`
    Name  string    `falkor:"name,indexed"`
    Email string    `falkor:"email,unique,indexed"`
    Age   int       `falkor:"age"`
}

func (u User) Labels() []string {
    return []string{"User"}
}
```

### Repository Usage

**Python**:
```python
from falkordb_py_orm import FalkorDBRepository

repo = FalkorDBRepository[User](graph)

# Save
user = User(name="Alice", email="alice@example.com", age=30)
repo.save(user)

# Find
user = repo.find_by_id(user_id)
users = repo.find_all()

# Query
users = repo.find_by_age_greater_than(25)
```

**Go**:
```go
import "github.com/FalkorDB/falkordb-go-orm/orm"

repo := orm.NewRepository[*User](graph, registry)

// Save
user := &User{Name: "Alice", Email: "alice@example.com", Age: 30}
repo.Save(user)

// Find
user, err := repo.FindByID(userID)
users, err := repo.FindAll()

// Query
users, err := repo.FindByAgeGreaterThan(25)
```

### Key Differences

| Feature | Python ORM | Go ORM |
|---------|-----------|--------|
| **Decorators** | `@node` | Struct tags + `Labels()` method |
| **Type Hints** | Runtime validation | Compile-time with generics |
| **Property Definition** | `property()` function | Struct tags |
| **Generated IDs** | `generated_id=True` | `falkor:"id,generated"` |
| **Relationships** | Property with `@relationship` | Struct field with `falkor:"rel=..."` |
| **Error Handling** | Exceptions | Explicit error returns |

### Relationships

**Python**:
```python
@node
class Post:
    id = property(generated_id=True)
    title = property()
    author = relationship("User", direction="in")
    tags = relationship("Tag", direction="out")
```

**Go**:
```go
type Post struct {
    ID     uuid.UUID             `falkor:"id,generated"`
    Title  string                `falkor:"title"`
    Author orm.LazySingle[*User] `falkor:"rel=AUTHORED,dir=in,lazy"`
    Tags   orm.LazyList[*Tag]    `falkor:"rel=TAGGED,dir=out,lazy"`
}

func (p Post) Labels() []string {
    return []string{"Post"}
}
```

### Sessions & Transactions

**Python**:
```python
with repo.session() as session:
    user = User(name="Alice")
    session.save(user)
    
    user.age = 30
    session.save(user)
    
    session.commit()
```

**Go**:
```go
session := orm.NewSession(graph, registry)

err := orm.WithSession(session, func(s *orm.Session) error {
    user := &User{Name: "Alice"}
    s.Save(user)
    
    user.Age = 30
    s.Save(user)
    
    return s.Commit()
})
```

---

## From Other Go ORMs

### From GORM (SQL ORM)

If you're familiar with GORM, the concepts are similar but adapted for graph databases.

**GORM** (SQL):
```go
type User struct {
    gorm.Model
    Name  string
    Email string `gorm:"uniqueIndex"`
}

db.Create(&user)
db.First(&user, id)
db.Where("age > ?", 18).Find(&users)
```

**FalkorDB ORM** (Graph):
```go
type User struct {
    ID    uuid.UUID `falkor:"id,generated"`
    Name  string    `falkor:"name"`
    Email string    `falkor:"email,unique,indexed"`
}

func (u User) Labels() []string {
    return []string{"User"}
}

repo.Save(&user)
repo.FindByID(id)
repo.Query().Where("age > $age", map[string]interface{}{"age": 18}).FindAll()
```

**Key Differences**:
- **Relationships**: Graph relationships are first-class, not foreign keys
- **Queries**: Cypher-based, not SQL
- **Indexes**: Graph indexes (RANGE, FULLTEXT, VECTOR)
- **Traversal**: Native graph traversal support

### From Ent (Graph ORM)

**Ent**:
```go
// Schema definition in separate files
client.User.
    Create().
    SetName("Alice").
    SetEmail("alice@example.com").
    Save(ctx)

user, err := client.User.Get(ctx, id)
```

**FalkorDB ORM**:
```go
// Schema in struct tags
user := &User{Name: "Alice", Email: "alice@example.com"}
repo.Save(user)

user, err := repo.FindByID(id)
```

**Key Differences**:
- **Schema Definition**: Struct tags vs. separate schema files
- **Code Generation**: FalkorDB ORM uses generics, no codegen needed
- **Backend**: FalkorDB (graph) vs. SQL databases

---

## Common Migration Patterns

### Pattern 1: Batch Operations

**Before** (raw client):
```go
for _, user := range users {
    query := "CREATE (u:User {name: $name, email: $email})"
    graph.Query(query, map[string]interface{}{
        "name":  user.Name,
        "email": user.Email,
    }, nil)
}
```

**After** (ORM):
```go
err := repo.SaveAll(users)
```

### Pattern 2: Pagination

**Before**:
```go
query := "MATCH (u:User) RETURN u SKIP $skip LIMIT $limit"
params := map[string]interface{}{
    "skip":  page * pageSize,
    "limit": pageSize,
}
result, err := graph.Query(query, params, nil)
```

**After**:
```go
pageable := orm.NewPageable(page, pageSize)
pageResult, err := repo.Query().Paginate(pageable).FindPage()
```

### Pattern 3: Relationships

**Before**:
```go
// Create relationship
query := "MATCH (u:User {id: $userId}), (f:User {id: $friendId}) CREATE (u)-[:FRIENDS]->(f)"
params := map[string]interface{}{
    "userId":   userID,
    "friendId": friendID,
}
graph.Query(query, params, nil)

// Load relationships
query = "MATCH (u:User {id: $userId})-[:FRIENDS]->(f:User) RETURN f"
params = map[string]interface{}{"userId": userID}
result, err := graph.Query(query, params, nil)
```

**After**:
```go
// Create relationship
relMgr := orm.NewRelationshipManager(graph, registry)
err := relMgr.SaveRelationships(user, "Friends", []interface{}{friend})

// Load relationships (lazy)
friends := user.Friends.Get()

// Or eager load
users, err := repo.Query().Fetch("Friends").FindAll()
```

### Pattern 4: Transactions

**Before**:
```go
tx := graph.BeginTransaction()

// Manual operations...
query1 := "CREATE (u:User {name: $name})"
tx.Query(query1, params1, nil)

query2 := "CREATE (p:Post {title: $title})"
tx.Query(query2, params2, nil)

tx.Commit()
```

**After**:
```go
session := orm.NewSession(graph, registry)

err := orm.WithSession(session, func(s *orm.Session) error {
    s.Save(&user)
    s.Save(&post)
    return s.Commit()
})
```

### Pattern 5: Dynamic Queries

**Before**:
```go
conditions := []string{}
params := map[string]interface{}{}

if nameFilter != "" {
    conditions = append(conditions, "u.name = $name")
    params["name"] = nameFilter
}

if ageFilter > 0 {
    conditions = append(conditions, "u.age > $age")
    params["age"] = ageFilter
}

where := strings.Join(conditions, " AND ")
query := fmt.Sprintf("MATCH (u:User) WHERE %s RETURN u", where)
```

**After**:
```go
qb := repo.Query()

if nameFilter != "" {
    qb = qb.Where("name = $name", map[string]interface{}{"name": nameFilter})
}

if ageFilter > 0 {
    qb = qb.Where("age > $age", map[string]interface{}{"age": ageFilter})
}

users, err := qb.FindAll()
```

---

## Performance Comparison

### Query Execution

| Operation | Raw Client | ORM | Overhead |
|-----------|------------|-----|----------|
| Simple CRUD | Baseline | +5-10% | Minimal (metadata lookup) |
| Complex Query | Baseline | +2-5% | Query builder overhead |
| Batch Insert | Baseline | Similar | Optimized batching |
| Relationship Load | Manual | Optimized | Lazy loading reduces queries |

### Memory Usage

- **Metadata**: Cached once at registration (~1-5KB per entity type)
- **Identity Map**: Per-session, proportional to loaded entities
- **Type Converters**: Negligible (interface dispatch)

### Best Practices for Performance

1. **Register entities once** at application startup
2. **Reuse repositories** (they're stateless and thread-safe)
3. **Use eager loading** (`Fetch()`) when you know you'll need relationships
4. **Batch operations** with `SaveAll()` instead of individual `Save()`
5. **Use indexes** for frequently queried properties

---

## Gradual Migration Strategy

You can migrate incrementally without rewriting everything at once:

### Step 1: Start with New Code

Use the ORM for new features while keeping existing code unchanged.

### Step 2: Migrate Simple CRUD

Replace simple Create/Read/Update/Delete operations first.

### Step 3: Migrate Complex Queries

Move complex queries to the query builder API.

### Step 4: Add Relationships

Introduce relationship management for interconnected entities.

### Step 5: Add Security

Layer on RBAC security where needed.

### Coexistence Example

```go
// Old code using raw client
func legacyFunction(graph *falkordb.Graph) {
    query := "MATCH (u:User) RETURN u"
    result, _ := graph.Query(query, nil, nil)
    // ...
}

// New code using ORM
func newFunction(repo *orm.Repository[*User]) {
    users, _ := repo.FindAll()
    // ...
}

// Both work with the same graph connection
graph := client.SelectGraph("myapp")
legacyFunction(graph)
newFunction(orm.NewRepository[*User](graph, registry))
```

---

## Troubleshooting Migration Issues

### Issue: "Entity not registered"

**Solution**: Ensure you call `registry.Register(&Entity{})` before creating repositories.

### Issue: Type conversion errors

**Solution**: Make sure your struct field types match FalkorDB types or register custom converters.

### Issue: Relationship loading fails

**Solution**: Verify relationship metadata (type, direction) matches actual graph relationships.

### Issue: Performance degradation

**Solution**: Use eager loading for known relationships, batch operations, and check index usage.

---

## Getting Help

- Review [TUTORIAL.md](TUTORIAL.md) for step-by-step examples
- Check [API.md](API.md) for complete API reference
- See [ARCHITECTURE.md](ARCHITECTURE.md) for design details
- Open an issue on GitHub for migration-specific questions

---

**Ready to migrate?** Start with the [Quick Start](README.md#quick-start) guide!
