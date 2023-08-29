package hooks_test

import (
	"fmt"

	"github.com/rlch/neogo"
	"github.com/rlch/neogo/client"
	"github.com/rlch/neogo/db"
	"github.com/rlch/neogo/hooks"
	"github.com/rlch/neogo/internal"
	"github.com/rlch/neogo/internal/tests"
)

func Example() {
	mergeP := new(hooks.PatternMatcher)
	mergeOptsP := new(hooks.MergeMatcher)
	mergeIDHook := hooks.
		New(func(c *hooks.Client) hooks.HookClient {
			return c.Merge(mergeP, mergeOptsP)
		}).
		Mutate(func(scope client.Scope) error {
			pat := mergeP.Head()
			for {
				node, _, _ := scope.Unfold(pat.Data)
				fmt.Printf("%+v\n", node)
				name := scope.Name(node)
				fmt.Println("name:", name)
				if n, ok := node.(internal.IDSetter); ok {
					if n.GetID() != "" {
						return nil
					}
					genID := db.NamedParam("new-id", "id")
					mergeOptsP.Merge.OnCreate = append(
						mergeOptsP.Merge.OnCreate,
						db.SetPropValue(name+".id", genID),
					)
				}

				pat = pat.Next()
				if pat == nil {
					break
				}
			}
			return nil
		})

	driver := neogo.New(nil)
	driver.UseHooks(mergeIDHook)
	c := driver.Exec()
	var p tests.Person
	cy, err := c.
		Merge(db.Node(db.Qual(&p, "p"))).
		Return(&p).
		Compile()
		// hooks.Use[client.Yielder](
		// 	c.Merge(nil),
		// 	hooks.Paginate(),
		// ).Yield()
	if err != nil {
		panic(err)
	}

	fmt.Println(cy.Cypher)
	// Output:
	// MERGE (p:Person)
	// ON CREATE
	//   SET p.id = $id
	// RETURN p
}
