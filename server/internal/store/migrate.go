package store

import (
	"context"
	"embed"
	"fmt"
	"io/fs"
	"sort"
	"strconv"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

//go:embed migrations/*.sql
var migrationsFS embed.FS

// advisoryLockKey serializes concurrent migration runs across replicas.
const advisoryLockKey int64 = 0x0017_0DDB_0017_0DDB

type migration struct {
	version int
	name    string
	sql     string
}

// Migrate applies all pending up-migrations embedded in the binary, under a
// Postgres advisory lock so concurrent instances don't race. It is idempotent.
func Migrate(ctx context.Context, pool *pgxpool.Pool) error {
	migs, err := loadMigrations()
	if err != nil {
		return err
	}

	conn, err := pool.Acquire(ctx)
	if err != nil {
		return fmt.Errorf("migrate: acquire conn: %w", err)
	}
	defer conn.Release()

	if _, err := conn.Exec(ctx, "SELECT pg_advisory_lock($1)", advisoryLockKey); err != nil {
		return fmt.Errorf("migrate: lock: %w", err)
	}
	defer func() { _, _ = conn.Exec(ctx, "SELECT pg_advisory_unlock($1)", advisoryLockKey) }()

	if _, err := conn.Exec(ctx, `CREATE TABLE IF NOT EXISTS schema_migrations (
		version    bigint PRIMARY KEY,
		applied_at timestamptz NOT NULL DEFAULT now()
	)`); err != nil {
		return fmt.Errorf("migrate: ensure schema_migrations: %w", err)
	}

	applied := map[int]bool{}
	rows, err := conn.Query(ctx, "SELECT version FROM schema_migrations")
	if err != nil {
		return fmt.Errorf("migrate: read applied: %w", err)
	}
	for rows.Next() {
		var v int
		if err := rows.Scan(&v); err != nil {
			rows.Close()
			return err
		}
		applied[v] = true
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return err
	}

	for _, m := range migs {
		if applied[m.version] {
			continue
		}
		if err := applyOne(ctx, conn.Conn(), m); err != nil {
			return fmt.Errorf("migrate: apply %06d_%s: %w", m.version, m.name, err)
		}
	}
	return nil
}

func applyOne(ctx context.Context, conn *pgx.Conn, m migration) error {
	tx, err := conn.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx) //nolint:errcheck // no-op after commit
	if _, err := tx.Exec(ctx, m.sql); err != nil {
		return err
	}
	if _, err := tx.Exec(ctx, "INSERT INTO schema_migrations(version) VALUES ($1)", m.version); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

// loadMigrations reads embedded *.up.sql files, parsing the version from the
// leading NNNNNN_ in each filename, sorted ascending.
func loadMigrations() ([]migration, error) {
	entries, err := fs.Glob(migrationsFS, "migrations/*.up.sql")
	if err != nil {
		return nil, err
	}
	var migs []migration
	for _, path := range entries {
		base := strings.TrimPrefix(path, "migrations/")
		us := strings.IndexByte(base, '_')
		if us < 0 {
			return nil, fmt.Errorf("migrate: bad filename %q", base)
		}
		version, err := strconv.Atoi(base[:us])
		if err != nil {
			return nil, fmt.Errorf("migrate: bad version in %q: %w", base, err)
		}
		body, err := migrationsFS.ReadFile(path)
		if err != nil {
			return nil, err
		}
		name := strings.TrimSuffix(base[us+1:], ".up.sql")
		migs = append(migs, migration{version: version, name: name, sql: string(body)})
	}
	sort.Slice(migs, func(i, j int) bool { return migs[i].version < migs[j].version })
	return migs, nil
}
