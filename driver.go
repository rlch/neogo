package neogo

import (
	"context"

	"github.com/neo4j/neo4j-go-driver/v5/neo4j"

	"github.com/rlch/neogo/internal"
)

type Driver interface {
	DB() neo4j.DriverWithContext

	ReadSession(ctx context.Context, configurers ...func(*neo4j.SessionConfig)) ReadSession
	WriteSession(ctx context.Context, configurers ...func(*neo4j.SessionConfig)) WriteSession
	Exec(configurers ...func(*ExecConfig)) Client
}

type ExecConfig struct {
	*neo4j.SessionConfig
	*neo4j.TransactionConfig
}

type TxWork func(begin func() Client) error

type ReadSession interface {
	Session() neo4j.SessionWithContext
	Close(ctx context.Context) error
	ReadTx(work TxWork, configurers ...func(*neo4j.TransactionConfig)) error
}

type WriteSession interface {
	ReadSession
	WriteTx(work TxWork, configurers ...func(*neo4j.TransactionConfig)) error
}

type (
	driver struct {
		db neo4j.DriverWithContext
	}
	session struct {
		db         neo4j.DriverWithContext
		execConfig ExecConfig
		session    neo4j.SessionWithContext
		currentTx  neo4j.ManagedTransaction
	}
)

func New(neo4j neo4j.DriverWithContext) Driver {
	return &driver{
		db: neo4j,
	}
}

func (d *driver) DB() neo4j.DriverWithContext {
	return d.db
}

func (d *driver) Exec(configurers ...func(*ExecConfig)) Client {
	config := ExecConfig{}
	for _, c := range configurers {
		c(&config)
	}
	session := &session{
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
		db:      d.db,
		session: sess,
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
		db:      d.db,
		session: sess,
	}
}

func (s *session) Session() neo4j.SessionWithContext {
	return s.session
}

func (s *session) Close(ctx context.Context) error {
	return s.session.Close(ctx)
}

func (s *session) ReadTx(work TxWork, configurers ...func(*neo4j.TransactionConfig)) error {
	config := neo4j.TransactionConfig{}
	for _, c := range configurers {
		c(&config)
	}
	return work(func() Client {
		return s.newClient(internal.NewCypherClient())
	})
}

func (s *session) WriteTx(work TxWork, configurers ...func(*neo4j.TransactionConfig)) error {
	config := neo4j.TransactionConfig{}
	for _, c := range configurers {
		c(&config)
	}
	return work(func() Client {
		return s.newClient(internal.NewCypherClient())
	})
}
