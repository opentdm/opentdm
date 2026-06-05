// Package store is opentdm's data-access layer: a pgx connection pool, an
// embedded migration runner, and hand-written repository methods. Domain and
// HTTP layers depend on the repositories here, never on pgx directly.
package store

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	pgxuuid "github.com/vgarvardt/pgx-google-uuid/v5"
)

// Store owns the database connection pool.
type Store struct {
	pool *pgxpool.Pool
}

// New opens a connection pool for databaseURL and verifies connectivity.
func New(ctx context.Context, databaseURL string) (*Store, error) {
	cfg, err := pgxpool.ParseConfig(databaseURL)
	if err != nil {
		return nil, fmt.Errorf("store: parse DATABASE_URL: %w", err)
	}
	// Register the google/uuid codec so uuid columns scan/encode as uuid.UUID.
	cfg.AfterConnect = func(_ context.Context, conn *pgx.Conn) error {
		pgxuuid.Register(conn.TypeMap())
		return nil
	}
	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		return nil, fmt.Errorf("store: connect: %w", err)
	}
	pingCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	if err := pool.Ping(pingCtx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("store: ping: %w", err)
	}
	return &Store{pool: pool}, nil
}

// Close releases the pool.
func (s *Store) Close() { s.pool.Close() }

// Pool exposes the underlying pool (for migrations and read paths).
func (s *Store) Pool() *pgxpool.Pool { return s.pool }

// Ping checks database connectivity (used by /readyz).
func (s *Store) Ping(ctx context.Context) error { return s.pool.Ping(ctx) }

// Migrate applies pending migrations.
func (s *Store) Migrate(ctx context.Context) error { return Migrate(ctx, s.pool) }

// Q returns pool-bound queries (for single statements / reads).
func (s *Store) Q() *Queries { return &Queries{db: s.pool} }

// InTx runs fn inside a transaction with transaction-bound queries, committing
// on success and rolling back on error or panic.
func (s *Store) InTx(ctx context.Context, fn func(q *Queries) error) (err error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("store: begin: %w", err)
	}
	defer func() {
		if p := recover(); p != nil {
			_ = tx.Rollback(ctx)
			panic(p)
		}
		if err != nil {
			_ = tx.Rollback(ctx)
		}
	}()
	if err = fn(&Queries{db: tx}); err != nil {
		return err
	}
	return tx.Commit(ctx)
}
