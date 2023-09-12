package neogo

import (
	"context"
	"testing"
)

func newHybridDriver(t *testing.T, ctx context.Context) (d Driver, m mockDriver) {
	m = NewMock()
	if testing.Short() {
		d = m
	} else {
		neo4j, cancel := startNeo4J(ctx)
		d = New(neo4j)
		t.Cleanup(func() {
			if err := cancel(ctx); err != nil {
				t.Fatal(err)
			}
		})
	}
	return
}
