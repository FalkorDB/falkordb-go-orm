# FalkorDB Go ORM Tutorial

This tutorial will guide you through building a complete blog application using FalkorDB Go ORM, covering all major features step-by-step.

## What We'll Build

A blog platform with:
- Users with authentication
- Blog posts with tags
- Comments on posts
- Follow relationships between users
- Role-based access control

## Prerequisites

- Go 1.21 or higher
- FalkorDB running locally (or Docker: `docker run -p 6379:6379 falkordb/falkordb`)
- Basic understanding of Go and graph databases

## Step 1: Project Setup

Create a new Go project:

```bash
mkdir blog-app && cd blog-app
go mod init github.com/yourusername/blog-app
go get github.com/FalkorDB/falkordb-go-orm
go get github.com/FalkorDB/falkordb-go/v2
go get github.com/google/uuid
```

## Step 2: Define Your Entities

Create `models.go`:

```go
package main

import (
    "time"
    "github.com/FalkorDB/falkordb-go-orm/orm"
    "github.com/google/uuid"
)

// User represents a blog user
type User struct {
    ID        uuid.UUID           `falkor:"id,generated"`
    Username  string              `falkor:"username,unique,indexed"`
    Email     string              `falkor:"email,unique,indexed"`
    Password  string              `falkor:"password"` // In production, hash this!
    Bio       string              `falkor:"bio"`
    CreatedAt time.Time           `falkor:"created_at"`
    Following orm.LazyList[*User] `falkor:"rel=FOLLOWS,dir=out,lazy"`
    Followers orm.LazyList[*User] `falkor:"rel=FOLLOWS,dir=in,lazy"`
}

func (u User) Labels() []string {
    return []string{"User"}
}

// Post represents a blog post
type Post struct {
    ID        uuid.UUID              `falkor:"id,generated"`
    Title     string                 `falkor:"title,indexed"`
    Content   string                 `falkor:"content"`
    Slug      string                 `falkor:"slug,unique,indexed"`
    Published bool                   `falkor:"published,indexed"`
    CreatedAt time.Time              `falkor:"created_at,indexed"`
    Author    orm.LazySingle[*User]  `falkor:"rel=AUTHORED,dir=in,lazy"`
    Tags      orm.LazyList[*Tag]     `falkor:"rel=TAGGED,dir=out,lazy"`
    Comments  orm.LazyList[*Comment] `falkor:"rel=HAS_COMMENT,dir=out,lazy"`
}

func (p Post) Labels() []string {
    return []string{"Post"}
}

// Tag for categorizing posts
type Tag struct {
    ID    uuid.UUID           `falkor:"id,generated"`
    Name  string              `falkor:"name,unique,indexed"`
    Posts orm.LazyList[*Post] `falkor:"rel=TAGGED,dir=in,lazy"`
}

func (t Tag) Labels() []string {
    return []string{"Tag"}
}

// Comment on a post
type Comment struct {
    ID        uuid.UUID             `falkor:"id,generated"`
    Content   string                `falkor:"content"`
    CreatedAt time.Time             `falkor:"created_at,indexed"`
    Author    orm.LazySingle[*User] `falkor:"rel=COMMENTED,dir=in,lazy"`
    Post      orm.LazySingle[*Post] `falkor:"rel=HAS_COMMENT,dir=in,lazy"`
}

func (c Comment) Labels() []string {
    return []string{"Comment"}
}
```

## Step 3: Connect to FalkorDB

Create `main.go`:

```go
package main

import (
    "fmt"
    "log"
    "time"
    
    "github.com/FalkorDB/falkordb-go/v2"
    "github.com/FalkorDB/falkordb-go-orm/orm"
    "github.com/google/uuid"
)

func main() {
    // Connect to FalkorDB
    client := falkordb.NewFalkorDB("localhost:6379", 
        falkordb.ClientOption{})
    graph := client.SelectGraph("blog")
    
    // Create entity registry
    registry := orm.NewEntityRegistry()
    registry.Register(&User{})
    registry.Register(&Post{})
    registry.Register(&Tag{})
    registry.Register(&Comment{})
    
    fmt.Println("Connected to FalkorDB!")
    
    // We'll add more code here
}
```

## Step 4: Create Repositories

Add repository setup to `main.go`:

```go
// Create repositories
userRepo := orm.NewRepository[*User](graph, registry)
postRepo := orm.NewRepository[*Post](graph, registry)
tagRepo := orm.NewRepository[*Tag](graph, registry)
commentRepo := orm.NewRepository[*Comment](graph, registry)

// Create relationship manager
relMgr := orm.NewRelationshipManager(graph, registry)
```

## Step 5: Create Sample Data

Add data creation functions:

```go
func createSampleData(
    userRepo *orm.Repository[*User],
    postRepo *orm.Repository[*Post],
    tagRepo *orm.Repository[*Tag],
    commentRepo *orm.Repository[*Comment],
    relMgr *orm.RelationshipManager,
) error {
    // Create users
    alice := &User{
        Username:  "alice",
        Email:     "alice@example.com",
        Password:  "hashed_password_1",
        Bio:       "Software engineer and blogger",
        CreatedAt: time.Now(),
    }
    
    bob := &User{
        Username:  "bob",
        Email:     "bob@example.com",
        Password:  "hashed_password_2",
        Bio:       "Tech enthusiast",
        CreatedAt: time.Now(),
    }
    
    if err := userRepo.SaveAll([]*User{alice, bob}); err != nil {
        return err
    }
    fmt.Println("✓ Created users")
    
    // Create follow relationship
    if err := relMgr.SaveRelationships(bob, "Following", []interface{}{alice}); err != nil {
        return err
    }
    fmt.Println("✓ Bob follows Alice")
    
    // Create tags
    goTag := &Tag{Name: "golang"}
    graphTag := &Tag{Name: "graph-databases"}
    
    if err := tagRepo.SaveAll([]*Tag{goTag, graphTag}); err != nil {
        return err
    }
    fmt.Println("✓ Created tags")
    
    // Create posts
    post1 := &Post{
        Title:     "Getting Started with FalkorDB",
        Content:   "FalkorDB is an amazing graph database...",
        Slug:      "getting-started-falkordb",
        Published: true,
        CreatedAt: time.Now(),
    }
    
    post2 := &Post{
        Title:     "Go Generics in Practice",
        Content:   "Go 1.21 introduced generics...",
        Slug:      "go-generics-practice",
        Published: true,
        CreatedAt: time.Now().Add(-24 * time.Hour),
    }
    
    if err := postRepo.SaveAll([]*Post{post1, post2}); err != nil {
        return err
    }
    fmt.Println("✓ Created posts")
    
    // Link posts to author
    if err := relMgr.SaveRelationships(post1, "Author", []interface{}{alice}); err != nil {
        return err
    }
    if err := relMgr.SaveRelationships(post2, "Author", []interface{}{alice}); err != nil {
        return err
    }
    
    // Tag posts
    if err := relMgr.SaveRelationships(post1, "Tags", []interface{}{graphTag}); err != nil {
        return err
    }
    if err := relMgr.SaveRelationships(post2, "Tags", []interface{}{goTag}); err != nil {
        return err
    }
    fmt.Println("✓ Linked posts to authors and tags")
    
    // Create comments
    comment := &Comment{
        Content:   "Great article!",
        CreatedAt: time.Now(),
    }
    
    if err := commentRepo.Save(comment); err != nil {
        return err
    }
    
    if err := relMgr.SaveRelationships(comment, "Author", []interface{}{bob}); err != nil {
        return err
    }
    if err := relMgr.SaveRelationships(comment, "Post", []interface{}{post1}); err != nil {
        return err
    }
    fmt.Println("✓ Created comment")
    
    return nil
}
```

## Step 6: Query Data with Derivation

Add query functions demonstrating query derivation:

```go
func queryExamples(
    userRepo *orm.Repository[*User],
    postRepo *orm.Repository[*Post],
) {
    fmt.Println("\n=== Query Examples ===")
    
    // Find user by username
    users, err := userRepo.FindByUsername("alice")
    if err != nil {
        log.Fatal(err)
    }
    if len(users) > 0 {
        fmt.Printf("✓ Found user: %s\n", users[0].Username)
    }
    
    // Find published posts
    posts, err := postRepo.FindByPublished(true)
    if err != nil {
        log.Fatal(err)
    }
    fmt.Printf("✓ Found %d published posts\n", len(posts))
    
    // Find posts by title containing keyword
    posts, err = postRepo.FindByTitleContains("FalkorDB")
    if err != nil {
        log.Fatal(err)
    }
    fmt.Printf("✓ Found %d posts about FalkorDB\n", len(posts))
    
    // Count published posts
    count, err := postRepo.CountByPublished(true)
    if err != nil {
        log.Fatal(err)
    }
    fmt.Printf("✓ Total published posts: %d\n", count)
}
```

## Step 7: Complex Queries with Query Builder

```go
func complexQueries(postRepo *orm.Repository[*Post]) {
    fmt.Println("\n=== Complex Queries ===")
    
    // Find recent published posts with pagination
    pageable := orm.NewPageable(0, 10)
    page, err := postRepo.Query().
        Where("published = $pub", map[string]interface{}{"pub": true}).
        OrderBy("created_at DESC").
        Paginate(pageable).
        FindPage()
    
    if err != nil {
        log.Fatal(err)
    }
    
    fmt.Printf("✓ Page 1: %d/%d posts\n", len(page.Content), page.TotalElements)
    for _, post := range page.Content {
        fmt.Printf("  - %s\n", post.Title)
    }
}
```

## Step 8: Eager Loading Relationships

```go
func eagerLoadingExample(postRepo *orm.Repository[*Post]) {
    fmt.Println("\n=== Eager Loading ===")
    
    // Load posts with their authors and tags
    posts, err := postRepo.Query().
        Where("published = $pub", map[string]interface{}{"pub": true}).
        Fetch("Author", "Tags").
        FindAll()
    
    if err != nil {
        log.Fatal(err)
    }
    
    for _, post := range posts {
        // Author is already loaded
        author := post.Author.Get()
        if len(author) > 0 {
            fmt.Printf("✓ Post: '%s' by %s\n", post.Title, author[0].Username)
        }
        
        // Tags are already loaded
        tags := post.Tags.Get()
        fmt.Printf("  Tags: ")
        for i, tag := range tags {
            if i > 0 {
                fmt.Print(", ")
            }
            fmt.Print(tag.Name)
        }
        fmt.Println()
    }
}
```

## Step 9: Use Sessions and Transactions

```go
func transactionExample(graph *falkordb.Graph, registry *orm.EntityRegistry) {
    fmt.Println("\n=== Transaction Example ===")
    
    session := orm.NewSession(graph, registry)
    
    err := orm.WithSession(session, func(s *orm.Session) error {
        // Create new user within transaction
        newUser := &User{
            Username:  "charlie",
            Email:     "charlie@example.com",
            Password:  "hashed_password_3",
            CreatedAt: time.Now(),
        }
        
        if err := s.Save(newUser); err != nil {
            return err
        }
        
        // Create post for the user
        post := &Post{
            Title:     "My First Post",
            Content:   "Hello world!",
            Slug:      "my-first-post",
            Published: true,
            CreatedAt: time.Now(),
        }
        
        if err := s.Save(post); err != nil {
            return err
        }
        
        // Link post to author
        relMgr := orm.NewRelationshipManager(graph, registry)
        if err := relMgr.SaveRelationships(post, "Author", []interface{}{newUser}); err != nil {
            return err
        }
        
        fmt.Println("✓ Created user and post in transaction")
        
        // Commit all changes
        return s.Commit()
    })
    
    if err != nil {
        log.Printf("Transaction failed: %v\n", err)
    } else {
        fmt.Println("✓ Transaction committed successfully")
    }
}
```

## Step 10: Add Indexes and Schema Validation

```go
func setupIndexes(graph *falkordb.Graph, registry *orm.EntityRegistry) {
    fmt.Println("\n=== Index Management ===")
    
    indexMgr := orm.NewIndexManager(graph)
    
    // Create indexes for all entities
    entities := []interface{}{&User{}, &Post{}, &Tag{}, &Comment{}}
    
    for _, entity := range entities {
        if err := indexMgr.EnsureIndexes(entity, orm.IndexTypeRange); err != nil {
            log.Printf("Warning: %v\n", err)
        }
    }
    
    fmt.Println("✓ Indexes created")
    
    // Validate schemas
    schemaMgr := orm.NewSchemaManager(registry, graph)
    for _, entity := range entities {
        result, err := schemaMgr.ValidateSchema(entity)
        if err != nil {
            log.Printf("Validation error: %v\n", err)
            continue
        }
        
        if result.IsValid {
            fmt.Printf("✓ Schema valid for %T\n", entity)
        } else {
            fmt.Printf("✗ Schema invalid for %T:\n", entity)
            for _, e := range result.Errors {
                fmt.Printf("  - %s\n", e)
            }
        }
    }
}
```

## Step 11: Complete Main Function

Update your `main.go`:

```go
func main() {
    // Connect to FalkorDB
    client := falkordb.NewFalkorDB("localhost:6379", falkordb.ClientOption{})
    graph := client.SelectGraph("blog")
    
    // Create entity registry
    registry := orm.NewEntityRegistry()
    registry.Register(&User{})
    registry.Register(&Post{})
    registry.Register(&Tag{})
    registry.Register(&Comment{})
    
    // Create repositories
    userRepo := orm.NewRepository[*User](graph, registry)
    postRepo := orm.NewRepository[*Post](graph, registry)
    tagRepo := orm.NewRepository[*Tag](graph, registry)
    commentRepo := orm.NewRepository[*Comment](graph, registry)
    relMgr := orm.NewRelationshipManager(graph, registry)
    
    fmt.Println("FalkorDB Go ORM Tutorial - Blog Application")
    fmt.Println("===========================================\n")
    
    // Setup indexes
    setupIndexes(graph, registry)
    
    // Clean existing data (for demo)
    fmt.Println("\nCleaning existing data...")
    graph.Query("MATCH (n) DETACH DELETE n", nil, nil)
    
    // Create sample data
    fmt.Println("\n=== Creating Sample Data ===")
    if err := createSampleData(userRepo, postRepo, tagRepo, commentRepo, relMgr); err != nil {
        log.Fatal(err)
    }
    
    // Run query examples
    queryExamples(userRepo, postRepo)
    complexQueries(postRepo)
    eagerLoadingExample(postRepo)
    transactionExample(graph, registry)
    
    fmt.Println("\n✅ Tutorial complete!")
}
```

## Step 12: Run the Application

```bash
go run main.go models.go
```

Expected output:
```
FalkorDB Go ORM Tutorial - Blog Application
===========================================

=== Index Management ===
✓ Indexes created
✓ Schema valid for *main.User
✓ Schema valid for *main.Post
✓ Schema valid for *main.Tag
✓ Schema valid for *main.Comment

Cleaning existing data...

=== Creating Sample Data ===
✓ Created users
✓ Bob follows Alice
✓ Created tags
✓ Created posts
✓ Linked posts to authors and tags
✓ Created comment

=== Query Examples ===
✓ Found user: alice
✓ Found 2 published posts
✓ Found 1 posts about FalkorDB
✓ Total published posts: 2

=== Complex Queries ===
✓ Page 1: 2/2 posts
  - Getting Started with FalkorDB
  - Go Generics in Practice

=== Eager Loading ===
✓ Post: 'Getting Started with FalkorDB' by alice
  Tags: graph-databases
✓ Post: 'Go Generics in Practice' by alice
  Tags: golang

=== Transaction Example ===
✓ Created user and post in transaction
✓ Transaction committed successfully

✅ Tutorial complete!
```

## Next Steps

Now that you've completed the tutorial, explore:

1. **Add Security**: Implement RBAC using `security.SecureRepository`
2. **Add Validation**: Validate user input before saving
3. **Optimize Queries**: Use indexes for frequently queried fields
4. **Add More Features**: Implement likes, bookmarks, full-text search
5. **Production Setup**: Add proper error handling, logging, and configuration

## Advanced Topics

### Adding RBAC Security

```go
import "github.com/FalkorDB/falkordb-go-orm/security"

// Create security context
ctx := security.NewSecurityContext(graph, registry, currentUserID)

// Define access rules
metadata := security.NewSecurityMetadataBuilder().
    RequirePermission("user", "read").
    RestrictProperty("email", "admin", "read").
    Build()

// Create secure repository
secureUserRepo := security.NewSecureRepository(userRepo, ctx, metadata)

// Queries now respect permissions
users, err := secureUserRepo.FindAll() // Email filtered for non-admins
```

### Custom Type Converters

```go
// Register custom converter for special types
type Metadata map[string]string

orm.GetTypeRegistry().Register(reflect.TypeOf(Metadata{}), 
    &MetadataConverter{})
```

### Graph Traversal Queries

```go
// Find friends of friends
query := `
MATCH (u:User {id: $userId})-[:FOLLOWS*2..2]->(fof:User)
RETURN DISTINCT fof
`

result, err := graph.Query(query, map[string]interface{}{
    "userId": userID.String(),
}, nil)
```

## Resources

- [API Reference](API.md) - Complete API documentation
- [Architecture Guide](ARCHITECTURE.md) - Internal design details
- [Migration Guide](MIGRATION.md) - Migrate from other ORMs
- [Examples](examples/) - More code examples

## Troubleshooting

**Issue**: Connection refused to FalkorDB

**Solution**: Ensure FalkorDB is running:
```bash
docker run -p 6379:6379 falkordb/falkordb
```

**Issue**: "Entity not registered" error

**Solution**: Make sure to call `registry.Register(&YourEntity{})` before using repositories.

**Issue**: Indexes not created

**Solution**: Check FalkorDB version supports index types. Use `IndexTypeRange` for basic indexes.

---

**Congratulations!** You've built a complete blog application with FalkorDB Go ORM. 🎉
