# neogo

[![Coverage Status](https://coveralls.io/repos/github/rlch/neogo/badge.svg?branch=main)](https://coveralls.io/github/rlch/neogo?branch=main) [![Go Report Card](https://goreportcard.com/badge/github.com/rlch/neogo)](https://goreportcard.com/report/github.com/rlch/neogo) [![Go Reference](https://pkg.go.dev/badge/github.com/rlch/neogo.svg)](https://pkg.go.dev/github.com/rlch/neogo)

A Golang-ORM for Neo4J which creates idiomatic & fluent Cypher.

Neogo was designed to make writing Cypher as simple as possible, providing a
safety-net and reducing boilerplate by leveraging canonical representations of
nodes and relationships. 

> [!WARNING]
> The neogo API is still in an experimental phase. Expect minor changes and
> additions until the first release.

---

## Overview

- Hands-free un/marshalling between Go and Neo4J
- Automatic & explicit:
    - Parameter injection
    - Variable qualification
    - Node/relationship label patterns
- No dynamic property, variable, label qualification necessary
- Abstract nodes with multiple concrete implementers
- Creates readable, interoperable Cypher queries
- Heavily tested; full coverage of Neo4J docs examples (see `internal/tests`)

## Getting Started

See the following resources to get started with neogo:

- [Docs](https://pkg.go.dev/github.com/rlch/neogo)
- [Tests](https://github.com/rlch/neogo/tree/main/internal/tests)
- [Official driver](https://github.com/neo4j/neo4j-go-driver)


## Example


```go
type Person struct {
	neogo.Node `neo4j:"Person"`

	Name    string `json:"name"`
	Surname string `json:"surname"`
	Age     int    `json:"age"`
}

func main() {
    // Simply obtain an instance of the neo4j.DriverWithContext
    d := neogo.New(driverWithContext)

    person := Person{
        Name:    "Spongebob",
        Surname: "Squarepants",
    }
    // person.GenerateID() can be used
    person.ID = "some-unique-id"

    err := d.Exec().
        Create(db.Node(&person)).
        Set(db.SetPropValue(&person.Age, 20)).
        Return(&person).
        Run(ctx)

    fmt.Printf("person: %v\n", person)
    // person: {{some-unique-id} Spongebob Squarepants 20}
}
```


## Contributions

See [the contributing guide](CONTRIBUTING.md) for detailed instructions on how to get started with our project.
