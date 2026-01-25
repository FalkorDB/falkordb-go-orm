# Contributing to FalkorDB Go ORM

Thank you for your interest in contributing to FalkorDB Go ORM! This project is under active development and we welcome contributions.

## Project Status

**Current Version**: 1.0.0-beta (Production Ready)

All 7 core phases complete:
- ✅ Phase 1: Core Entity Mapping
- ✅ Phase 2: Repository Pattern
- ✅ Phase 3: Query Derivation  
- ✅ Phase 4: Relationships
- ✅ Phase 5: Sessions & Transactions
- ✅ Phase 6: RBAC Security
- ✅ Phase 7: Advanced Features (Indexes, Schema Validation)
- 🚧 Phase 8: Documentation & Examples (In Progress)

**Current focus**: Documentation, additional examples, and community building

## Development Setup

### Prerequisites

- Go 1.21 or higher (for generics support)
- FalkorDB instance running (for integration tests)
- Git

### Getting Started

1. Fork the repository
2. Clone your fork:
   ```bash
   git clone https://github.com/YOUR_USERNAME/falkordb-go-orm
   cd falkordb-go-orm
   ```

3. Install dependencies:
   ```bash
   go mod download
   ```

4. Run tests:
   ```bash
   go test ./...
   ```

## Project Structure

```
falkordb-go-orm/
├── orm/                    # Core ORM implementation
│   ├── entity.go          # Entity metadata and registry
│   ├── mapper.go          # Struct ↔ Cypher mapping
│   ├── types.go           # Type conversion system
│   ├── repository.go      # Repository pattern
│   ├── query_builder.go   # Fluent query API
│   ├── query_parser.go    # Method name derivation
│   ├── relationships.go   # Relationship management
│   ├── session.go         # Session and transactions
│   ├── indexes.go         # Index management
│   ├── schema.go          # Schema validation
│   └── pagination.go      # Pagination support
├── security/              # RBAC security module
│   ├── models.go          # Security entities
│   ├── metadata.go        # Security metadata
│   ├── context.go         # Security context
│   ├── repository.go      # Secure repository
│   └── manager.go         # RBAC manager
├── examples/              # Example applications
│   ├── basic/
│   ├── repository/
│   ├── query_derivation/
│   ├── relationships/
│   ├── session/
│   ├── security/
│   └── indexes/
└── docs/                  # Additional documentation
```

## How to Contribute

### Reporting Bugs

1. Check if the bug has already been reported in [Issues](../../issues)
2. If not, create a new issue with:
   - Clear title and description
   - Steps to reproduce
   - Expected vs actual behavior
   - Go version and OS

### Suggesting Features

1. Open an issue with the `enhancement` label
2. Describe the feature and its use case
3. Consider how it fits into the existing design

### Submitting Pull Requests

1. Create a new branch:
   ```bash
   git checkout -b feature/your-feature-name
   ```

2. Make your changes following the coding standards

3. Add tests for your changes

4. Run tests and ensure they pass:
   ```bash
   go test ./...
   go fmt ./...
   go vet ./...
   ```

5. Commit with clear messages:
   ```bash
   git commit -m "feat: add query builder support for IN operator"
   ```

6. Push to your fork and create a Pull Request

## Coding Standards

### Go Style

- Follow standard Go conventions
- Use `gofmt` for formatting
- Run `go vet` to catch common issues
- Add comments for exported functions

### Naming Conventions

- Use clear, descriptive names
- Follow Go naming conventions (camelCase for private, PascalCase for public)
- Prefix interfaces with `I` only when necessary (e.g., avoid `IConverter`, use `Converter`)

### Testing

- Write unit tests for all new functionality
- Aim for >80% code coverage
- Use table-driven tests where appropriate
- Add integration tests for database interactions

Example test structure:
```go
func TestMapperCreate(t *testing.T) {
    tests := []struct {
        name    string
        entity  interface{}
        want    string
        wantErr bool
    }{
        {
            name: "simple person",
            entity: &Person{Name: "Alice", Age: 25},
            want: "CREATE (n:Person) SET ...",
            wantErr: false,
        },
        // More test cases...
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            mapper := NewMapper()
            got, _, err := mapper.MapToCreate(tt.entity)
            if (err != nil) != tt.wantErr {
                t.Errorf("MapToCreate() error = %v, wantErr %v", err, tt.wantErr)
                return
            }
            // Assert result...
        })
    }
}
```

### Documentation

- Add GoDoc comments for all exported types and functions
- Update README.md for significant changes
- Add examples in `examples/` directory for new features

## Development Workflow

### Contribution Areas

With core features complete, we welcome contributions in:

**Documentation**:
- Expand API documentation
- Add more tutorials and guides
- Improve code examples
- Add architecture diagrams

**Examples**:
- Real-world application examples
- Industry-specific use cases
- Performance optimization examples
- Migration guides from other ORMs

**Features**:
- Additional query operators
- Performance optimizations
- New index types (e.g., geospatial)
- Schema migration tools

**Testing**:
- Increase test coverage
- Add more integration tests
- Performance benchmarks
- Stress testing

**Bug Fixes**:
- Fix reported issues
- Improve error messages
- Handle edge cases

## Communication

- Join the [FalkorDB Discord](https://discord.gg/falkordb)
- Use GitHub Issues for bug reports and feature requests
- Use Pull Requests for code contributions

## License

By contributing, you agree that your contributions will be licensed under the BSD-3-Clause License.

## Questions?

Feel free to open an issue or ask in the Discord community!

---

Thank you for contributing to FalkorDB Go ORM! 🚀
