# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Overview

`neogo` is a lightweight Golang ORM for Neo4j that provides raw Cypher query execution with type-safe result binding. Unlike traditional ORMs with fluent query builders, neogo takes a minimalist approach: you write raw Cypher queries and neogo handles marshalling/unmarshalling between Go structs and Neo4j.

## Architecture

### Core Components

- **Driver** (`driver.go`): Main entry point, wraps neo4j.DriverWithContext with sessions and transactions
- **Client** (`client_impl.go`): Query interface returned by `Driver.Exec()`, provides `Cypher()` method
- **Entity System** (`entity.go`): Node and relationship definitions with INode/IRelationship interfaces
- **Registry** (`registry.go`): Type registration for nodes and relationships
- **Schema** (`schema.go`, `schema_impl.go`): Index and constraint management from struct tags

### Internal Components

- **internal/codec/**: Zero-reflection encoding/decoding system with opcode-based compilation
- **internal/binding.go**: Result binding from Neo4j records to Go structs
- **internal/binding_plan.go**: Pre-compiled binding metadata for zero-reflection hot path
- **internal/cypher.go**: Cypher query compilation with parameter handling
- **internal/registry.go**: Type metadata and Neo4j label/relationship type extraction

### Key Design Principles

1. **Raw Cypher**: Users write raw Cypher queries, neogo doesn't generate Cypher
2. **Type-Safe Binding**: Result binding is name/pointer pairs: `Run(ctx, "name", &target, "name2", &target2)`
3. **Zero-Reflection Hot Path**: Pre-compiled codecs and binding plans avoid reflection during query execution
4. **Schema from Tags**: Indexes/constraints defined via struct tags, applied with AutoMigrate

### API Pattern

```go
// The main pattern: Cypher() returns Runner, then Run/RunWithParams/Stream/StreamWithParams
driver.Exec().
    Cypher(`MATCH (p:Person {id: $id}) RETURN p`).
    RunWithParams(ctx, map[string]any{"id": "123"}, "p", &person)

// Bindings are variadic name/pointer pairs
Run(ctx, "name1", &binding1, "name2", &binding2)
RunWithParams(ctx, params, "name1", &binding1, "name2", &binding2)
Stream(ctx, sinkFunc, "name1", &binding1)
StreamWithParams(ctx, params, sinkFunc, "name1", &binding1)
```

### Entity System

- **Node**: Base struct for Neo4j nodes with auto-ID generation via ULID
- **Relationship**: Base struct for Neo4j relationships
- **Abstract**: Interface for polymorphic nodes with multiple concrete implementations
- **One/Many**: Zero-cost phantom types for relationship cardinality in schema

## Development Commands

### Testing

```bash
# Run all tests
go test ./...

# Run tests with coverage
go test -cover ./...

# Run short tests only (skip integration tests)
go test -short ./...
```

### Building

```bash
# Build the module
go build ./...

# Verify module dependencies
go mod tidy
```

### Linting & Formatting

```bash
# Format code
go fmt ./...

# Vet code
go vet ./...

# Run golangci-lint (if installed)
golangci-lint run
```

## Test Structure

- Root level `*_test.go` files: Driver, client, mock, schema, registry tests
- `internal/codec/*_test.go`: Codec system tests
- Uses testcontainers-go for integration tests with real Neo4j instances
- Tests are marked with `testing.Short()` to skip integration tests in CI

## Key Files to Understand

- `driver.go`: Driver interface, session/transaction management
- `client_impl.go`: Client/Runner interfaces, query execution logic
- `entity.go`: Node, Relationship, Abstract base types
- `schema.go`, `schema_impl.go`: Schema interface and migration implementation
- `internal/binding.go`, `internal/binding_plan.go`: Result binding system
- `internal/codec/`: Zero-reflection codec system

## What Was Removed (Simplification)

The codebase was recently simplified by removing:
- `builder/` directory (fluent query builder interfaces)
- `db/` directory (Node, Patterns, Var, Qual, Props, Cond, Where DSL)
- `internal/tests/` (builder tests)
- `internal/cypher.go` write* methods (Cypher generation)
- `internal/scope.go` variable tracking
- `internal/patterns.go` pattern builder
- `internal/option.go` builder options

## Development Notes

- Uses Neo4J Go driver v5
- Requires Go 1.21+
- API is experimental and subject to change before v1.0

## Communication Guidelines

- Never write throwaway markdown documents
- Prefer communicating in chat window
