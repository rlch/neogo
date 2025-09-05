# CLAUDE.md

`neogo` - Golang ORM for Neo4J with type-safe fluent Cypher queries.

## Core Architecture

- **Driver** (`driver.go`): Main entry point & query execution
- **Entity System** (`entity.go`): Node/relationship structs with struct tags
- **Query Builder** (`query/`): Fluent API for building Cypher
- **DB Package** (`db/`): Cypher clause building blocks  
- **Internal** (`internal/`): Query compilation & parameter handling

**Key Features**: Type-safe generics, automatic parameter injection, abstract node support, scope-based variable management

## Development

**Prerequisites**: Go 1.22+, Docker, gotestsum (optional)

**Quick Start**:
```bash
task test        # All tests (auto manages Neo4J)
task test:unit   # Unit tests only (no Neo4J) 
task dev:setup   # Full development setup
```

**Manual Commands**:
```bash
go test -short ./...    # Unit tests
go test ./...          # All tests (needs local Neo4J)
go build ./...         # Build
```

**Key Files**:
- `driver.go` - Main API
- `internal/cypher.go` - Query compilation  
- `db/patterns.go` - Pattern builders
- `internal/tests/` - Comprehensive test suite