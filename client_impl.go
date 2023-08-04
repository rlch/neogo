package neogo

import (
	"context"
	"errors"
	"fmt"
	"reflect"

	"github.com/goccy/go-json"

	"github.com/neo4j/neo4j-go-driver/v5/neo4j"

	"github.com/rlch/neogo/internal"
)

type (
	clientImpl struct {
		*session
		cy *internal.CypherClient
		reader
		updater[querier]
	}
	querierImpl struct {
		*session
		cy *internal.CypherQuerier
		reader
		runner
		updater[querier]
	}
	readerImpl struct {
		*session
		cy *internal.CypherReader
	}
	yielderImpl struct {
		*session
		cy *internal.CypherYielder
		querier
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
)

func (s *session) newClient(cy *internal.CypherClient) *clientImpl {
	return &clientImpl{
		session: s,
		cy:      cy,
		reader:  s.newReader(cy.CypherReader),
		updater: newUpdater[querier, *internal.CypherQuerier](
			s,
			cy.CypherUpdater,
			func(c *internal.CypherQuerier) querier {
				return s.newQuerier(c)
			},
		),
	}
}

func (s *session) newQuerier(cy *internal.CypherQuerier) *querierImpl {
	return &querierImpl{
		session: s,
		cy:      cy,
		reader:  s.newReader(cy.CypherReader),
		runner:  s.newRunner(cy.CypherRunner),
		updater: newUpdater[querier, *internal.CypherQuerier](
			s,
			cy.CypherUpdater,
			func(c *internal.CypherQuerier) querier {
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
	return &runnerImpl{cy: cy}
}

func (c *clientImpl) Use(graphExpr string) querier {
	return c.newQuerier(c.cy.Use(graphExpr))
}

func (c *clientImpl) Union(unions ...func(c Client) runner) querier {
	inUnions := make([]func(c *internal.CypherClient) *internal.CypherRunner, len(unions))
	for i, union := range unions {
		inUnions[i] = func(cc *internal.CypherClient) *internal.CypherRunner {
			return union(c.newClient(cc)).(*runnerImpl).cy
		}
	}
	return c.newQuerier(c.cy.Union(inUnions...))
}

func (c *clientImpl) UnionAll(unions ...func(c Client) runner) querier {
	inUnions := make([]func(c *internal.CypherClient) *internal.CypherRunner, len(unions))
	for i, union := range unions {
		inUnions[i] = func(cc *internal.CypherClient) *internal.CypherRunner {
			return union(c.newClient(cc)).(*runnerImpl).cy
		}
	}
	return c.newQuerier(c.cy.UnionAll(inUnions...))
}

func (c *readerImpl) OptionalMatch(patterns internal.Patterns) querier {
	return c.newQuerier(c.cy.OptionalMatch(patterns))
}

func (c *readerImpl) Match(patterns internal.Patterns) querier {
	return c.newQuerier(c.cy.Match(patterns))
}

func (c *readerImpl) Subquery(subquery func(c Client) runner) querier {
	inSubquery := func(cc *internal.CypherClient) *internal.CypherRunner {
		return subquery(c.newClient(cc)).(*runnerImpl).cy
	}
	return c.newQuerier(c.cy.Subquery(inSubquery))
}

func (c *readerImpl) With(variables ...any) querier {
	return c.newQuerier(c.cy.With(variables))
}

func (c *readerImpl) Unwind(expr any, as string) querier {
	return c.newQuerier(c.cy.Unwind(expr, as))
}

func (c *readerImpl) Call(procedure string) yielder {
	return c.newYielder(c.cy.Call(procedure))
}

func (c *readerImpl) Show(command string) yielder {
	return c.newYielder(c.cy.Show(command))
}

func (c *readerImpl) Return(matches ...any) runner {
	return c.newRunner(c.cy.Return(matches))
}

func (c *readerImpl) Cypher(query func(s Scope) string) querier {
	q := c.cy.Cypher(func(scope *internal.Scope) string {
		return query(scope)
	})
	return c.newQuerier(q)
}

func (c *querierImpl) Where(opts ...internal.WhereOption) querier {
	return c.newQuerier(c.cy.Where(opts...))
}

func (c *updaterImpl[To, ToCypher]) Create(pattern internal.Patterns) To {
	return c.to(c.cy.Create(pattern))
}

func (c *updaterImpl[To, ToCypher]) Merge(pattern internal.Pattern, opts ...internal.MergeOption) To {
	return c.to(c.cy.Merge(pattern, opts...))
}

func (c *updaterImpl[To, ToCypher]) DetachDelete(variables ...any) To {
	return c.to(c.cy.DetachDelete(variables))
}

func (c *updaterImpl[To, ToCypher]) Delete(variables ...any) To {
	return c.to(c.cy.Delete(variables))
}

func (c *updaterImpl[To, ToCypher]) Set(items ...internal.SetItem) To {
	return c.to(c.cy.Set(items...))
}

func (c *updaterImpl[To, ToCypher]) Remove(items ...internal.RemoveItem) To {
	return c.to(c.cy.Remove(items...))
}

func (c *updaterImpl[To, ToCypher]) ForEach(entity, elementsExpr any, do func(c updater[any])) To {
	return c.to(c.cy.ForEach(entity, elementsExpr, func(c *internal.CypherUpdater[any]) {
	}))
}

func (c *yielderImpl) Yield(variables ...any) querier {
	return c.newQuerier(c.cy.Yield(variables))
}

func (c *runnerImpl) Run(ctx context.Context) (err error) {
	cy, err := c.cy.Compile()
	if err != nil {
		return err
	}
	params, err := canonicalizeParams(cy.Parameters)
	if err != nil {
		return err
	}
	runTx := func(tx neo4j.ManagedTransaction) (neo4j.ResultWithContext, error) {
		return tx.Run(ctx, cy.Cypher, params)
	}
	var result neo4j.ResultWithContext
	if tx := c.currentTx; tx == nil {
		sess := c.Session()
		if sess == nil {
			config := neo4j.SessionConfig{}
			if conf := c.execConfig.SessionConfig; conf != nil {
				config = *conf
			}
			if cy.IsWrite {
				config.AccessMode = neo4j.AccessModeWrite
				sess = c.db.NewSession(ctx, config)
			} else {
				config.AccessMode = neo4j.AccessModeRead
				sess = c.db.NewSession(ctx, config)
			}
			defer func() {
				if closeErr := sess.Close(ctx); closeErr != nil {
					err = errors.Join(err, closeErr)
				}
			}()
		}
		var resultI any
		config := func(tc *neo4j.TransactionConfig) {
			if conf := c.execConfig.TransactionConfig; conf != nil {
				*tc = *conf
			}
		}
		if cy.IsWrite {
			resultI, err = sess.ExecuteWrite(ctx, func(tx neo4j.ManagedTransaction) (any, error) {
				return runTx(tx)
			}, config)
		} else {
			resultI, err = sess.ExecuteRead(ctx, func(tx neo4j.ManagedTransaction) (any, error) {
				return runTx(tx)
			}, config)
		}
		if err != nil {
			return err
		}
		result = resultI.(neo4j.ResultWithContext)
	} else {
		result, err = runTx(tx)
	}
	if err != nil {
		return err
	}
	if result.Peek(ctx) {
		records, err := result.Collect(ctx)
		if err != nil {
			return err
		}
		n := len(records)
		slices := make(map[string]reflect.Value)
		for name, binding := range cy.Bindings {
			for binding.Kind() == reflect.Ptr {
				binding = binding.Elem()
			}
			if binding.Kind() != reflect.Slice {
				return fmt.Errorf("cannot allocate results a non-slice value, name: %q", name)
			}
			binding.SetLen(n)
			slices[name] = binding
		}
		for i, record := range records {
			for key, binding := range slices {
				value, ok := record.Get(key)
				if !ok {
					return fmt.Errorf("no value associated with key %q", key)
				}
				if err := bindValue(value, binding.Index(i)); err != nil {
					return fmt.Errorf(
						"error binding key %q to type %s: %w",
						key, binding.Type().Elem().Name(), err,
					)
				}
			}
		}
	} else {
		record := result.Record()
		for key, binding := range cy.Bindings {
			value, ok := record.Get(key)
			if !ok {
				return fmt.Errorf("no value associated with key %q", key)
			}
			if err := bindValue(value, binding); err != nil {
				return fmt.Errorf(
					"error binding key %q to type %s: %w",
					key, binding.Type().Name(), err,
				)
			}
		}
	}
	return nil
}

func canonicalizeParams(params map[string]any) (map[string]any, error) {
	canon := make(map[string]any, len(params))
	for k, v := range params {
		vv := reflect.ValueOf(v)
		switch vv.Kind() {
		case reflect.Slice:
			bytes, err := json.Marshal(v)
			if err != nil {
				return nil, err
			}
			var js []any
			if err := json.Unmarshal(bytes, &js); err != nil {
				return nil, err
			}
			canon[k] = js
		case reflect.Map, reflect.Struct:
			bytes, err := json.Marshal(v)
			if err != nil {
				return nil, err
			}
			var js map[string]any
			if err := json.Unmarshal(bytes, &js); err != nil {
				return nil, err
			}
			canon[k] = js
		default:
			canon[k] = v
		}
	}
	return canon, nil
}
