# Contributing to opentdm

Thanks for your interest! opentdm is a Go + React monorepo. Start by reading
[`DECISIONS.md`](./DECISIONS.md) — it is the binding architecture and naming
contract.

## Layout

```
server/      Go API server (Chi + pgx + envelope crypto), embeds the web build
apiclient/   zero-dep Go client (shared by the CLI and Go SDK)
cli/         the `opentdm` CLI (login / pull / run)
web/         React + Vite + @primer/react SPA (built into the server binary)
```

`go.work` ties the Go modules together for local development.

## Dev setup

```bash
# 1. Start Postgres + the server stack
cp .env.example .env
printf 'OPENTDM_MASTER_KEY=%s\n'    "$(openssl rand -base64 32)" >> .env
printf 'OPENTDM_TOKEN_PEPPER=%s\n'  "$(openssl rand -base64 32)" >> .env
printf 'OPENTDM_SESSION_SECRET=%s\n' "$(openssl rand -base64 32)" >> .env
docker compose up -d

# 2. Frontend dev server (proxies /api to :8080) — optional, for UI work
cd web && npm install && npm run dev
```

The first-run **setup token** is printed in the server logs
(`docker compose logs app`); use it to create the first admin.

## Building the web UI

The SPA is embedded into the Go binary via `go:embed`, so the build output
lives at `server/internal/webui/dist` and is **committed** (this lets
`go build`/`go test` run without Node). Regenerate it after UI changes:

```bash
cd web && npm run build   # writes ../server/internal/webui/dist
```

## Database changes

- Migrations live in `server/internal/store/migrations/` as
  `NNNNNN_name.up.sql` / `.down.sql` (golang-migrate, embedded, applied on
  startup under an advisory lock).
- Repository methods are hand-written pgx in `server/internal/store/`.

## Tests

```bash
go test -race -cover ./server/... ./cli/... ./apiclient/...
# integration/e2e tests run when TEST_DATABASE_URL points at a Postgres:
TEST_DATABASE_URL=postgres://opentdm:opentdm@localhost:5432/opentdm_test?sslmode=disable \
  go test -race ./server/...
```

Pure logic (crypto, merge, codecs) is unit-tested; the store and the full HTTP
slice are integration-tested against a real Postgres.

## Conventions

- Go: `gofmt`; small interfaces; wrap errors with context.
- Commits: [Conventional Commits](https://www.conventionalcommits.org)
  (`feat:`, `fix:`, `refactor:`, `docs:`, `test:`, `chore:`).
- Security issues: do **not** open a public issue — see [`SECURITY.md`](./SECURITY.md).
