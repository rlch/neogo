package neogo

import (
	"context"
	"reflect"

	"github.com/neo4j/neo4j-go-driver/v5/neo4j"

	"github.com/rlch/neogo/internal"
)

func New(neo4j neo4j.DriverWithContext, configurers ...config) Driver {
	d := driver{db: neo4j}
	for _, c := range configurers {
		c(&d)
	}
	return &d
}

type Driver interface {
	DB() neo4j.DriverWithContext

	ReadSession(ctx context.Context, configurers ...func(*neo4j.SessionConfig)) ReadSession
	WriteSession(ctx context.Context, configurers ...func(*neo4j.SessionConfig)) WriteSession
	Exec(configurers ...func(*ExecConfig)) Client
}

type config func(*driver)

type ExecConfig struct {
	*neo4j.SessionConfig
	*neo4j.TransactionConfig
}

type txWork func(begin func() Client) error

type ReadSession interface {
	Session() neo4j.SessionWithContext
	Close(ctx context.Context) error
	ReadTx(ctx context.Context, work txWork, configurers ...func(*neo4j.TransactionConfig)) error
}

type WriteSession interface {
	ReadSession
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
		execConfig ExecConfig
		session    neo4j.SessionWithContext
		currentTx  neo4j.ManagedTransaction
	}
)

func (d *driver) DB() neo4j.DriverWithContext {
	return d.db
}

func (d *driver) Exec(configurers ...func(*ExecConfig)) Client {
	sessionConfig := neo4j.SessionConfig{}
	txConfig := neo4j.TransactionConfig{}
	config := ExecConfig{
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

func (d *driver) ReadSession(ctx context.Context, configurers ...func(*neo4j.SessionConfig)) ReadSession {
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

func (d *driver) WriteSession(ctx context.Context, configurers ...func(*neo4j.SessionConfig)) WriteSession {
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
		return nil, work(func() Client {
			c := s.newClient(internal.NewCypherClient())
			c.currentTx = tx
			return c
		})
	}, configurers...)
	return err
}

func (s *session) WriteTx(ctx context.Context, work txWork, configurers ...func(*neo4j.TransactionConfig)) error {
	_, err := s.session.ExecuteWrite(ctx, func(tx neo4j.ManagedTransaction) (any, error) {
		return nil, work(func() Client {
			c := s.newClient(internal.NewCypherClient())
			c.currentTx = tx
			return c
		})
	}, configurers...)
	return err
}
