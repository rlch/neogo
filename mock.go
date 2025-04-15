package neogo

import (
	"context"
	"errors"
	"net/url"

	"github.com/neo4j/neo4j-go-driver/v5/neo4j"
)

// NewMock creates a mock neogo [Driver] for testing.
func NewMock() mockDriver {
	m := &mockBindings{}
	return &mockDriverImpl{
		mockBindings: m,
		driver: &driver{db: &mockNeo4jDriver{
			mockBindings: m,
		}},
	}
}

type (
	mockBindings struct {
		Current *mockBindingsNode
	}
	mockBindingsNode struct {
		Single  map[string]any
		Records []map[string]any
		Next    *mockBindingsNode
	}
	mockDriver interface {
		Driver

		Bind(record map[string]any)
		BindRecords(records []map[string]any)
		Clear()
	}
	mockDriverImpl struct {
		*mockBindings
		*driver
	}

	mockNeo4jDriver struct {
		*mockBindings
	}
	mockNeo4jSession struct {
		*mockBindings
		neo4j.SessionWithContext
	}
	mockNeo4jTx struct {
		*mockBindings
		neo4j.ManagedTransaction
	}
	mockNeo4jResult struct {
		neo4j.ResultWithContext
		records []*neo4j.Record
		cursor  int
		started bool
	}
)

var (
	_ mockDriver               = (*mockDriverImpl)(nil)
	_ neo4j.DriverWithContext  = (*mockNeo4jDriver)(nil)
	_ neo4j.SessionWithContext = (*mockNeo4jSession)(nil)
	_ neo4j.ManagedTransaction = (*mockNeo4jTx)(nil)
	_ neo4j.ResultWithContext  = (*mockNeo4jResult)(nil)
)

func (d *mockBindings) Bind(m map[string]any) {
	if d.Current == nil {
		d.Current = &mockBindingsNode{
			Single: m,
		}
		return
	}
	node := d.Current
	for node.Next != nil {
		node = node.Next
	}
	node.Next = &mockBindingsNode{Single: m}
}

func (d *mockBindings) BindRecords(m []map[string]any) {
	if d.Current == nil {
		d.Current = &mockBindingsNode{
			Records: m,
		}
		return
	}
	node := d.Current
	for node.Next != nil {
		node = node.Next
	}
	node.Next = &mockBindingsNode{Records: m}
}

func (d *mockBindings) Clear() {
	d.Current = nil
}

func (d *mockNeo4jDriver) ExecuteQueryBookmarkManager() neo4j.BookmarkManager {
	panic(errors.New("not implemented"))
}

func (d *mockNeo4jDriver) Target() url.URL {
	panic(errors.New("not implemented"))
}

func (d *mockNeo4jDriver) NewSession(ctx context.Context, config neo4j.SessionConfig) neo4j.SessionWithContext {
	return &mockNeo4jSession{mockBindings: d.mockBindings}
}

func (d *mockNeo4jDriver) VerifyConnectivity(ctx context.Context) error {
	return nil
}

func (d *mockNeo4jDriver) VerifyAuthentication(ctx context.Context, auth *neo4j.AuthToken) error {
	return nil
}

func (d *mockNeo4jDriver) Close(ctx context.Context) error {
	return nil
}

func (d *mockNeo4jDriver) IsEncrypted() bool {
	panic(errors.New("not implemented"))
}

func (d *mockNeo4jDriver) GetServerInfo(ctx context.Context) (neo4j.ServerInfo, error) {
	panic(errors.New("not implemented"))
}

func (s *mockNeo4jSession) LastBookmarks() neo4j.Bookmarks {
	return nil
}

func (s *mockNeo4jSession) BeginTransaction(ctx context.Context, configurers ...func(*neo4j.TransactionConfig)) (neo4j.ExplicitTransaction, error) {
	panic(errors.New("not implemented"))
}

func (s *mockNeo4jSession) ExecuteRead(ctx context.Context, work neo4j.ManagedTransactionWork, configurers ...func(*neo4j.TransactionConfig)) (any, error) {
	_, err := work(&mockNeo4jTx{mockBindings: s.mockBindings})
	return nil, err
}

func (s *mockNeo4jSession) ExecuteWrite(ctx context.Context, work neo4j.ManagedTransactionWork, configurers ...func(*neo4j.TransactionConfig)) (any, error) {
	_, err := work(&mockNeo4jTx{mockBindings: s.mockBindings})
	return nil, err
}

func (s *mockNeo4jSession) Run(ctx context.Context, cypher string, params map[string]any, configurers ...func(*neo4j.TransactionConfig)) (neo4j.ResultWithContext, error) {
	panic(errors.New("not implemented"))
}

func (s *mockNeo4jSession) Close(ctx context.Context) error {
	return nil
}

func (t *mockNeo4jTx) Run(ctx context.Context, cypher string, params map[string]any) (neo4j.ResultWithContext, error) {
	panic(errors.New("not implemented, fix regsitry"))
	// r := &mockNeo4jResult{}
	// toRecord := func(m map[string]any) (*neo4j.Record, error) {
	// 	n := len(m)
	// 	rec := &neo4j.Record{
	// 		Keys:   make([]string, n),
	// 		Values: make([]any, n),
	// 	}
	// 	var i int
	// 	for k, v := range m {
	// 		rec.Keys[i] = k
	// 		if _, ok := v.(INode); ok {
	// 			labels := internal.ExtractNodeLabels(v)
	// 			var props map[string]any
	// 			bytes, err := json.Marshal(v)
	// 			if err != nil {
	// 				return nil, err
	// 			}
	// 			err = json.Unmarshal(bytes, &props)
	// 			if err != nil {
	// 				return nil, err
	// 			}
	// 			rec.Values[i] = neo4j.Node{
	// 				Labels: labels,
	// 				Props:  props,
	// 			}
	// 		} else if _, ok := v.(IRelationship); ok {
	// 			typ := internal.ExtractRelationshipType(v)
	// 			var props map[string]any
	// 			bytes, err := json.Marshal(v)
	// 			if err != nil {
	// 				return nil, err
	// 			}
	// 			err = json.Unmarshal(bytes, &props)
	// 			if err != nil {
	// 				return nil, err
	// 			}
	// 			rec.Values[i] = neo4j.Relationship{
	// 				Type:  typ,
	// 				Props: props,
	// 			}
	// 		} else {
	// 			rec.Values[i] = v
	// 		}
	// 		i++
	// 	}
	// 	return rec, nil
	// }
	// if t.Current == nil {
	// 	panic(errors.New("mock client used without bindings for all transactions"))
	// }
	// bindings := *t.Current
	// t.Current = t.Current.Next
	// if bindings.Single != nil {
	// 	rec, err := toRecord(bindings.Single)
	// 	if err != nil {
	// 		return nil, err
	// 	}
	// 	r.records = []*neo4j.Record{rec}
	// } else if bindings.Records != nil {
	// 	r.records = make([]*neo4j.Record, len(bindings.Records))
	// 	for i, recMap := range bindings.Records {
	// 		rec, err := toRecord(recMap)
	// 		if err != nil {
	// 			return nil, err
	// 		}
	// 		r.records[i] = rec
	// 	}
	// }
	// return r, nil
}

func (r *mockNeo4jResult) Keys() ([]string, error) {
	return r.records[r.cursor].Keys, nil
}

func (r *mockNeo4jResult) NextRecord(ctx context.Context, record **neo4j.Record) bool {
	if r.cursor < len(r.records) {
		*record = r.records[r.cursor]
		r.cursor++
		return true
	}
	return false
}

func (r *mockNeo4jResult) Next(ctx context.Context) bool {
	if r.cursor == 0 && !r.started {
		r.started = true
	} else {
		r.cursor++
	}
	return r.cursor < len(r.records)
}

func (r *mockNeo4jResult) PeekRecord(ctx context.Context, record **neo4j.Record) bool {
	if r.cursor+1 < len(r.records) {
		*record = r.records[r.cursor+1]
		return true
	}
	return false
}

func (r *mockNeo4jResult) Peek(ctx context.Context) bool {
	return r.cursor+1 < len(r.records)
}

func (r *mockNeo4jResult) Err() error {
	return nil
}

func (r *mockNeo4jResult) Record() *neo4j.Record {
	return r.records[r.cursor]
}

func (r *mockNeo4jResult) Collect(ctx context.Context) ([]*neo4j.Record, error) {
	if r.cursor+1 == len(r.records) {
		return nil, nil
	}
	return r.records[r.cursor+1:], nil
}

func (r *mockNeo4jResult) Single(ctx context.Context) (*neo4j.Record, error) {
	return r.records[r.cursor], nil
}

func (r *mockNeo4jResult) Consume(ctx context.Context) (neo4j.ResultSummary, error) {
	panic(errors.New("not implemented"))
}

func (r *mockNeo4jResult) IsOpen() bool {
	return true
}
