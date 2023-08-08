package neogo_test

import (
	"context"
	"fmt"
	"time"

	"github.com/neo4j/neo4j-go-driver/v5/neo4j"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"

	"github.com/rlch/neogo"
	"github.com/rlch/neogo/client"
	"github.com/rlch/neogo/db"
)

func startNeo4J(ctx context.Context) neo4j.DriverWithContext {
	request := testcontainers.ContainerRequest{
		Name:         "neo4j",
		Image:        "neo4j:5.7-enterprise",
		ExposedPorts: []string{"7687/tcp"},
		WaitingFor:   wait.ForLog("Bolt enabled").WithStartupTimeout(time.Minute * 2),
		Env: map[string]string{
			"NEO4J_AUTH":                     fmt.Sprintf("%s/%s", "neo4j", "password"),
			"NEO4J_PLUGINS":                  `["apoc"]`,
			"NEO4J_ACCEPT_LICENSE_AGREEMENT": "yes",
		},
	}
	container, err := testcontainers.GenericContainer(
		ctx, testcontainers.GenericContainerRequest{
			ContainerRequest: request,
			Started:          true,
			Reuse:            true,
		})
	if err != nil {
		panic(fmt.Errorf("container should start: %w", err))
	}
	// defer func() {
	// 	if err := container.Terminate(ctx); err != nil {
	// 		panic(err)
	// 	}
	// }()
	port, err := container.MappedPort(ctx, "7687")
	if err != nil {
		panic(err)
	}
	uri := fmt.Sprintf("bolt://localhost:%d", port.Int())
	driver, err := neo4j.NewDriverWithContext(
		uri,
		neo4j.BasicAuth("neo4j", "password", ""),
	)
	if err != nil {
		panic(err)
	}
	return driver
}

type Person struct {
	neogo.Node `neo4j:"Person"`

	Name    string `json:"name"`
	Surname string `json:"surname"`
	Age     int    `json:"age"`
}

func ExampleDriver() {
	ctx := context.Background()
	neo4j := startNeo4J(ctx)
	d := neogo.New(neo4j)

	person := Person{
		Name:    "Spongebob",
		Surname: "Squarepants",
	}
	person.ID = "some-unique-id"
	err := d.Exec().
		Create(db.Node(&person)).
		Set(db.SetPropValue(&person.Age, 20)).
		Return(&person).
		Run(ctx)
	fmt.Printf("err: %v\n", err)
	fmt.Printf("person: %v\n", person)
	// Output: err: <nil>
	// person: {{some-unique-id} Spongebob Squarepants 20}
}

func ExampleDriver_readSession() {
	ctx := context.Background()
	neo4j := startNeo4J(ctx)
	d := neogo.New(neo4j)

	var ns, nsTimes2 []int
	session := d.ReadSession(ctx)
	defer func() {
		if err := session.Close(ctx); err != nil {
			panic(err)
		}
	}()
	err := session.ReadTx(ctx, func(begin func() client.Client) error {
		if err := begin().
			Unwind("range(0, 10)", "i").
			Return(db.Qual(&ns, "i")).Run(ctx); err != nil {
			return err
		}
		if err := begin().
			Unwind(&ns, "i").
			Return(db.Qual(&nsTimes2, "i * 2")).Run(ctx); err != nil {
			return err
		}
		return nil
	})
	fmt.Printf("err: %v\n", err)

	fmt.Printf("ns:       %v\n", ns)
	fmt.Printf("nsTimes2: %v\n", nsTimes2)
	// Output: err: <nil>
	// ns:       [0 1 2 3 4 5 6 7 8 9 10]
	// nsTimes2: [0 2 4 6 8 10 12 14 16 18 20]
}

func ExampleDriver_writeSession() {
	ctx := context.Background()
	neo4j := startNeo4J(ctx)
	d := neogo.New(neo4j)

	var people []*Person
	session := d.WriteSession(ctx)
	defer func() {
		if err := session.Close(ctx); err != nil {
			panic(err)
		}
	}()
	err := session.WriteTx(ctx, func(begin func() client.Client) error {
		if err := begin().
			Unwind("range(1, 10)", "i").
			Create(db.Node(
				db.Qual(
					Person{},
					"p",
					db.Props{"id": "toString(i)"},
				),
			)).
			Run(ctx); err != nil {
			return err
		}
		if err := begin().
			Unwind("range(1, 10)", "i").
			Match(db.Node(db.Qual(&people, "p"))).
			Where(db.Cond("p.id", "=", "toString(i)")).
			Return(&people).
			Run(ctx); err != nil {
			return err
		}
		return nil
	})
	ids := make([]string, len(people))
	for i, p := range people {
		ids[i] = p.ID
	}
	fmt.Printf("err: %v\n", err)
	fmt.Printf("ids: %v\n", ids)
	// Output: err: <nil>
	// ids: [1 2 3 4 5 6 7 8 9 10]
}
