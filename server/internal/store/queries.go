package store

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

// ErrNotFound is returned by repository methods when no row matches.
var ErrNotFound = errors.New("store: not found")

// DBTX is satisfied by both *pgxpool.Pool and pgx.Tx, so repository methods run
// either standalone or inside a transaction.
type DBTX interface {
	Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error)
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
}

// Queries is the repository surface, bound to a DBTX.
type Queries struct {
	db DBTX
}

// mapNoRows converts pgx.ErrNoRows into the package-level ErrNotFound.
func mapNoRows(err error) error {
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrNotFound
	}
	return err
}
