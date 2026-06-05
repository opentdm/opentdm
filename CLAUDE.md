# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

opentdm is an open-source, self-hosted **test data & configuration management** tool: typed config
artifacts (`env`/`properties`/`secret` variables and `json`/`csv`/`xml` fixtures) per project, per
environment, consumed from CI/tests via REST, a CLI, and (later) a GitHub Action + SDKs.

**Read [`DECISIONS.md`](./DECISIONS.md) first** — it is the binding architecture + naming contract that
reconciles the design. When code and a vague memory disagree, `DECISIONS.md` wins.

## Commands

This is a Go multi-module workspace (`go.work` ties `server/`, `apiclient/`, `cli/`) plus a `web/` SPA.
Run Go commands from the repo root so the workspace resolves.

```bash
# Build / vet / format (all modules)
go build ./server/... ./cli/... ./apiclient/...
go vet ./server/... ./cli/... ./apiclient/...
gofmt -l server cli apiclient            # must be empty; gofmt -w to fix

# Test (pure unit tests run with no setup; store + httpapi e2e need a DB)
go test -race -cover ./server/... ./cli/... ./apiclient/...
go test -race -run TestE2E_VerticalSlice ./server/internal/httpapi/...   # a single test

# Integration/e2e tests are SKIPPED unless TEST_DATABASE_URL is set:
docker run -d --name otdm-pg -e POSTGRES_USER=opentdm -e POSTGRES_PASSWORD=opentdm \
  -e POSTGRES_DB=opentdm_test -p 5432:5432 postgres:16-alpine
export TEST_DATABASE_URL="postgres://opentdm:opentdm@localhost:5432/opentdm_test?sslmode=disable"
go test -race ./server/...

# Web UI (embedded into the server binary; build output is committed)
cd web && npm install && npm run build     # writes ../server/internal/webui/dist
cd web && npm run dev                       # dev server :5173, proxies /api -> :8080

# Run the whole stack (app + postgres)
docker compose up -d        # first-run setup token prints in `docker compose logs app`
make help                   # build / test / up / down / gen-key / logs targets
```

The server binary has subcommands: `serve` (default), `gen-key` (print a base64 32-byte key),
`healthcheck` (GETs /readyz, used by the distroless HEALTHCHECK), `version`. `serve` requires
`OPENTDM_MASTER_KEY`, `OPENTDM_TOKEN_PEPPER`, `OPENTDM_SESSION_SECRET`, and `DATABASE_URL`.

## Architecture

### Module layout
- `server/` — the API server. Embeds the web build via `go:embed` → single binary.
- `apiclient/` — zero-dependency Go HTTP client (the resolve contract in one place); shared by the CLI
  and the future Go SDK.
- `cli/` — the `opentdm` CLI (`login`/`pull`/`run`); read-only consumption via service tokens in v1.
- `web/` — React + Vite + `@primer/react` SPA.

### Server layering (dependencies point inward)
`internal/httpapi` (thin handlers + middleware) → `internal/app` (service layer; all business rules) →
`internal/{store, codec, crypto, resolve}`. Handlers never touch SQL or crypto directly — they call one
`app.Service` method. `app` maps domain errors (`ErrUnauthorized/ErrForbidden/ErrConflict/ErrNotFound`,
`*ValidationError`) which `httpapi/handlers.go:writeErr` turns into RFC 9457 problem+json.

### The data model is the crux (see DECISIONS.md §"Data model")
A **config is a named bundle**, not one row per key. `kind` is `variable` or `file`.
- Variables live in `config_items(config_id, env_id, key, value_ciphertext, dek_version, is_secret,
  deleted)`. **`env_id IS NULL` is the base layer**; a set `env_id` is an environment override.
  `deleted=true` is a tombstone that unsets an inherited base key.
- `internal/resolve` is a **pure** engine: base→env override merge, tombstones, and cross-config
  collisions resolved by an immutable per-config `sort_order` (never by name), with collisions reported
  in `meta.collisions`. It is golden-tested before anything HTTP touches it — keep it pure.
- The single most error-prone schema detail: the NULL base layer needs **paired partial unique
  indexes** (`WHERE env_id IS NULL` and `WHERE env_id IS NOT NULL`) because NULL is distinct in a
  multi-column UNIQUE.

### Encryption (`internal/crypto`)
Envelope: master key (KEK, from `OPENTDM_MASTER_KEY`) → per-project DEK (wrapped, stored on `projects`)
→ per-value ciphertext. Wire format is `[1-byte alg][nonce][ciphertext||tag]` (AES-256-GCM default,
XChaCha20 available behind the alg byte). **AAD binds immutable identity only**
(`project ‖ env|"base" ‖ config ‖ key` via `crypto.ItemAAD`) — never include `dek_version` (rotation
mutates it). When you change how an item's `(project, env, config, key)` is computed at write time, you
must change it identically at read time or GCM auth fails closed. `app.cipherFor` unwraps+caches the DEK
per project. Tokens/sessions are hashed with `HMAC-SHA256(pepper)`; passwords with argon2id.

### Auth: two planes
Session (cookie `otdm_session`, opaque revocable DB rows, double-submit CSRF via `otdm_csrf` +
`X-CSRF-Token`) for the UI/management endpoints; project+environment-scoped **service tokens**
(`otdm_…`, Bearer) for `/resolve` consumption. `loadAuth` populates whichever is present; management
routes require a user; `/resolve` accepts either and enforces token scope **default-deny**
(`requestedEnv ∈ token.envIDs` or 403). First-run bootstrap is guarded by a DB singleton + a one-time
setup token printed to logs.

### Store + migrations
pgx/v5 directly (no sqlc) — repository methods in `internal/store/*.go` on a `*Queries` bound to a
`DBTX` (pool or tx); `Store.InTx` provides a tx-bound `*Queries`. Migrations are embedded
`internal/store/migrations/*.up.sql` (golang-migrate-style names), applied on startup under a Postgres
advisory lock by a small custom runner (`migrate.go`). uuid columns scan as `google/uuid` via a codec
registered in `store.New`'s `AfterConnect`.

### Codecs (`internal/codec`)
Parsers (dotenv, properties) and the **injection-safe renderers** (`Render(format, []KV)`) that produce
`/resolve` output. `shell`/`dotenv` rendering is the security-sensitive path (values may be `eval`'d) —
it single-quote/escape-quotes values and validates every key with `ValidKey` (rejects `BASH_FUNC_*`,
`=`, newlines). Heavily tested; preserve the escaping behavior.

## Conventions & gotchas
- **`server/internal/webui/dist` is committed** (the embedded build) so `go build`/`go test` work
  without Node. After UI changes, `cd web && npm run build` (stable filenames, no content hashes).
- Repo follows the user's global Go rules: `gofmt`/`goimports` mandatory, small interfaces, wrap errors
  with `fmt.Errorf("...: %w", err)`, table-driven tests, always `-race`.
- Conventional Commits. Do not commit `.env` (gitignored; holds the master key).
- The merge semantics, the crypto envelope, and the schema are the parts that bite if changed casually —
  re-run the `resolve`, `crypto`, and `TestE2E_VerticalSlice` tests after touching any of them.
