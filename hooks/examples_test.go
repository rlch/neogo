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
	mergeIDHook := hooks.New(func(c *hooks.Client) hooks.HookClient {
		return c.Merge(mergeP, mergeOptsP)
	}).Before(func(scope client.Scope) error {
		pat := mergeP.Head()
		if pat.Next() != nil {
			return nil
		}
		node, _, _ := scope.Unfold(pat.Data)
		name := scope.Name(node)
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
	if err != nil {
		panic(err)
	}

	fmt.Println(cy.Cypher)
	// Output:
	// MERGE (p:Person)
	//   ON CREATE
	//     SET p.id = $id
	// RETURN p
}
