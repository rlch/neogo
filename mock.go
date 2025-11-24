package neogo

import (
	"context"
	"errors"
	"net/url"

	"github.com/neo4j/neo4j-go-driver/v5/neo4j"
	"github.com/rlch/neogo/internal"
	"golang.org/x/sync/semaphore"
)

// NewMock creates a mock neogo [Driver] for testing.
func NewMock() mockDriver {
	m := &mockBindings{}
	reg := internal.NewRegistry()
	return &mockDriverImpl{
		mockBindings: m,
		driver: &driver{
			db: &mockNeo4jDriverWithContext{
				mockBindings: m,
				Registry:     reg,
			},
			reg:              reg,
			sessionSemaphore: semaphore.NewWeighted(100),
		},
	}
}

type (
	mockBindings struct {
		Single  map[string]any
		Records []map[string]any
		Next    *mockBindings
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

	mockNeo4jDriverWithContext struct {
		*mockBindings
		*internal.Registry
	}
	mockNeo4jSessionWithContext struct {
		*mockNeo4jDriverWithContext
		neo4j.SessionWithContext
	}
	mockNeo4jManagedTransaction struct {
		*mockNeo4jSessionWithContext
		neo4j.ManagedTransaction
	}
	mockNeo4jResultWithContext struct {
		neo4j.ResultWithContext
		records []*neo4j.Record
		cursor  int
		started bool
	}
)

var (
	_ mockDriver               = (*mockDriverImpl)(nil)
	_ neo4j.DriverWithContext  = (*mockNeo4jDriverWithContext)(nil)
	_ neo4j.SessionWithContext = (*mockNeo4jSessionWithContext)(nil)
	_ neo4j.ManagedTransaction = (*mockNeo4jManagedTransaction)(nil)
	_ neo4j.ResultWithContext  = (*mockNeo4jResultWithContext)(nil)
)

func (d *mockBindings) Bind(m map[string]any) {
	b := d
	for b.Next != nil {
		b = b.Next
	}
	b.Single = m
	b.Next = &mockBindings{}
}

func (d *mockBindings) BindRecords(m []map[string]any) {
	b := d
	for b.Next != nil {
		b = b.Next
	}
	b.Records = m
	b.Next = &mockBindings{}
}

func (d *mockBindings) Clear() {
	d.Single = nil
	d.Records = nil
	d.Next = nil
}

func (d *mockNeo4jDriverWithContext) ExecuteQueryBookmarkManager() neo4j.BookmarkManager {
	panic(errors.New("not implemented"))
}

func (d *mockNeo4jDriverWithContext) Target() url.URL {
	panic(errors.New("not implemented"))
}

func (d *mockNeo4jDriverWithContext) NewSession(ctx context.Context, config neo4j.SessionConfig) neo4j.SessionWithContext {
	return &mockNeo4jSessionWithContext{mockNeo4jDriverWithContext: d}
}

func (d *mockNeo4jDriverWithContext) VerifyConnectivity(ctx context.Context) error {
	return nil
}

func (d *mockNeo4jDriverWithContext) VerifyAuthentication(ctx context.Context, auth *neo4j.AuthToken) error {
	return nil
}

func (d *mockNeo4jDriverWithContext) Close(ctx context.Context) error {
	return nil
}

func (d *mockNeo4jDriverWithContext) IsEncrypted() bool {
	panic(errors.New("not implemented"))
}

func (d *mockNeo4jDriverWithContext) GetServerInfo(ctx context.Context) (neo4j.ServerInfo, error) {
	panic(errors.New("not implemented"))
}

func (s *mockNeo4jSessionWithContext) LastBookmarks() neo4j.Bookmarks {
	return nil
}

func (s *mockNeo4jSessionWithContext) BeginTransaction(ctx context.Context, configurers ...func(*neo4j.TransactionConfig)) (neo4j.ExplicitTransaction, error) {
	panic(errors.New("not implemented"))
}

func (s *mockNeo4jSessionWithContext) ExecuteRead(ctx context.Context, work neo4j.ManagedTransactionWork, configurers ...func(*neo4j.TransactionConfig)) (any, error) {
	_, err := work(&mockNeo4jManagedTransaction{mockNeo4jSessionWithContext: s})
	return nil, err
}

func (s *mockNeo4jSessionWithContext) ExecuteWrite(ctx context.Context, work neo4j.ManagedTransactionWork, configurers ...func(*neo4j.TransactionConfig)) (any, error) {
	_, err := work(&mockNeo4jManagedTransaction{mockNeo4jSessionWithContext: s})
	return nil, err
}

func (s *mockNeo4jSessionWithContext) Run(ctx context.Context, cypher string, params map[string]any, configurers ...func(*neo4j.TransactionConfig)) (neo4j.ResultWithContext, error) {
	r := &mockNeo4jResultWithContext{}
	toRecord := func(m map[string]any) (*neo4j.Record, error) {
		n := len(m)
		rec := &neo4j.Record{
			Keys:   make([]string, n),
			Values: make([]any, n),
		}
		var i int
		for k, v := range m {
			rec.Keys[i] = k
			if _, ok := v.(INode); ok {
				meta, err := s.Codecs().ExtractNeo4jNodeMeta(v)
				if err != nil {
					return nil, err
				}
				props, err := s.Codecs().Encode(v)
				if err != nil {
					return nil, err
				}
				rec.Values[i] = neo4j.Node{
					Labels: meta.Labels,
					Props:  props,
				}
			} else if _, ok := v.(IRelationship); ok {
				meta, err := s.Codecs().ExtractRelationshipMeta(v)
				if err != nil {
					return nil, err
				}
				props, err := s.Codecs().Encode(v)
				if err != nil {
					return nil, err
				}
				rec.Values[i] = neo4j.Relationship{
					Type:  meta.Type,
					Props: props,
				}
			} else {
				rec.Values[i] = v
			}
			i++
		}
		return rec, nil
	}
	if s.Single == nil && s.Records == nil {
		panic(errors.New("mock client used without bindings for all transactions"))
	}
	bindings := s.mockBindings
	if bindings.Single != nil {
		rec, err := toRecord(bindings.Single)
		if err != nil {
			return nil, err
		}
		r.records = []*neo4j.Record{rec}
	} else if bindings.Records != nil {
		r.records = make([]*neo4j.Record, len(bindings.Records))
		for i, recMap := range bindings.Records {
			rec, err := toRecord(recMap)
			if err != nil {
				return nil, err
			}
			r.records[i] = rec
		}
	}
	s.mockBindings = s.Next
	return r, nil
}

func (s *mockNeo4jSessionWithContext) Close(ctx context.Context) error {
	return nil
}

func (t *mockNeo4jManagedTransaction) Run(ctx context.Context, cypher string, params map[string]any) (neo4j.ResultWithContext, error) {
	return t.mockNeo4jSessionWithContext.Run(ctx, cypher, params)
}

func (r *mockNeo4jResultWithContext) Keys() ([]string, error) {
	return r.records[r.cursor].Keys, nil
}

func (r *mockNeo4jResultWithContext) NextRecord(ctx context.Context, record **neo4j.Record) bool {
	if r.cursor < len(r.records) {
		*record = r.records[r.cursor]
		r.cursor++
		return true
	}
	return false
}

func (r *mockNeo4jResultWithContext) Next(ctx context.Context) bool {
	if r.cursor == 0 && !r.started {
		r.started = true
	} else {
		r.cursor++
	}
	return r.cursor < len(r.records)
}

func (r *mockNeo4jResultWithContext) PeekRecord(ctx context.Context, record **neo4j.Record) bool {
	if r.cursor+1 < len(r.records) {
		*record = r.records[r.cursor+1]
		return true
	}
	return false
}

func (r *mockNeo4jResultWithContext) Peek(ctx context.Context) bool {
	return r.cursor+1 < len(r.records)
}

func (r *mockNeo4jResultWithContext) Err() error {
	return nil
}

func (r *mockNeo4jResultWithContext) Record() *neo4j.Record {
	return r.records[r.cursor]
}

func (r *mockNeo4jResultWithContext) Collect(ctx context.Context) ([]*neo4j.Record, error) {
	if r.cursor+1 == len(r.records) {
		return nil, nil
	}
	return r.records[r.cursor+1:], nil
}

func (r *mockNeo4jResultWithContext) Single(ctx context.Context) (*neo4j.Record, error) {
	return r.records[r.cursor], nil
}

func (r *mockNeo4jResultWithContext) Consume(ctx context.Context) (neo4j.ResultSummary, error) {
	panic(errors.New("not implemented"))
}

func (r *mockNeo4jResultWithContext) IsOpen() bool {
	return true
}
