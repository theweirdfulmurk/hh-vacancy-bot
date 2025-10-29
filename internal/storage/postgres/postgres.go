package postgres

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/gocraft/dbr/v2"
	_ "github.com/lib/pq"
	"go.uber.org/zap"
)

type Store struct {
	conn   *dbr.Connection
	sess   *dbr.Session
	logger *zap.Logger
}

func New(dsn string, logger *zap.Logger) (*Store, error) {
	conn, err := dbr.Open("postgres", dsn, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// set up connection pool
	conn.SetMaxOpenConns(25)
	conn.SetMaxIdleConns(5)
	conn.SetConnMaxLifetime(5 * time.Minute)

	// check connection
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := conn.PingContext(ctx); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	sess := conn.NewSession(nil)

	logger.Info("successfully connected to PostgreSQL")

	return &Store{
		conn:   conn,
		sess:   sess,
		logger: logger,
	}, nil
}

func (s *Store) Close() error {
	return s.conn.Close()
}

func (s *Store) Ping(ctx context.Context) error {
	return s.conn.PingContext(ctx)
}

func (s *Store) Session() *dbr.Session {
	return s.sess
}

func (s *Store) BeginTx(ctx context.Context) (*dbr.Tx, error) {
	return s.sess.BeginTx(ctx, &sql.TxOptions{
		Isolation: sql.LevelReadCommitted,
	})
}