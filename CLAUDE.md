# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Overview

`neogo` is a Golang ORM for Neo4J that creates idiomatic & fluent Cypher queries. It provides type-safe query building with automatic marshalling/unmarshalling between Go structs and Neo4J nodes/relationships.

## Architecture

### Core Components

- **Driver** (`driver.go`): Main entry point, wraps neo4j.DriverWithContext and provides query execution
- **Entity System** (`entity.go`): Node and relationship definitions with INode/IAbstract interfaces
- **Query Builder** (`query/`): Fluent API for building Cypher queries
- **DB Package** (`db/`): Building blocks for Cypher clauses (patterns, expressions, variables)
- **Internal** (`internal/`): Core compilation logic, scope management, parameter handling

### Key Architecture Patterns

1. **Type-Safe Query Building**: Uses Go generics and interfaces to ensure compile-time safety
2. **Fluent API**: Chainable methods for building complex Cypher queries
3. **Automatic Parameter Injection**: Converts Go values to Neo4J parameters automatically
4. **Abstract Node Support**: Allows multiple concrete implementations of abstract node types
5. **Scope-based Variable Management**: Automatic variable qualification and naming in complex queries

### Entity System

- **Node**: Base struct for Neo4J nodes with ID, labels via struct tags
- **Relationship**: Base struct for Neo4J relationships with type via struct tags
- **Abstract**: Interface for nodes with multiple concrete implementations
- **Registry**: Manages mappings between abstract interfaces and concrete types

## Development Commands

### Testing

```bash
# Run all tests
go test ./...

# Run tests with coverage
go test -cover ./...

# Run specific test package
go test ./internal/tests

# Run single test file
go test ./internal/tests/match_test.go
```

### Building

```bash
# Build the module
go build ./...

# Compile without running
go build -o neogo
```

### Linting & Formatting

```bash
# Format code
go fmt ./...

# Vet code
go vet ./...
```

## Test Structure

- `internal/tests/`: Comprehensive test suite covering all Cypher clause types
- Tests are organized by Cypher clause (match, create, set, etc.)
- `common.go` provides shared test utilities and Neo4J container setup
- Uses testcontainers-go for integration testing with real Neo4J instances

## Key Files to Understand

- `driver.go`: Main API entry points and interfaces
- `entity.go`: Core entity types and constructors
- `internal/cypher.go`: Query compilation and execution logic
- `internal/scope.go`: Variable scoping and parameter management
- `db/patterns.go`: Node and relationship pattern builders
- `query/client.go`: Fluent query building API

## Development Notes

- Uses Neo4J Go driver v5
- Requires Go 1.22+
- Heavily tested with full coverage of Neo4J documentation examples
- API is experimental and subject to change before v1.0

## Codec System (internal/codec)

Handles encoding/decoding of Go structs to/from Neo4J data. Zero-reflection hot path architecture.

## Communication Guidelines

- Never write throwaway markdown documents
- Prefer communicating in chat window

