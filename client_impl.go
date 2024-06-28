package neogo

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"strings"

	"github.com/goccy/go-json"
	"github.com/neo4j/neo4j-go-driver/v5/neo4j"

	"github.com/rlch/neogo/internal"
	"github.com/rlch/neogo/query"
)

type (
	clientImpl struct {
		*session
		cy *internal.CypherClient
		query.Reader
		query.Updater[query.Querier]
	}
	querierImpl struct {
		*session
		cy *internal.CypherQuerier
		query.Reader
		query.Runner
		query.Updater[query.Querier]
	}
	readerImpl struct {
		*session
		cy *internal.CypherReader
	}
	yielderImpl struct {
		*session
		cy *internal.CypherYielder
		query.Querier
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
		Updater: newUpdater[query.Querier, *internal.CypherQuerier](
			s,
			cy.CypherUpdater,
			func(c *internal.CypherQuerier) query.Querier {
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
		Updater: newUpdater[query.Querier, *internal.CypherQuerier](
			s,
			cy.CypherUpdater,
			func(c *internal.CypherQuerier) query.Querier {
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

func (c *clientImpl) Use(graphExpr string) query.Querier {
	return c.newQuerier(c.cy.Use(graphExpr))
}

func (c *clientImpl) Union(unions ...func(c Query) query.Runner) query.Querier {
	inUnions := make([]func(c *internal.CypherClient) *internal.CypherRunner, len(unions))
	for i, union := range unions {
		union := union
		inUnions[i] = func(cc *internal.CypherClient) *internal.CypherRunner {
			return union(c.newClient(cc)).(baseRunner).GetRunner()
		}
	}
	return c.newQuerier(c.cy.Union(inUnions...))
}

func (c *clientImpl) UnionAll(unions ...func(c Query) query.Runner) query.Querier {
	inUnions := make([]func(c *internal.CypherClient) *internal.CypherRunner, len(unions))
	for i, union := range unions {
		union := union
		inUnions[i] = func(cc *internal.CypherClient) *internal.CypherRunner {
			return union(c.newClient(cc)).(baseRunner).GetRunner()
		}
	}
	return c.newQuerier(c.cy.UnionAll(inUnions...))
}

func (c *readerImpl) OptionalMatch(patterns internal.Patterns) query.Querier {
	return c.newQuerier(c.cy.OptionalMatch(patterns))
}

func (c *readerImpl) Match(patterns internal.Patterns) query.Querier {
	return c.newQuerier(c.cy.Match(patterns))
}

func (c *readerImpl) Subquery(subquery func(c Query) query.Runner) query.Querier {
	inSubquery := func(cc *internal.CypherClient) *internal.CypherRunner {
		runner := subquery(c.newClient(cc))
		return runner.(baseRunner).GetRunner()
	}
	return c.newQuerier(c.cy.Subquery(inSubquery))
}

func (c *readerImpl) With(identifiers ...any) query.Querier {
	return c.newQuerier(c.cy.With(identifiers...))
}

func (c *readerImpl) Unwind(expr any, as string) query.Querier {
	return c.newQuerier(c.cy.Unwind(expr, as))
}

func (c *readerImpl) Call(procedure string) query.Yielder {
	return c.newYielder(c.cy.Call(procedure))
}

func (c *readerImpl) Show(command string) query.Yielder {
	return c.newYielder(c.cy.Show(command))
}

func (c *readerImpl) Return(identifiers ...any) query.Runner {
	return c.newRunner(c.cy.Return(identifiers...))
}

func (c *readerImpl) Cypher(query string) query.Querier {
	q := c.cy.Cypher(query)
	return c.newQuerier(q)
}

func (c *readerImpl) Eval(expression query.Expression) query.Querier {
	q := c.cy.Eval(func(s *internal.Scope, b *strings.Builder) {
		expression.Compile(s, b)
	})
	return c.newQuerier(q)
}

func (c *querierImpl) Where(opts ...internal.WhereOption) query.Querier {
	return c.newQuerier(c.cy.Where(opts...))
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

func (c *updaterImpl[To, ToCypher]) ForEach(identifier, elementsExpr any, do func(c query.Updater[any])) To {
	return c.to(c.cy.ForEach(identifier, elementsExpr, func(cu *internal.CypherUpdater[any]) {
		u := &updaterImpl[any, any]{
			session: c.session,
			cy:      cu,
			to:      func(tc any) any { return nil },
		}
		do(u)
	}))
}

func (c *yielderImpl) Yield(identifiers ...any) query.Querier {
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

func (c *runnerImpl) Print() query.Runner {
	c.cy.Print()
	return c
}

func (c *runnerImpl) RunWithParams(ctx context.Context, params map[string]any) (err error) {
	cy, err := c.cy.CompileWithParams(params)
	if err != nil {
		return fmt.Errorf("cannot compile cypher: %w", err)
	}
	canonicalizedParams, err := canonicalizeParams(cy.Parameters)
	if err != nil {
		return fmt.Errorf("cannot serialize parameters: %w", err)
	}
	return c.executeTransaction(
		ctx, cy,
		func(tx neo4j.ManagedTransaction) (any, error) {
			var result neo4j.ResultWithContext
			result, err = tx.Run(ctx, cy.Cypher, canonicalizedParams)
			if err != nil {
				return nil, fmt.Errorf("cannot run cypher: %w", err)
			}
			return nil, c.unmarshalResult(ctx, cy, result)
		})
}

func (c *runnerImpl) Run(ctx context.Context) error {
	return c.RunWithParams(ctx, nil)
}

func (c *runnerImpl) StreamWithParams(ctx context.Context, params map[string]any, sink func(r query.Result) error) (err error) {
	cy, err := c.cy.CompileWithParams(params)
	if err != nil {
		return fmt.Errorf("cannot compile cypher: %w", err)
	}
	canonicalizedParams, err := canonicalizeParams(cy.Parameters)
	if err != nil {
		return fmt.Errorf("cannot serialize parameters: %w", err)
	}
	return c.executeTransaction(ctx, cy, func(tx neo4j.ManagedTransaction) (any, error) {
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
}

func (c *runnerImpl) Stream(ctx context.Context, sink func(r query.Result) error) (err error) {
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
) error {
	if !result.Next(ctx) {
		return nil
	}
	first := result.Record()

	// If we have more than one record, we know we should be unmarshalling
	// into slices. If we have a single record, we don't necessarily know if
	// we want to marshal into slices or not.
	//
	// Compare the depth of nesting for the bindings and their corresponding
	// record values:
	// - If any of the bindings have a non-zero depth, i.e. non-slice, we
	//   assume single.
	// - If any of the bindings have a different depth than the correpodning
	//   record-values, assume we have multiple records.
	isRecords, err := (func() (bool, error) {
		if result.Peek(ctx) {
			return true, nil
		}
		allSlices := true
		bindingTypes := map[string]reflect.Type{}
		for k, binding := range cy.Bindings {
			typ := binding.Type()
			for typ.Kind() == reflect.Ptr {
				typ = typ.Elem()
			}
			bindingTypes[k] = typ
			if typ.Kind() == reflect.Slice ||
				typ.Kind() == reflect.Array {
				continue
			}
			allSlices = false
		}
		if !allSlices {
			return false, nil
		}
		for k, bindingType := range bindingTypes {
			recordV, ok := first.Get(k)
			if !ok {
				return false, fmt.Errorf("no value associated with key %q", k)
			}
			recordType := reflect.TypeOf(recordV)
			for {
				bindingNext := bindingType.Kind() == reflect.Array || bindingType.Kind() == reflect.Slice
				recordNext := recordType.Kind() == reflect.Array || recordType.Kind() == reflect.Slice
				if bindingNext && recordNext {
					bindingType = bindingType.Elem()
					recordType = recordType.Elem()
					continue
				} else if !bindingNext && !recordNext {
					break
				}
				return true, nil
			}
		}
		return false, nil
	})()
	if err != nil {
		return err
	}
	if isRecords {
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
			if err := s.bindValue(value, to); err != nil {
				return fmt.Errorf(
					"error binding key %s to type %T: %w",
					key, binding.Interface(), err,
				)
			}
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
		if err := s.bindValue(value, binding); err != nil {
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
) (err error) {
	if c.currentTx == nil {
		sess := c.Session()
		sessConfig := neo4j.SessionConfig{
			// We default to read mode and overwrite if:
			//  - the user explicitly requested write mode
			//  - the query is a write query
			AccessMode: neo4j.AccessModeRead,
		}
		if sess == nil {
			if conf := c.execConfig.SessionConfig; conf != nil {
				sessConfig = *conf
			}
			if cy.IsWrite || sessConfig.AccessMode == neo4j.AccessModeWrite {
				sessConfig.AccessMode = neo4j.AccessModeWrite
				sess = c.db.NewSession(ctx, sessConfig)
			} else {
				sessConfig.AccessMode = neo4j.AccessModeRead
				sess = c.db.NewSession(ctx, sessConfig)
			}
			defer func() {
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
			_, err = sess.ExecuteWrite(ctx, exec, config)
		} else {
			_, err = sess.ExecuteRead(ctx, exec, config)
		}
		if err != nil {
			return err
		}
	} else {
		_, err = exec(c.currentTx)
		if err != nil {
			return err
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
