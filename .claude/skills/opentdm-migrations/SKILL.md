---
name: opentdm-migrations
description: How to change opentdm's PostgreSQL schema and write data-access code — golang-migrate migrations plus hand-written pgx repositories (no sqlc). Use when adding tables/columns/indexes, writing store queries, or touching server/internal/store or server/internal/model.
---

# opentdm data layer

Schema = golang-migrate (embedded) + hand-written pgx repositories. **No sqlc, no ORM.**

## Add a migration
- Create `server/internal/store/migrations/NNNNNN_name.up.sql` and `.down.sql` (6-digit, next after the highest existing number).
- The embedded, advisory-locked runner (`store/migrate.go`) applies pending migrations in order on startup — no runner changes needed.
- Reuse the shared `set_updated_at` trigger for any `updated_at` column.

## The NULL-base gotcha (most common bug)
A layer's *base* is `env_id IS NULL`. Postgres treats NULL as **distinct** in a multi-column UNIQUE, so enforce base uniqueness with **paired partial indexes**:
```sql
CREATE UNIQUE INDEX uq_x_base ON t(config_id)         WHERE env_id IS NULL;
CREATE UNIQUE INDEX uq_x_env  ON t(config_id, env_id) WHERE env_id IS NOT NULL;
```
In WHERE clauses, `env_id = $2` never matches NULL — use the `layerPred` pattern from `store/versions.go`:
```sql
config_id = $1 AND (env_id = $2 OR ($2::uuid IS NULL AND env_id IS NULL))
```
For "exactly one current per layer" add partial-unique-on-flag indexes (see `config_versions`).

## Repository pattern (mirror items.go / tokens.go / versions.go)
- Methods hang off `*Queries`, which wraps a `DBTX` (satisfied by `*pgxpool.Pool` and `pgx.Tx`).
- `Store.Q()` for single statements/reads; `Store.InTx(func(q *Queries) error)` for atomic multi-statement writes.
- Each file: a `const xCols` column list + a local `scannable` interface + a `scanX(row)` helper.
- UUIDs scan as `github.com/google/uuid.UUID` (codec registered in `store.New`'s `AfterConnect`). Nullable columns → pointer fields. Nullable `description` → `COALESCE(description,'')` in SELECT.
- Pass `*uuid.UUID` for the env arg (nil = base).
- In the service, map `isUniqueViolation(err)` → `ErrConflict`.
- Add domain structs to `server/internal/model/model.go`.

## Verify
```bash
go build ./server/internal/store/...
# start a throwaway Postgres (see the opentdm-testing skill), then:
TEST_DATABASE_URL=... go test -race ./server/internal/store/... ./server/internal/httpapi/...
```
