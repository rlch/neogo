package neogo

import (
	"context"
	"errors"
	"fmt"
	"reflect"

	"github.com/neo4j/neo4j-go-driver/v5/neo4j"

	"github.com/rlch/neogo/internal"
	"github.com/rlch/neogo/internal/codec"
)

type (
	// Client is the query interface returned by Driver.Exec().
	Client interface {
		// Cypher sets the Cypher query string and returns a Runner for execution.
		Cypher(query string) Runner
	}

	// Runner executes Cypher queries with bindings.
	Runner interface {
		// Print prints the query to stdout for debugging.
		Print() Runner

		// Run executes the query with bindings.
		// Bindings should be pairs of (name string, target any).
		//
		// Example:
		//   var person Person
		//   var movie Movie
		//   err := d.Exec().
		//       Cypher("MATCH (p:Person)-[:ACTED_IN]->(m:Movie) RETURN p, m").
		//       Run(ctx, "p", &person, "m", &movie)
		Run(ctx context.Context, bindings ...any) error

		// RunWithParams executes the query with parameters and bindings.
		//
		// Example:
		//   var person Person
		//   err := d.Exec().
		//       Cypher("MATCH (p:Person {name: $name}) RETURN p").
		//       RunWithParams(ctx, map[string]any{"name": "Alice"}, "p", &person)
		RunWithParams(ctx context.Context, params map[string]any, bindings ...any) error

		// Stream executes the query and streams results one-by-one.
		// Bindings should be pairs of (name string, target any).
		Stream(ctx context.Context, sink func() error, bindings ...any) error

		// StreamWithParams executes the query with parameters and streams results.
		StreamWithParams(ctx context.Context, params map[string]any, sink func() error, bindings ...any) error
	}

	clientImpl struct {
		*session
		cy *internal.CypherClient
	}

	runnerImpl struct {
		*session
		cy *internal.CypherRunner
	}
)

func (s *session) newClient(cy *internal.CypherClient) *clientImpl {
	return &clientImpl{
		session: s,
		cy:      cy,
	}
}

func (c *clientImpl) Cypher(query string) Runner {
	return &runnerImpl{
		session: c.session,
		cy:      c.cy.Cypher(query),
	}
}

func (r *runnerImpl) Print() Runner {
	r.cy.Print()
	return r
}

func (r *runnerImpl) Run(ctx context.Context, bindings ...any) error {
	return r.RunWithParams(ctx, nil, bindings...)
}

func (r *runnerImpl) RunWithParams(ctx context.Context, params map[string]any, bindings ...any) error {
	cy, err := r.cy.CompileWithParams(params, bindings...)
	if err != nil {
		return fmt.Errorf("cannot compile cypher: %w", err)
	}

	canonicalizedParams, err := canonicalizeParams(r.Registry().Codecs(), cy.Parameters)
	if err != nil {
		return fmt.Errorf("cannot serialize parameters: %w", err)
	}
	if canonicalizedParams != nil {
		canonicalizedParams["__isWrite"] = cy.IsWrite
	}

	_, err = r.executeTransaction(
		ctx, cy,
		func(tx neo4j.ManagedTransaction) (any, error) {
			result, err := tx.Run(ctx, cy.Cypher, canonicalizedParams)
			if err != nil {
				return nil, fmt.Errorf("cannot run cypher: %w", err)
			}
			if err = r.unmarshalResult(ctx, cy, result); err != nil {
				return nil, err
			}
			return nil, nil
		})
	return err
}

func (r *runnerImpl) Stream(ctx context.Context, sink func() error, bindings ...any) error {
	return r.StreamWithParams(ctx, nil, sink, bindings...)
}

func (r *runnerImpl) StreamWithParams(ctx context.Context, params map[string]any, sink func() error, bindings ...any) error {
	cy, err := r.cy.CompileWithParams(params, bindings...)
	if err != nil {
		return fmt.Errorf("cannot compile cypher: %w", err)
	}

	canonicalizedParams, err := canonicalizeParams(r.Registry().Codecs(), cy.Parameters)
	if err != nil {
		return fmt.Errorf("cannot serialize parameters: %w", err)
	}

	_, err = r.executeTransaction(ctx, cy, func(tx neo4j.ManagedTransaction) (any, error) {
		result, err := tx.Run(ctx, cy.Cypher, canonicalizedParams)
		if err != nil {
			return nil, fmt.Errorf("cannot run cypher: %w", err)
		}

		for result.Next(ctx) {
			record := result.Record()
			if record == nil {
				continue
			}
			if err = r.unmarshalRecord(cy, record); err != nil {
				return nil, fmt.Errorf("cannot unmarshal record: %w", err)
			}
			if err = sink(); err != nil {
				return nil, err
			}
		}

		if err = result.Err(); err != nil {
			return nil, fmt.Errorf("result error: %w", err)
		}
		return nil, nil
	})
	return err
}

func (s *session) unmarshalResult(
	ctx context.Context,
	cy *internal.CompiledCypher,
	result neo4j.ResultWithContext,
) (err error) {
	if !result.Next(ctx) {
		return nil
	}
	first := result.Record()
	if result.Peek(ctx) {
		var records []*neo4j.Record
		records, err = result.Collect(ctx)
		if err != nil {
			return fmt.Errorf("cannot collect records: %w", err)
		}
		records = append([]*neo4j.Record{first}, records...)
		if err = s.unmarshalRecords(cy, records); err != nil {
			return fmt.Errorf("cannot unmarshal records: %w", err)
		}
	} else {
		single := result.Record()
		if single == nil {
			return nil
		}
		if err = s.unmarshalRecord(cy, single); err != nil {
			return fmt.Errorf("cannot unmarshal record: %w", err)
		}
	}
	return nil
}

func (s *session) unmarshalRecords(
	cy *internal.CompiledCypher,
	records []*neo4j.Record,
) error {
	n := len(records)
	if n == 0 {
		return nil
	}

	for key, binding := range cy.Bindings {
		values := make([]any, n)
		for i, record := range records {
			value, ok := record.Get(key)
			if !ok {
				return fmt.Errorf("no value associated with key %q", key)
			}
			values[i] = value
		}

		plan := cy.Plans[key]

		if plan != nil && plan.IsSlice && !plan.IsSliceAbstract && plan.Decoder != nil {
			if err := plan.DecodeMultiple(values); err == nil {
				continue
			}
		}

		if err := s.unmarshalRecordsFallback(key, values, binding, plan); err != nil {
			return err
		}
	}

	return nil
}

func (s *session) unmarshalRecordsFallback(key string, values []any, binding any, plan *internal.BindingPlan) error {
	n := len(values)

	bindingV := reflect.ValueOf(binding)
	if bindingV.Kind() != reflect.Ptr {
		return fmt.Errorf("binding for key %q must be a pointer", key)
	}
	sliceV := bindingV.Elem()
	for sliceV.Kind() == reflect.Ptr {
		if sliceV.IsNil() {
			sliceV.Set(reflect.New(sliceV.Type().Elem()))
		}
		sliceV = sliceV.Elem()
	}
	if sliceV.Kind() != reflect.Slice {
		return fmt.Errorf("binding for key %q must be a pointer to slice, got %v", key, sliceV.Kind())
	}

	if plan != nil && plan.SliceAllocator != nil {
		slice := plan.SliceAllocator(n)
		sliceV.Set(reflect.ValueOf(slice))
	} else {
		sliceV.Set(reflect.MakeSlice(sliceV.Type(), n, n))
	}

	for i, value := range values {
		elemV := sliceV.Index(i)

		if value == nil {
			continue
		}

		if elemV.Kind() == reflect.Ptr && elemV.IsNil() {
			if plan != nil && plan.ElemAllocator != nil {
				elemPtr := plan.ElemAllocator()
				elemV.Set(reflect.NewAt(elemV.Type().Elem(), elemPtr))
			} else {
				elemV.Set(reflect.New(elemV.Type().Elem()))
			}
		}

		if elemV.CanAddr() {
			elemV = elemV.Addr()
		}
		if err := s.reg.BindValue(value, elemV.Elem()); err != nil {
			return fmt.Errorf("error binding key %q index %d: %w", key, i, err)
		}
	}

	return nil
}

func (s *session) unmarshalRecord(
	cy *internal.CompiledCypher,
	record *neo4j.Record,
) error {
	for key, binding := range cy.Bindings {
		value, ok := record.Get(key)
		if !ok {
			return fmt.Errorf("no value associated with key %q", key)
		}

		if plan := cy.Plans[key]; plan != nil {
			if !plan.IsSlice && !plan.IsAbstract && plan.Decoder != nil {
				if err := plan.DecodeSingle(value); err == nil {
					continue
				}
			}
		}

		if err := s.reg.Bind(value, binding); err != nil {
			return fmt.Errorf("error binding key %q: %w", key, err)
		}
	}
	return nil
}

func (r *runnerImpl) executeTransaction(
	ctx context.Context,
	cy *internal.CompiledCypher,
	exec neo4j.ManagedTransactionWork,
) (out any, err error) {
	if r.currentTx == nil {
		sess := r.Session()
		sessConfig := neo4j.SessionConfig{
			AccessMode: neo4j.AccessModeRead,
		}
		r.ensureCausalConsistency(ctx, &sessConfig)
		if sess == nil {
			if conf := r.execConfig.SessionConfig; conf != nil {
				sessConfig = *conf
			}
			if cy.IsWrite || sessConfig.AccessMode == neo4j.AccessModeWrite {
				sessConfig.AccessMode = neo4j.AccessModeWrite
			} else {
				sessConfig.AccessMode = neo4j.AccessModeRead
			}
			sess = r.db.NewSession(ctx, sessConfig)
			defer func() {
				if sessConfig.AccessMode == neo4j.AccessModeWrite {
					bookmarks := sess.LastBookmarks()
					if bookmarks == nil || r.causalConsistencyKey == nil {
						return
					}
					key := r.causalConsistencyKey(ctx)
					if cur, ok := causalConsistencyCache[key]; ok {
						causalConsistencyCache[key] = neo4j.CombineBookmarks(cur, bookmarks)
					} else {
						causalConsistencyCache[key] = bookmarks
						go func(key string) {
							<-ctx.Done()
							causalConsistencyCache[key] = nil
						}(key)
					}
				}
				if closeErr := sess.Close(ctx); closeErr != nil {
					err = errors.Join(err, closeErr)
				}
			}()
		}
		config := func(tc *neo4j.TransactionConfig) {
			if conf := r.execConfig.TransactionConfig; conf != nil {
				*tc = *conf
			}
		}
		if cy.IsWrite || sessConfig.AccessMode == neo4j.AccessModeWrite {
			out, err = sess.ExecuteWrite(ctx, exec, config)
		} else {
			out, err = sess.ExecuteRead(ctx, exec, config)
		}
		if err != nil {
			return nil, err
		}
	} else {
		out, err = exec(r.currentTx)
		if err != nil {
			return nil, err
		}
	}
	return
}

func canonicalizeParams(codecs *codec.CodecRegistry, params map[string]any) (map[string]any, error) {
	canon := make(map[string]any, len(params))
	if len(params) == 0 {
		return canon, nil
	}
	for k, v := range params {
		if v == nil {
			canon[k] = nil
			continue
		}
		val, err := codecs.EncodeValue(v)
		if err != nil {
			return nil, fmt.Errorf("cannot encode param %q: %w", k, err)
		}
		canon[k] = val
	}
	return canon, nil
}
