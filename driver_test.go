package neogo_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/rlch/neogo"
	"github.com/rlch/neogo/client"
	"github.com/rlch/neogo/db"
	testutils "github.com/rlch/neogo/test_utils"
)

type Person struct {
	neogo.Node `neo4j:"Person"`

	Name    string `json:"name"`
	Surname string `json:"surname"`
	Age     int    `json:"age"`
}

func ExampleDriver() {
	if testing.Short() {
		fmt.Println("err: <nil>")
		fmt.Printf("person: %v\n", Person{
			Node: neogo.Node{
				ID: "some-unique-id",
			},
			Name:    "Spongebob",
			Surname: "Squarepants",
			Age:     20,
		})
		return
	}

	ctx := context.Background()
	neo4j, cancel := testutils.StartNeo4J(ctx)
	d := neogo.New(neo4j)
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
	neo4j, cancel := testutils.StartNeo4J(ctx)
	d := neogo.New(neo4j)
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
	neo4j, cancel := testutils.StartNeo4J(ctx)
	d := neogo.New(neo4j)
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
