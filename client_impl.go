package neogo

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"strings"

	"github.com/goccy/go-json"
	"github.com/neo4j/neo4j-go-driver/v5/neo4j"

	"github.com/rlch/neogo/builder"
	"github.com/rlch/neogo/internal"
)

type (
	clientImpl struct {
		*session
		cy *internal.CypherClient
		builder.Reader
		builder.Updater[builder.Querier]
	}
	querierImpl struct {
		*session
		cy *internal.CypherQuerier
		builder.Reader
		builder.Runner
		builder.Updater[builder.Querier]
	}
	readerImpl struct {
		*session
		cy *internal.CypherReader
	}
	yielderImpl struct {
		*session
		cy *internal.CypherYielder
		builder.Querier
	}
	updaterImpl[To, ToCypher any] struct {
		*session
		cy *internal.CypherUpdater[ToCypher]
		to func(ToCypher) To
	}
	runnerImpl struct {
		*session
		cy *internal.CypherRunner
	}
	resultImpl struct {
		*session
		neo4j.ResultWithContext
		compiled *internal.CompiledCypher
	}

	baseRunner interface {
		GetRunner() *internal.CypherRunner
	}
)

func (s *session) newClient(cy *internal.CypherClient) *clientImpl {
	return &clientImpl{
		session: s,
		cy:      cy,
		Reader:  s.newReader(cy.CypherReader),
		Updater: newUpdater(
			s,
			cy.CypherUpdater,
			func(c *internal.CypherQuerier) builder.Querier {
				return s.newQuerier(c)
			},
		),
	}
}

func (s *session) newQuerier(cy *internal.CypherQuerier) *querierImpl {
	return &querierImpl{
		session: s,
		cy:      cy,
		Reader:  s.newReader(cy.CypherReader),
		Runner:  s.newRunner(cy.CypherRunner),
		Updater: newUpdater(
			s,
			cy.CypherUpdater,
			func(c *internal.CypherQuerier) builder.Querier {
				return s.newQuerier(c)
			},
		),
	}
}

func (s *session) newReader(cy *internal.CypherReader) *readerImpl {
	return &readerImpl{
		session: s,
		cy:      cy,
	}
}

func (s *session) newYielder(cy *internal.CypherYielder) *yielderImpl {
	return &yielderImpl{
		session: s,
		cy:      cy,
		Querier: s.newQuerier(cy.CypherQuerier),
	}
}

func newUpdater[To, ToCypher any](
	s *session,
	cy *internal.CypherUpdater[ToCypher],
	to func(ToCypher) To,
) *updaterImpl[To, ToCypher] {
	return &updaterImpl[To, ToCypher]{
		session: s,
		cy:      cy,
		to:      to,
	}
}

func (s *session) newRunner(cy *internal.CypherRunner) *runnerImpl {
	return &runnerImpl{session: s, cy: cy}
}

func (c *clientImpl) Use(graphExpr string) builder.Querier {
	return c.newQuerier(c.cy.Use(graphExpr))
}

func (c *clientImpl) Union(unions ...func(c Query) builder.Runner) builder.Querier {
	inUnions := make([]func(c *internal.CypherClient) *internal.CypherRunner, len(unions))
	for i, union := range unions {
		union := union
		inUnions[i] = func(cc *internal.CypherClient) *internal.CypherRunner {
			return union(c.newClient(cc)).(baseRunner).GetRunner()
		}
	}
	return c.newQuerier(c.cy.Union(inUnions...))
}

func (c *clientImpl) UnionAll(unions ...func(c Query) builder.Runner) builder.Querier {
	inUnions := make([]func(c *internal.CypherClient) *internal.CypherRunner, len(unions))
	for i, union := range unions {
		union := union
		inUnions[i] = func(cc *internal.CypherClient) *internal.CypherRunner {
			return union(c.newClient(cc)).(baseRunner).GetRunner()
		}
	}
	return c.newQuerier(c.cy.UnionAll(inUnions...))
}

func (c *readerImpl) OptionalMatch(patterns internal.Patterns) builder.Querier {
	return c.newQuerier(c.cy.OptionalMatch(patterns))
}

func (c *readerImpl) Match(patterns internal.Patterns) builder.Querier {
	return c.newQuerier(c.cy.Match(patterns))
}

func (c *readerImpl) Subquery(subquery func(c Query) builder.Runner) builder.Querier {
	inSubquery := func(cc *internal.CypherClient) *internal.CypherRunner {
		runner := subquery(c.newClient(cc))
		return runner.(baseRunner).GetRunner()
	}
	return c.newQuerier(c.cy.Subquery(inSubquery))
}

func (c *readerImpl) With(identifiers ...any) builder.Querier {
	return c.newQuerier(c.cy.With(identifiers...))
}

func (c *readerImpl) Unwind(expr any, as string) builder.Querier {
	return c.newQuerier(c.cy.Unwind(expr, as))
}

func (c *readerImpl) Call(procedure string) builder.Yielder {
	return c.newYielder(c.cy.Call(procedure))
}

func (c *readerImpl) Show(command string) builder.Yielder {
	return c.newYielder(c.cy.Show(command))
}

func (c *readerImpl) Return(identifiers ...any) builder.Runner {
	return c.newRunner(c.cy.Return(identifiers...))
}

func (c *readerImpl) Cypher(query string) builder.Querier {
	q := c.cy.Cypher(query)
	return c.newQuerier(q)
}

func (c *readerImpl) Eval(expression builder.Expression) builder.Querier {
	q := c.cy.Eval(func(s *internal.Scope, b *strings.Builder) {
		expression.Compile(s, b)
	})
	return c.newQuerier(q)
}

func (c *querierImpl) Where(args ...any) builder.Querier {
	return c.newQuerier(c.cy.Where(args...))
}

func (c *updaterImpl[To, ToCypher]) Create(pattern internal.Patterns) To {
	return c.to(c.cy.Create(pattern))
}

func (c *updaterImpl[To, ToCypher]) Merge(pattern internal.Pattern, opts ...internal.MergeOption) To {
	return c.to(c.cy.Merge(pattern, opts...))
}

func (c *updaterImpl[To, ToCypher]) DetachDelete(identifiers ...any) To {
	return c.to(c.cy.DetachDelete(identifiers...))
}

func (c *updaterImpl[To, ToCypher]) Delete(identifiers ...any) To {
	return c.to(c.cy.Delete(identifiers...))
}

func (c *updaterImpl[To, ToCypher]) Set(items ...internal.SetItem) To {
	return c.to(c.cy.Set(items...))
}

func (c *updaterImpl[To, ToCypher]) Remove(items ...internal.RemoveItem) To {
	return c.to(c.cy.Remove(items...))
}

func (c *updaterImpl[To, ToCypher]) ForEach(identifier, elementsExpr any, do func(c builder.Updater[any])) To {
	return c.to(c.cy.ForEach(identifier, elementsExpr, func(cu *internal.CypherUpdater[any]) {
		u := &updaterImpl[any, any]{
			session: c.session,
			cy:      cu,
			to:      func(tc any) any { return nil },
		}
		do(u)
	}))
}

func (c *yielderImpl) Yield(identifiers ...any) builder.Querier {
	return c.newQuerier(c.cy.Yield(identifiers...))
}

func (c *yielderImpl) GetRunner() *internal.CypherRunner {
	return c.cy.CypherRunner
}

func (c *querierImpl) GetRunner() *internal.CypherRunner {
	return c.cy.CypherRunner
}

func (c *runnerImpl) GetRunner() *internal.CypherRunner {
	return c.cy
}

func (c *runnerImpl) Print() builder.Runner {
	c.cy.Print()
	return c
}

func (c *runnerImpl) DebugPrint() builder.Runner {
	c.cy.DebugPrint()
	return c
}

func (c *runnerImpl) run(
	ctx context.Context,
	params map[string]any,
	mapResult func(r neo4j.ResultWithContext) (any, error),
) (out any, err error) {
	cy, err := c.cy.CompileWithParams(params)
	if err != nil {
		return nil, fmt.Errorf("cannot compile cypher: %w", err)
	}
	canonicalizedParams, err := canonicalizeParams(cy.Parameters)
	if err != nil {
		return nil, fmt.Errorf("cannot serialize parameters: %w", err)
	}
	return c.executeTransaction(
		ctx, cy,
		func(tx neo4j.ManagedTransaction) (any, error) {
			var result neo4j.ResultWithContext
			result, err = tx.Run(ctx, cy.Cypher, canonicalizedParams)
			if err != nil {
				return nil, fmt.Errorf("cannot run cypher: %w", err)
			}
			err = c.unmarshalResult(ctx, cy, result)
			if err != nil {
				return nil, err
			}
			if mapResult == nil {
				return nil, nil
			}
			return mapResult(result)
		})
}

func (c *runnerImpl) RunWithParams(ctx context.Context, params map[string]any) (err error) {
	_, err = c.run(ctx, params, nil)
	return
}

func (c *runnerImpl) Run(ctx context.Context) (err error) {
	_, err = c.run(ctx, nil, nil)
	return
}

func (c *runnerImpl) RunSummary(ctx context.Context) (neo4j.ResultSummary, error) {
	return c.RunSummaryWithParams(ctx, nil)
}

func (c *runnerImpl) RunSummaryWithParams(ctx context.Context, params map[string]any) (neo4j.ResultSummary, error) {
	summary, err := c.run(ctx, params, func(r neo4j.ResultWithContext) (any, error) {
		return r.Consume(ctx)
	})
	if err != nil {
		return nil, err
	}
	return summary.(neo4j.ResultSummary), nil
}

func (c *runnerImpl) StreamWithParams(ctx context.Context, params map[string]any, sink func(r builder.Result) error) (err error) {
	cy, err := c.cy.CompileWithParams(params)
	if err != nil {
		return fmt.Errorf("cannot compile cypher: %w", err)
	}
	canonicalizedParams, err := canonicalizeParams(cy.Parameters)
	if err != nil {
		return fmt.Errorf("cannot serialize parameters: %w", err)
	}
	_, err = c.executeTransaction(ctx, cy, func(tx neo4j.ManagedTransaction) (any, error) {
		var result neo4j.ResultWithContext
		result, err = tx.Run(ctx, cy.Cypher, canonicalizedParams)
		if err != nil {
			return nil, fmt.Errorf("cannot run cypher: %w", err)
		}
		err := sink(&resultImpl{
			session:           c.session,
			ResultWithContext: result,
			compiled:          cy,
		})
		if err != nil {
			return nil, fmt.Errorf("cannot sink result: %w", err)
		}
		return nil, nil
	})
	return err
}

func (c *runnerImpl) Stream(ctx context.Context, sink func(r builder.Result) error) (err error) {
	return c.StreamWithParams(ctx, nil, sink)
}

func (c *resultImpl) Peek(ctx context.Context) bool {
	return c.ResultWithContext.Peek(ctx)
}

func (c *resultImpl) Next(ctx context.Context) bool {
	return c.ResultWithContext.Next(ctx)
}

func (c *resultImpl) Err() error {
	return c.ResultWithContext.Err()
}

func (c *resultImpl) Read() error {
	record := c.Record()
	if record == nil {
		return nil
	}
	if err := c.unmarshalRecord(c.compiled, record); err != nil {
		return fmt.Errorf("cannot unmarshal record: %w", err)
	}
	return nil
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
	names := cy.Names()
	for name, query := range cy.Queries {
		rootBinding := cy.Bindings[name]
	}
	return nil
}

func (s *session) unmarshalRecords(
	cy *internal.CompiledCypher,
	records []*neo4j.Record,
) error {
	n := len(records)
	slices := make(map[string]reflect.Value)
	for name, binding := range cy.Bindings {
		for binding.Kind() == reflect.Ptr {
			binding = binding.Elem()
		}
		if binding.Kind() != reflect.Slice {
			return fmt.Errorf("cannot allocate results a non-slice value, name: %q", name)
		}
		binding.Set(reflect.MakeSlice(
			binding.Type(),
			n, n,
		))
		slices[name] = binding
	}
	for i, record := range records {
		for key, binding := range slices {
			value, ok := record.Get(key)
			if !ok {
				return fmt.Errorf("no value associated with key %q", key)
			}
			to := binding.Index(i)
			if to.Kind() == reflect.Ptr {
				to.Set(reflect.New(to.Type().Elem()))
			} else {
				to.Set(reflect.New(to.Type()).Elem())
			}
			if to.CanAddr() {
				to = to.Addr()
			}
			if err := s.reg.BindValue(value, to); err != nil {
				return fmt.Errorf(
					"error binding key %s to type %T: %w",
					key, binding.Interface(), err,
				)
			}
		}
	}
	// names := cy.Names()
	// for name, selection := range cy.Queries {
	// 	root := selection.Alloc
	// 	curAlloc := reflect.ValueOf(root)
	// 	nextNode := []*internal.NodeSelection{selection}
	// 	nextAlloc := []reflect.Value{curAlloc}
	// 	// n, [a, b, c], [[w], [x], [y, z]], [[[1]], [[2]], [[3], [4, 5]]]
	// 	for nextNode != nil {
	// 		rel := selection.Next
	// 		if rel == nil {
	// 			break
	// 		}
	// 		// relBinding :=
	// 		relAllocs := make([]*reflect.Value, len(nextAlloc))
	// 		relAllocs := curAlloc.FieldByName(rel.Field)
	// 		if _, ok := names[nextAlloc]; ok {
	// 			relAlloc.Set(reflect.ValueOf(selection.Alloc))
	// 		} else {
	// 			rel.Allloc.Set(reflect.MakeSlice())
	// 		}
	// 	}
	// }
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
		if err := s.reg.BindValue(value, binding); err != nil {
			return fmt.Errorf(
				"error binding key %q to type %T: %w",
				key, binding.Interface(), err,
			)
		}
	}
	return nil
}

func (c *runnerImpl) executeTransaction(
	ctx context.Context,
	cy *internal.CompiledCypher,
	exec neo4j.ManagedTransactionWork,
) (out any, err error) {
	if c.currentTx == nil {
		sess := c.Session()
		sessConfig := neo4j.SessionConfig{
			// We default to read mode and overwrite if:
			//  - the user explicitly requested write mode
			//  - the query is a write query
			AccessMode: neo4j.AccessModeRead,
		}
		c.ensureCausalConsistency(ctx, &sessConfig)
		if sess == nil {
			if conf := c.execConfig.SessionConfig; conf != nil {
				sessConfig = *conf
			}
			if cy.IsWrite || sessConfig.AccessMode == neo4j.AccessModeWrite {
				sessConfig.AccessMode = neo4j.AccessModeWrite
			} else {
				sessConfig.AccessMode = neo4j.AccessModeRead
			}
			sess = c.db.NewSession(ctx, sessConfig)
			defer func() {
				if sessConfig.AccessMode == neo4j.AccessModeWrite {
					bookmarks := sess.LastBookmarks()
					if bookmarks == nil || c.causalConsistencyKey == nil {
						return
					}
					key := c.causalConsistencyKey(ctx)
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
			if conf := c.execConfig.TransactionConfig; conf != nil {
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
		out, err = exec(c.currentTx)
		if err != nil {
			return nil, err
		}
	}
	return
}

func canonicalizeParams(params map[string]any) (map[string]any, error) {
	canon := make(map[string]any, len(params))
	if len(params) == 0 {
		return canon, nil
	}
	for k, v := range params {
		if v == nil {
			canon[k] = nil
		}
		vv := reflect.ValueOf(v)
		for vv.Kind() == reflect.Ptr {
			vv = vv.Elem()
		}
		switch vv.Kind() {
		case reflect.Slice:
			bytes, err := json.Marshal(v)
			if err != nil {
				return nil, fmt.Errorf("cannot marshal slice: %w", err)
			}
			var js []any
			if err := json.Unmarshal(bytes, &js); err != nil {
				return nil, fmt.Errorf("cannot unmarshal slice: %w", err)
			}
			canon[k] = js
		case reflect.Map, reflect.Struct:
			bytes, err := json.Marshal(v)
			if err != nil {
				return nil, fmt.Errorf("cannot marshal map: %w", err)
			}
			var js any
			if err := json.Unmarshal(bytes, &js); err != nil {
				return nil, fmt.Errorf("cannot unmarshal map: %w", err)
			}
			canon[k] = js
		default:
			canon[k] = v
		}
	}
	return canon, nil
}
