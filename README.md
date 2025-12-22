# neogo

![logo](https://i.imgur.com/4bK7CqC.png)

[![Go Report Card](https://goreportcard.com/badge/github.com/rlch/neogo)](https://goreportcard.com/report/github.com/rlch/neogo) [![codecov](https://codecov.io/gh/rlch/neogo/branch/main/graph/badge.svg?token=K1NYHBQD1A)](https://codecov.io/gh/rlch/neogo) [![Go Reference](https://pkg.go.dev/badge/github.com/rlch/neogo.svg)](https://pkg.go.dev/github.com/rlch/neogo)

A lightweight Golang ORM for Neo4j with raw Cypher queries and type-safe result binding.

> [!WARNING]
> The neogo API is still in an experimental phase. Expect minor changes and
> additions until the first release.


## Overview

`neogo` provides a thin layer over the official Neo4j Go driver, letting you write raw Cypher queries while neogo handles:

- **Automatic marshalling/unmarshalling** between Go structs and Neo4j
- **Type-safe result binding** with named bindings
- **Schema management** with indexes and constraints from struct tags
- **Session and transaction** management with causal consistency support
- **Zero-reflection hot path** for high performance

## Getting Started

### Installation

```bash
go get github.com/rlch/neogo
```

### Define Your Models

```go
import "github.com/rlch/neogo"

type Person struct {
    neogo.Node `neo4j:"Person"`

    Name    string `neo4j:"name"`
    Surname string `neo4j:"surname"`
    Age     int    `neo4j:"age"`
}
```

### Connect and Query

```go
import (
    "github.com/neo4j/neo4j-go-driver/v5/neo4j"
    "github.com/rlch/neogo"
)

func main() {
    ctx := context.Background()

    // Create driver
    driver, err := neogo.New(
        "neo4j://localhost:7687",
        neo4j.BasicAuth("neo4j", "password", ""),
        neogo.WithTypes(&Person{}),
    )
    if err != nil {
        log.Fatal(err)
    }
    defer driver.DB().Close(ctx)

    // Create a person
    err = driver.Exec().
        Cypher(`
            CREATE (p:Person {id: $id, name: $name, surname: $surname, age: $age})
        `).
        RunWithParams(ctx, map[string]any{
            "id":      "some-unique-id",
            "name":    "Spongebob",
            "surname": "Squarepants",
            "age":     20,
        })

    // Query with result binding
    var person Person
    err = driver.Exec().
        Cypher(`
            MATCH (p:Person {name: $name})
            RETURN p
        `).
        RunWithParams(ctx, map[string]any{"name": "Spongebob"}, "p", &person)

    fmt.Printf("person: %s %s, age %d\n", person.Name, person.Surname, person.Age)
    // Output: person: Spongebob Squarepants, age 20
}
```

## API

### Basic Query Execution

```go
// Simple query with no results
err := driver.Exec().
    Cypher(`CREATE (n:Node {id: $id})`).
    RunWithParams(ctx, map[string]any{"id": "123"})

// Query with result binding - bindings are name/pointer pairs
var person Person
err := driver.Exec().
    Cypher(`MATCH (p:Person {id: $id}) RETURN p`).
    RunWithParams(ctx, map[string]any{"id": "123"}, "p", &person)

// Multiple result bindings
var person Person
var count int64
err := driver.Exec().
    Cypher(`
        MATCH (p:Person {id: $id})-[:KNOWS]->(f:Person)
        WITH p, count(f) AS cnt
        RETURN p, cnt
    `).
    RunWithParams(ctx, map[string]any{"id": "123"}, "p", &person, "cnt", &count)
```

### Sessions and Transactions

```go
// Read session with transaction
session := driver.ReadSession(ctx)
defer session.Close(ctx)

err := session.ReadTransaction(ctx, func(c neogo.Client) error {
    var people []Person
    return c.Cypher(`MATCH (p:Person) RETURN p`).
        Run(ctx, "p", &people)
})

// Write session with transaction
session := driver.WriteSession(ctx)
defer session.Close(ctx)

err := session.WriteTransaction(ctx, func(c neogo.Client) error {
    return c.Cypher(`CREATE (p:Person {name: $name})`).
        RunWithParams(ctx, map[string]any{"name": "Alice"})
})

// Explicit transaction control
tx, err := session.BeginTransaction(ctx)
defer tx.Close(ctx)

err = tx.Run(func(c neogo.Client) error {
    return c.Cypher(`CREATE (p:Person {name: $name})`).
        RunWithParams(ctx, map[string]any{"name": "Bob"})
})
if err != nil {
    tx.Rollback(ctx)
    return err
}
tx.Commit(ctx)
```

### Streaming Results

```go
// Stream results one-by-one for memory efficiency
var person Person
err := driver.Exec().
    Cypher(`MATCH (p:Person) RETURN p`).
    Stream(ctx, func() error {
        fmt.Printf("Processing: %s\n", person.Name)
        return nil
    }, "p", &person)

// Streaming with parameters
err := driver.Exec().
    Cypher(`MATCH (p:Person) WHERE p.age > $minAge RETURN p`).
    StreamWithParams(ctx, map[string]any{"minAge": 18}, func() error {
        fmt.Printf("Adult: %s\n", person.Name)
        return nil
    }, "p", &person)
```

### Schema Management

```go
// Define schema with struct tags
type Person struct {
    neogo.Node `neo4j:"Person"`

    Email string `neo4j:"email,unique"`           // Unique constraint
    Name  string `neo4j:"name,index"`             // Range index
    Bio   string `neo4j:"bio,fulltext"`           // Fulltext index
    Age   int    `neo4j:"age,notNull"`            // Not null constraint
}

// Auto-migrate schema
actions, err := driver.Schema().AutoMigrate(ctx)
for _, action := range actions {
    fmt.Printf("Applied: %s\n", action.Description)
}

// Check if migration is needed
needed, err := driver.Schema().NeedsMigration(ctx)
```

## Resources

- [Go Reference](https://pkg.go.dev/github.com/rlch/neogo)
- [Official Neo4j Go Driver](https://github.com/neo4j/neo4j-go-driver)
- [Neo4j Cypher Manual](https://neo4j.com/docs/cypher-manual/current/)


## Contributing

See [CONTRIBUTING.md](CONTRIBUTING.md) for detailed instructions on how to contribute to `neogo`.
