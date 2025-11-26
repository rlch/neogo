package neogo

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"strings"

	"github.com/neo4j/neo4j-go-driver/v5/neo4j"

	"github.com/rlch/neogo/builder"
	"github.com/rlch/neogo/internal"
	"github.com/rlch/neogo/internal/codec"
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
	canonicalizedParams, err := canonicalizeParams(c.Registry().Codecs(), cy.Parameters)
	if err != nil {
		return nil, fmt.Errorf("cannot serialize parameters: %w", err)
	}
	if canonicalizedParams != nil {
		canonicalizedParams["__isWrite"] = cy.IsWrite
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
	canonicalizedParams, err := canonicalizeParams(c.Registry().Codecs(), cy.Parameters)
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
	// names := cy.Names()
	// for name, query := range cy.Queries {
	// 	rootBinding := cy.Bindings[name]
	// }
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

	// For each binding, use pre-compiled plan (always available after Compile())
	for key, binding := range cy.Bindings {
		// Collect values for this key from all records
		values := make([]any, n)
		for i, record := range records {
			value, ok := record.Get(key)
			if !ok {
				return fmt.Errorf("no value associated with key %q", key)
			}
			values[i] = value
		}

		plan := cy.Plans[key]

		// Fast path: use plan's DecodeMultiple for non-abstract slice bindings
		// This avoids all per-record reflection by using pre-compiled decoder
		if plan != nil && plan.IsSlice && !plan.IsSliceAbstract && plan.Decoder != nil {
			if err := plan.DecodeMultiple(values); err == nil {
				continue // Success - next binding
			}
			// On error, fall through to fallback
		}

		// Reflection fallback for special cases:
		// - Abstract node (polymorphic) bindings (need runtime label lookup)
		// - Valuer interface implementations (need interface assertion)
		// - Complex pointer chains
		// - Errors from fast path
		if err := s.unmarshalRecordsFallback(key, values, binding, plan); err != nil {
			return err
		}
	}

	return nil
}

// unmarshalRecordsFallback handles special cases that plan.DecodeMultiple can't:
// - Valuer interface implementations
// - Abstract node (polymorphic) bindings
// - Complex types requiring runtime inspection
func (s *session) unmarshalRecordsFallback(key string, values []any, binding any, plan *internal.BindingPlan) error {
	n := len(values)

	// Use reflect to allocate the slice - once per binding, not per record
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

	// Allocate the slice using plan's allocator if available (Phase 6)
	// This uses pre-compiled allocator closure, avoiding reflect.MakeSlice per call
	if plan != nil && plan.SliceAllocator != nil {
		slice := plan.SliceAllocator(n)
		sliceV.Set(reflect.ValueOf(slice))
	} else {
		// Fallback to reflection-based allocation
		sliceV.Set(reflect.MakeSlice(sliceV.Type(), n, n))
	}

	// Decode each value into the slice
	for i, value := range values {
		elemV := sliceV.Index(i)

		// Handle nil values - leave as zero value
		if value == nil {
			continue
		}

		// Allocate pointer elements using plan's allocator if available
		if elemV.Kind() == reflect.Ptr && elemV.IsNil() {
			if plan != nil && plan.ElemAllocator != nil {
				// Use pre-compiled allocator (avoids reflect.New per element)
				elemPtr := plan.ElemAllocator()
				elemV.Set(reflect.NewAt(elemV.Type().Elem(), elemPtr))
			} else {
				// Fallback to reflection-based allocation
				elemV.Set(reflect.New(elemV.Type().Elem()))
			}
		}

		// Use BindValue which handles Valuer, abstract nodes, etc.
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

		// Try using pre-compiled plan for zero-reflection decode (Phase 6)
		if plan := cy.Plans[key]; plan != nil {
			// Fast path for non-slice, non-abstract bindings with pre-compiled decoder
			if !plan.IsSlice && !plan.IsAbstract && plan.Decoder != nil {
				if err := plan.DecodeSingle(value); err == nil {
					continue // Success - next binding
				}
				// Fall through to reflection path on error
			}
			// Slice bindings in single-record case need special handling
			// (wrapping single value in slice) - fall through to Bind
		}

		// Reflection fallback for special cases:
		// - Abstract node (polymorphic) bindings
		// - Valuer interface implementations
		// - Slice bindings that need single-value wrapping
		// - Complex pointer chains
		if err := s.reg.Bind(value, binding); err != nil {
			return fmt.Errorf("error binding key %q: %w", key, err)
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
