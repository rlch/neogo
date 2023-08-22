package neogo

import (
	"context"
	"reflect"

	"github.com/neo4j/neo4j-go-driver/v5/neo4j"

	"github.com/rlch/neogo/client"
	"github.com/rlch/neogo/hooks"
	"github.com/rlch/neogo/internal"
)

// New creates a new neogo [Driver] from a [neo4j.DriverWithContext].
func New(neo4j neo4j.DriverWithContext, configurers ...config) Driver {
	d := driver{db: neo4j, registry: registry{
		hooks: hooks.NewRegistry(),
	}}
	for _, c := range configurers {
		c(&d)
	}
	return &d
}

// Driver represents a pool of connections to a neo4j server or cluster. It
// provides an entrypoint to a neogo [client.Client], which can be used to build
// cypher queries.
//
// It's safe for concurrent use.
type Driver interface {
	// DB returns the underlying neo4j driver.
	DB() neo4j.DriverWithContext

	// ReadSession creates a new read-access session based on the specified session configuration.
	ReadSession(ctx context.Context, configurers ...func(*neo4j.SessionConfig)) readSession

	// WriteSession creates a new write-access session based on the specified session configuration.
	WriteSession(ctx context.Context, configurers ...func(*neo4j.SessionConfig)) writeSession

	// UseHooks registers the given hooks to be used with queries created by the driver.
	UseHooks(hooks ...*hooks.Hook)

	// Exec creates a new transaction + session and executes the given Cypher
	// query.
	//
	// The access mode is inferred from the clauses used in the query. If using
	// Cypher() to inject a write query, one should use [WithSessionConfig] to
	// override the access mode.
	//
	// The session is closed after the query is executed.
	Exec(configurers ...func(*execConfig)) client.Client
}

type config func(*driver)

type execConfig struct {
	*neo4j.SessionConfig
	*neo4j.TransactionConfig
}

type txWork func(begin func() client.Client) error

type readSession interface {
	Session() neo4j.SessionWithContext
	Close(ctx context.Context) error
	ReadTx(ctx context.Context, work txWork, configurers ...func(*neo4j.TransactionConfig)) error
}

type writeSession interface {
	readSession
	WriteTx(ctx context.Context, work txWork, configurers ...func(*neo4j.TransactionConfig)) error
}

type (
	driver struct {
		registry
		db neo4j.DriverWithContext
	}
	session struct {
		registry
		db         neo4j.DriverWithContext
		execConfig execConfig
		session    neo4j.SessionWithContext
		currentTx  neo4j.ManagedTransaction
	}
)

func (d *driver) DB() neo4j.DriverWithContext {
	return d.db
}

func (d *driver) Exec(configurers ...func(*execConfig)) client.Client {
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
		registry:   d.registry,
		db:         d.db,
		execConfig: config,
	}
	return session.newClient(internal.NewCypherClient())
}

func (d *driver) ReadSession(ctx context.Context, configurers ...func(*neo4j.SessionConfig)) readSession {
	config := neo4j.SessionConfig{}
	for _, c := range configurers {
		c(&config)
	}
	config.AccessMode = neo4j.AccessModeRead
	sess := d.db.NewSession(ctx, config)
	return &session{
		registry: d.registry,
		db:       d.db,
		session:  sess,
	}
}

func (d *driver) WriteSession(ctx context.Context, configurers ...func(*neo4j.SessionConfig)) writeSession {
	config := neo4j.SessionConfig{}
	for _, c := range configurers {
		c(&config)
	}
	config.AccessMode = neo4j.AccessModeWrite
	sess := d.db.NewSession(ctx, config)
	return &session{
		registry: d.registry,
		db:       d.db,
		session:  sess,
	}
}

func (s *session) Session() neo4j.SessionWithContext {
	return s.session
}

func (s *session) Close(ctx context.Context) error {
	return s.session.Close(ctx)
}

func (s *session) ReadTx(ctx context.Context, work txWork, configurers ...func(*neo4j.TransactionConfig)) error {
	_, err := s.session.ExecuteRead(ctx, func(tx neo4j.ManagedTransaction) (any, error) {
		return nil, work(func() client.Client {
			c := s.newClient(internal.NewCypherClient())
			c.currentTx = tx
			return c
		})
	}, configurers...)
	return err
}

func (s *session) WriteTx(ctx context.Context, work txWork, configurers ...func(*neo4j.TransactionConfig)) error {
	_, err := s.session.ExecuteWrite(ctx, func(tx neo4j.ManagedTransaction) (any, error) {
		return nil, work(func() client.Client {
			c := s.newClient(internal.NewCypherClient())
			c.currentTx = tx
			return c
		})
	}, configurers...)
	return err
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
