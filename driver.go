// Package neogo provides a Neo4j ORM and query builder for Go.
// It wraps the official Neo4j Go driver with type-safe query construction,
// automatic struct mapping, and support for nodes, relationships, and Cypher queries.
package neogo

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"time"

	"github.com/neo4j/neo4j-go-driver/v5/neo4j"
	"github.com/neo4j/neo4j-go-driver/v5/neo4j/auth"
	"github.com/neo4j/neo4j-go-driver/v5/neo4j/config"
	"golang.org/x/sync/semaphore"

	"github.com/rlch/neogo/builder"
	"github.com/rlch/neogo/internal"
)

// New creates a new neogo [Driver] from a [neo4j.DriverWithContext].
func New(
	target string,
	auth auth.TokenManager,
	configurers ...Configurer,
) (Driver, error) {
	cfg := &Config{
		Config: *defaultConfig(),
	}

	for _, c := range configurers {
		c(cfg)
	}

	neo4j, err := neo4j.NewDriverWithContext(
		target,
		auth,
		func(c *config.Config) { *c = cfg.Config },
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create Neo4J driver: %w", err)
	}

	d := driver{
		db:                   neo4j,
		causalConsistencyKey: cfg.CausalConsistencyKey,
		sessionSemaphore:     semaphore.NewWeighted(int64(cfg.MaxConnectionPoolSize)),
	}

	// Initialize registry
	d.reg = internal.NewRegistry()

	// Register types from config
	if len(cfg.Types) > 0 {
		d.reg.RegisterTypes(cfg.Types...)
	}

	return &d, nil
}

type (
	// Driver represents a pool of connections to a neo4j server or cluster. It
	// provides an entrypoint to a neogo [query.Client], which can be used to build
	// cypher queries.
	//
	// It's safe for concurrent use.
	Driver interface {
		// Registry returns the internal registry used by this driver.
		Registry() *internal.Registry

		// DB returns the underlying neo4j driver.
		DB() neo4j.DriverWithContext

		// Schema returns the schema interface for introspection and migration.
		Schema() Schema

		// ReadSession creates a new read-access session based on the specified session configuration.
		ReadSession(ctx context.Context, configurers ...func(*neo4j.SessionConfig)) readSession

		// WriteSession creates a new write-access session based on the specified session configuration.
		WriteSession(ctx context.Context, configurers ...func(*neo4j.SessionConfig)) writeSession

		// Exec creates a new transaction + session and executes the given Cypher
		// query.
		//
		// The access mode is inferred from the clauses used in the query. If using
		// Cypher() to inject a write query, one should use [WithSessionConfig] to
		// override the access mode.
		//
		// The session is closed after the query is executed.
		Exec(configurers ...func(*execConfig)) Query
	}

	// Expression is an interface for compiling a Cypher expression outside the context of a query.
	Expression = builder.Expression

	// Query is the interface for constructing a Cypher query.
	Query = builder.Builder

	// Work is a function that allows Cypher to be executed within a Transaction.
	Work func(start func() Query) error

	// Transaction represents an explicit transaction that can be committed or rolled back.
	Transaction interface {
		// Run executes a statement on this transaction and returns a result
		// Contexts terminating too early negatively affect connection pooling and degrade the driver performance.
		Run(work Work) error
		// Commit commits the transaction
		// Contexts terminating too early negatively affect connection pooling and degrade the driver performance.
		Commit(ctx context.Context) error
		// Rollback rolls back the transaction
		// Contexts terminating too early negatively affect connection pooling and degrade the driver performance.
		Rollback(ctx context.Context) error
		// Close rolls back the actual transaction if it's not already committed/rolled back
		// and closes all resources associated with this transaction
		// Contexts terminating too early negatively affect connection pooling and degrade the driver performance.
		Close(ctx context.Context, joinedErrors ...error) error
	}

	// Config extends the neo4j config with additional neogo-specific options.
	Config struct {
		config.Config

		CausalConsistencyKey func(context.Context) string
		Types                []any
	}

	// Configurer is a function that configures a neogo Config.
	Configurer func(*Config)

	readSession interface {
		// Session returns the underlying Neo4J session.
		Session() neo4j.SessionWithContext
		// Close closes any open resources and marks this session as unusable.
		// Contexts terminating too early negatively affect connection pooling and degrade the driver performance.
		Close(ctx context.Context, joinedErrors ...error) error
		// ReadTransaction executes the given unit of work in a AccessModeRead transaction with retry logic in place.
		// Contexts terminating too early negatively affect connection pooling and degrade the driver performance.
		ReadTransaction(ctx context.Context, work Work, configurers ...func(*neo4j.TransactionConfig)) error
		BeginTransaction(ctx context.Context, configurers ...func(*neo4j.TransactionConfig)) (Transaction, error)
	}
	writeSession interface {
		readSession
		// ExecuteWrite executes the given unit of work in a AccessModeWrite transaction with retry logic in place.
		// Contexts terminating too early negatively affect connection pooling and degrade the driver performance.
		WriteTransaction(ctx context.Context, work Work, configurers ...func(*neo4j.TransactionConfig)) error
	}
	execConfig struct {
		*neo4j.SessionConfig
		*neo4j.TransactionConfig
	}
)

type (
	driver struct {
		reg                  *internal.Registry
		db                   neo4j.DriverWithContext
		causalConsistencyKey func(ctx context.Context) string
		sessionSemaphore     *semaphore.Weighted
	}
	session struct {
		*driver
		db         neo4j.DriverWithContext
		execConfig execConfig
		session    neo4j.SessionWithContext
		currentTx  neo4j.ManagedTransaction
	}
	transactionImpl struct {
		session *session
		tx      neo4j.ExplicitTransaction
	}
)

// defaultConfig returns default configuration values from the neo4j driver.
// This configuration should be maintained and updated when the neo4j driver is updated.
func defaultConfig() *config.Config {
	return &config.Config{
		MaxTransactionRetryTime:      30 * time.Second,
		MaxConnectionPoolSize:        100,
		MaxConnectionLifetime:        1 * time.Hour,
		ConnectionAcquisitionTimeout: 1 * time.Minute,
		SocketConnectTimeout:         5 * time.Second,
		SocketKeepalive:              true,
		UserAgent:                    neo4j.UserAgent,
		FetchSize:                    neo4j.FetchDefault,
	}
}

var causalConsistencyCache map[string]neo4j.Bookmarks = map[string]neo4j.Bookmarks{}

func WithCausalConsistency(when func(ctx context.Context) string) Configurer {
	return func(c *Config) {
		c.CausalConsistencyKey = when
	}
}

// WithTxConfig configures the transaction used by Exec().
func WithTxConfig(configurers ...func(*neo4j.TransactionConfig)) func(ec *execConfig) {
	return func(ec *execConfig) {
		for _, c := range configurers {
			c(ec.TransactionConfig)
		}
	}
}

// WithSessionConfig configures the session used by Exec().
func WithSessionConfig(configurers ...func(*neo4j.SessionConfig)) func(ec *execConfig) {
	return func(ec *execConfig) {
		for _, c := range configurers {
			c(ec.SessionConfig)
		}
	}
}

// WithTypes is an option for [New] that allows you to register instances of
// [IAbstract], [INode] and [IRelationship] to be used with [neogo].
func WithTypes(types ...any) Configurer {
	return func(c *Config) {
		c.Types = append(c.Types, types...)
	}
}

func (d *driver) Registry() *internal.Registry { return d.reg }

func (d *driver) DB() neo4j.DriverWithContext { return d.db }

func (d *driver) Schema() Schema { return newSchema(d.db, d.reg) }

func (d *driver) Exec(configurers ...func(*execConfig)) Query {
	sessionConfig := neo4j.SessionConfig{}
	txConfig := neo4j.TransactionConfig{}
	config := execConfig{
		SessionConfig:     &sessionConfig,
		TransactionConfig: &txConfig,
	}
	for _, c := range configurers {
		c(&config)
	}
	if reflect.ValueOf(sessionConfig).IsZero() {
		config.SessionConfig = nil
	}
	if reflect.ValueOf(txConfig).IsZero() {
		config.TransactionConfig = nil
	}
	session := &session{
		driver:     d,
		db:         d.db,
		execConfig: config,
	}
	return session.newClient(internal.NewCypherClient(d.reg))
}

func (d *driver) ensureCausalConsistency(ctx context.Context, sc *neo4j.SessionConfig) {
	if d == nil || d.causalConsistencyKey == nil {
		return
	}
	var key string
	if key = d.causalConsistencyKey(ctx); key == "" {
		return
	}
	bookmarks := causalConsistencyCache[key]
	if bookmarks == nil {
		return
	}
	sc.Bookmarks = bookmarks
}

func (d *driver) ReadSession(ctx context.Context, configurers ...func(*neo4j.SessionConfig)) readSession {
	config := neo4j.SessionConfig{}
	for _, c := range configurers {
		c(&config)
	}
	config.AccessMode = neo4j.AccessModeRead
	d.ensureCausalConsistency(ctx, &config)
	if err := d.sessionSemaphore.Acquire(ctx, 1); err != nil {
		panic(fmt.Errorf("failed to acquire session semaphore: %w", err))
	}
	sess := d.db.NewSession(ctx, config)
	return &session{
		driver:  d,
		db:      d.db,
		session: sess,
	}
}

func (d *driver) WriteSession(ctx context.Context, configurers ...func(*neo4j.SessionConfig)) writeSession {
	config := neo4j.SessionConfig{}
	for _, c := range configurers {
		c(&config)
	}
	config.AccessMode = neo4j.AccessModeWrite
	d.ensureCausalConsistency(ctx, &config)
	if err := d.sessionSemaphore.Acquire(ctx, 1); err != nil {
		panic(fmt.Errorf("failed to acquire session semaphore: %w", err))
	}
	sess := d.db.NewSession(ctx, config)
	return &session{
		driver:  d,
		db:      d.db,
		session: sess,
	}
}

func (s *session) Session() neo4j.SessionWithContext {
	return s.session
}

func (s *session) Close(ctx context.Context, errs ...error) error {
	sessErr := s.session.Close(ctx)
	s.sessionSemaphore.Release(1)
	if sessErr != nil {
		errs = append(errs, sessErr)
		return errors.Join(errs...)
	}
	if len(errs) == 0 {
		return nil
	}
	return errors.Join(errs...)
}

func (s *session) ReadTransaction(ctx context.Context, work Work, configurers ...func(*neo4j.TransactionConfig)) error {
	_, err := s.session.ExecuteRead(ctx, func(tx neo4j.ManagedTransaction) (any, error) {
		return nil, work(func() Query {
			c := s.newClient(internal.NewCypherClient(s.reg))
			c.currentTx = tx
			return c
		})
	}, configurers...)
	return err
}

func (s *session) WriteTransaction(ctx context.Context, work Work, configurers ...func(*neo4j.TransactionConfig)) error {
	_, err := s.session.ExecuteWrite(ctx, func(tx neo4j.ManagedTransaction) (any, error) {
		return nil, work(func() Query {
			c := s.newClient(internal.NewCypherClient(s.reg))
			c.currentTx = tx
			return c
		})
	}, configurers...)
	return err
}

func (s *session) BeginTransaction(ctx context.Context, configurers ...func(*neo4j.TransactionConfig)) (Transaction, error) {
	tx, err := s.session.BeginTransaction(ctx, configurers...)
	if err != nil {
		return nil, err
	}
	return &transactionImpl{s, tx}, nil
}

func (t *transactionImpl) Run(work Work) error {
	return work(func() Query {
		c := t.session.newClient(internal.NewCypherClient(t.session.reg))
		c.currentTx = t.tx
		return c
	})
}

func (t *transactionImpl) Commit(ctx context.Context) error {
	return t.tx.Commit(ctx)
}

func (t *transactionImpl) Rollback(ctx context.Context) error {
	return t.tx.Rollback(ctx)
}

func (t *transactionImpl) Close(ctx context.Context, errs ...error) error {
	sessErr := t.tx.Close(ctx)
	if sessErr != nil {
		errs = append(errs, sessErr)
		return errors.Join(errs...)
	}
	if len(errs) == 0 {
		return nil
	}
	return errors.Join(errs...)
}
