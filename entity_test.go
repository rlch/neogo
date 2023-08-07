package neogo_test

import (
	"fmt"

	"github.com/rlch/neogo"
)

func ExampleNewNode() {
	n := neogo.NewNode[neogo.Node]()
	fmt.Printf("generated: %v", n.ID != "")
	// Output: generated: true
}

func ExampleNodeWithID() {
	n := neogo.NodeWithID[neogo.Node]("test")
	fmt.Printf("id: %v", n.ID)
	// Output: id: test
}
