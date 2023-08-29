package neogo

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/neo4j/neo4j-go-driver/v5/neo4j"
	"github.com/rlch/neogo/client"
	"github.com/rlch/neogo/db"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
)

func startNeo4J(ctx context.Context) (neo4j.DriverWithContext, func(context.Context) error) {
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
	return driver, container.Terminate
}

type Person struct {
	Node `neo4j:"Person"`

	Name    string `json:"name"`
	Surname string `json:"surname"`
	Age     int    `json:"age"`
}

func ExampleDriver() {
	if testing.Short() {
		fmt.Println("err: <nil>")
		fmt.Printf("person: %v\n", Person{
			Node: Node{
				ID: "some-unique-id",
			},
			Name:    "Spongebob",
			Surname: "Squarepants",
			Age:     20,
		})
		return
	}

	ctx := context.Background()
	neo4j, cancel := startNeo4J(ctx)
	d := New(neo4j)
	defer func() {
		if err := cancel(ctx); err != nil {
			panic(err)
		}
	}()

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
	if testing.Short() {
		fmt.Printf("err: %v\n", nil)
		fmt.Printf("ns:       %v\n", []int{
			0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10,
		})
		fmt.Printf("nsTimes2: %v\n", []int{
			0, 2, 4, 6, 8, 10, 12, 14, 16, 18, 20,
		})
		return
	}

	ctx := context.Background()
	neo4j, cancel := startNeo4J(ctx)
	d := New(neo4j)
	defer func() {
		if err := cancel(ctx); err != nil {
			panic(err)
		}
	}()

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
	if testing.Short() {
		fmt.Printf("err: %v\n", nil)
		fmt.Printf("ids: %v\n", []int{
			1, 2, 3, 4, 5, 6, 7, 8, 9, 10,
		})
		return
	}

	ctx := context.Background()
	neo4j, cancel := startNeo4J(ctx)
	d := New(neo4j)
	defer func() {
		if err := cancel(ctx); err != nil {
			panic(err)
		}
	}()

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
			Merge(db.Node(
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
			Where(db.And(
				db.Cond("p.id", "=", "toString(i)"),
			)).
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

func ExampleDriver_runWithParams() {
	if testing.Short() {
		fmt.Printf("err: %v\n", nil)
		fmt.Printf("ns: %v\n", []int{1, 2, 3})
		return
	}

	ctx := context.Background()
	neo4j, cancel := startNeo4J(ctx)
	d := New(neo4j)
	defer func() {
		if err := cancel(ctx); err != nil {
			panic(err)
		}
	}()

	var ns []int
	err := d.Exec().
		Return(db.Qual(&ns, "$ns")).
		RunWithParams(ctx, map[string]interface{}{
			"ns": []int{1, 2, 3},
		})

	fmt.Printf("err: %v\n", err)
	fmt.Printf("ns: %v\n", ns)

	// Output: err: <nil>
	// ns: [1 2 3]
}

func ExampleDriver_streamWithParams() {
	if testing.Short() {
		fmt.Printf("err: %v\n", nil)
		fmt.Printf("ns: %v\n", []int{0, 1, 2, 3})
		return
	}

	ctx := context.Background()
	neo4j, cancel := startNeo4J(ctx)
	d := New(neo4j)
	defer func() {
		if err := cancel(ctx); err != nil {
			panic(err)
		}
	}()

	ns := []int{}
	session := d.ReadSession(ctx)
	defer func() {
		if err := session.Close(ctx); err != nil {
			panic(err)
		}
	}()
	err := session.ReadTx(ctx, func(begin func() client.Client) error {
		var num int
		params := map[string]interface{}{
			"total": 3,
		}
		return d.Exec().Unwind("range(0, $total)", "i").
			Return(db.Qual(&num, "i")).
			StreamWithParams(ctx, params, func(r client.Result) error {
				for i := 0; r.Next(ctx); i++ {
					if err := r.Read(); err != nil {
						return err
					}
					ns = append(ns, num)
				}
				return nil
			})
	})

	fmt.Printf("err: %v\n", err)
	fmt.Printf("ns: %v\n", ns)
	// Output: err: <nil>
	// ns: [0 1 2 3]
}
