# neogo

![logo](https://i.imgur.com/4bK7CqC.png)

[![Go Report Card](https://goreportcard.com/badge/github.com/rlch/neogo)](https://goreportcard.com/report/github.com/rlch/neogo) [![codecov](https://codecov.io/gh/rlch/neogo/branch/main/graph/badge.svg?token=K1NYHBQD1A)](https://codecov.io/gh/rlch/neogo) [![Go Reference](https://pkg.go.dev/badge/github.com/rlch/neogo.svg)](https://pkg.go.dev/github.com/rlch/neogo)

A Golang-ORM for Neo4J which creates idiomatic & fluent Cypher.

> [!WARNING]
> The neogo API is still in an experimental phase. Expect minor changes and
> additions until the first release.


## Overview

`neogo` was designed to make writing Cypher as simple as possible, providing a
safety-net and reducing boilerplate by leveraging canonical representations of
nodes and relationships as Golang structs. 

- Hands-free un/marshalling between Go and Neo4J
- No dynamic property, variable, label qualification necessary
- Creates readable, interoperable Cypher queries
- Abstract nodes with multiple concrete implementers
- Heavily tested; full coverage of Neo4J docs examples (see `internal/tests`)
- Automatic & explicit:
    - Variable qualification
    - Node/relationship label patterns
    - Parameter injection

## Getting Started

See the following resources to get started with `neogo`:

- [Docs](https://pkg.go.dev/github.com/rlch/neogo)
- [Tests](https://github.com/rlch/neogo/tree/main/internal/tests)
- [Official driver](https://github.com/neo4j/neo4j-go-driver)


## Example


```go
type Person struct {
	neogo.Node `neo4j:"Person"`

	Name    string `neo4j:"name"`
	Surname string `neo4j:"surname"`
	Age     int    `neo4j:"age"`
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
        Print().
        Run(ctx)
    // Output:
    // CREATE (person:Person {name: $person_name, surname: $person_surname})
    // SET person.age = $v1
    // RETURN person

    fmt.Printf("person: %v\n", person)
    // Output:
    // person: {{some-unique-id} Spongebob Squarepants 20}
}
```


## Contributions

See [the contributing guide](CONTRIBUTING.md) for detailed instructions on how to start contibuting to `neogo`.
