package neogo

import (
	"context"
	"fmt"
	"strconv"
	"testing"
	"time"

	"github.com/neo4j/neo4j-go-driver/v5/neo4j"
	"github.com/stretchr/testify/assert"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"

	"github.com/rlch/neogo/db"
	"github.com/rlch/neogo/internal"
	"github.com/rlch/neogo/internal/tests"
)

func startContainer(t *testing.T, ctx context.Context) testcontainers.Container {
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
		t.Fatal("container should start: %w", err)
	}
	return container
}

func TestIntegration(t *testing.T) {
	ctx := context.Background()
	container := startContainer(t, ctx)
	defer func() {
		if err := container.Terminate(ctx); err != nil {
			t.Error(err)
		}
	}()
	port, err := container.MappedPort(ctx, "7687")
	if err != nil {
		t.Fatal(err)
	}
	uri := fmt.Sprintf("bolt://localhost:%d", port.Int())
	driver, err := neo4j.NewDriverWithContext(
		uri,
		neo4j.BasicAuth("neo4j", "password", ""),
	)
	if err != nil {
		t.Fatal(err)
	}
	c := New(driver)

	t.Run("Exec", func(t *testing.T) {
		p := tests.Person{
			Name:    "Spongebob",
			Surname: "Squarepants",
		}
		err := c.Exec().
			Create(db.Node(&p)).
			Set(db.SetPropValue(&p.Age, 20)).
			Return(&p).
			Run(ctx)
		assert.NoError(t, err)
		assert.Equal(t, tests.Person{
			Name: "Spongebob", Surname: "Squarepants",
			Age: 20,
		}, p)
	})

	t.Run("ReadSession", func(t *testing.T) {
		var ns, nsTimes2 []int
		session := c.ReadSession(ctx)
		defer func() {
			err := session.Close(ctx)
			assert.NoError(t, err)
		}()
		err := session.ReadTx(ctx, func(begin func() Client) error {
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
		assert.NoError(t, err)
		assert.Equal(t, []int{
			0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10,
		}, ns)
		assert.Equal(t, []int{
			0, 2, 4, 6, 8, 10, 12, 14, 16, 18, 20,
		}, nsTimes2)
	})

	t.Run("WriteSession", func(t *testing.T) {
		var people []*tests.Person
		session := c.WriteSession(ctx)
		defer func() {
			err := session.Close(ctx)
			assert.NoError(t, err)
		}()
		err := session.WriteTx(ctx, func(begin func() Client) error {
			if err := begin().
				Unwind("range(1, 10)", "i").
				Create(db.Node(
					db.Qual(
						tests.Person{},
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
		assert.NoError(t, err)
		p := func(i int) *tests.Person {
			return &tests.Person{
				NodeEntity: internal.NodeEntity{
					ID: strconv.Itoa(i),
				},
			}
		}
		assert.Equal(t, []*tests.Person{
			p(1), p(2), p(3), p(4), p(5), p(6), p(7), p(8), p(9), p(10),
		}, people)
	})
}
