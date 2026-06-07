---
name: opentdm-testing
description: How to build, run, and test opentdm locally — the go.work commands, integration/e2e tests gated on TEST_DATABASE_URL, a throwaway Postgres, the web build, and the dockerized stack. Use when running tests, building, or launching the app.
---

# Building, running, and testing opentdm

Run Go commands from the repo root (`go.work` ties `server/`, `apiclient/`, `cli/`).

## Build / format / unit tests
```bash
go build ./server/... ./cli/... ./apiclient/...
gofmt -l server cli apiclient            # must print nothing
go test -race -cover ./server/... ./cli/... ./apiclient/...
```
Pure packages (`crypto`, `resolve`, `codec`, `cli`) test with no setup.

## Integration + e2e tests (need Postgres)
One-command path: **`make test-e2e`** — stands up a throwaway Postgres, runs the full race+cover suite with
`-p 1`, and tears it down. Use this unless you need a single test.

`store` and `httpapi` tests SKIP unless `TEST_DATABASE_URL` is set. To run manually:
```bash
docker run -d --name otdm-pg -e POSTGRES_USER=opentdm -e POSTGRES_PASSWORD=opentdm \
  -e POSTGRES_DB=opentdm_test -p 5432:5432 postgres:16-alpine
export TEST_DATABASE_URL="postgres://opentdm:opentdm@localhost:5432/opentdm_test?sslmode=disable"
go test -race -p 1 ./server/...                  # -p 1 REQUIRED: store + httpapi e2e share one DB and
                                                 # TRUNCATE it, so package test binaries must not run concurrently
go test -race -run TestE2E_Phase2 ./server/internal/httpapi/...   # a single e2e (one package → -p 1 moot)
docker rm -f otdm-pg                              # cleanup
```
`httpapi/e2e_test.go` and `e2e_phase2_test.go` TRUNCATE and rebuild state each run.

## Web UI
```bash
cd web && npm install && npm run build   # outputs to ../server/internal/webui/dist (committed; go:embed)
cd web && npm run dev                     # :5173, proxies /api -> :8080
cd web && npx tsc --noEmit                # type-check (vite build skips types)
```
After UI changes, rebuild so the embed is current. Don't hand-edit `server/internal/webui/dist`.

## Run the full stack (docker compose)
```bash
cp .env.example .env
for k in OPENTDM_MASTER_KEY OPENTDM_TOKEN_PEPPER OPENTDM_SESSION_SECRET; do
  printf '%s=%s\n' "$k" "$(openssl rand -base64 32)" >> .env; done
docker compose up -d --build
docker compose logs app | grep 'setup token'     # first-run admin token
```
**Gotcha:** if you change `POSTGRES_PASSWORD`, run `docker compose down -v` first — the postgres volume keeps the password it was first created with, so the app gets an auth failure otherwise.

## Run the server binary directly (fast iteration)
```bash
DATABASE_URL=$TEST_DATABASE_URL PORT=8080 \
OPENTDM_MASTER_KEY=$(go run ./server/cmd/opentdm-server gen-key) \
OPENTDM_TOKEN_PEPPER=$(go run ./server/cmd/opentdm-server gen-key) \
OPENTDM_SESSION_SECRET=$(go run ./server/cmd/opentdm-server gen-key) \
  go run ./server/cmd/opentdm-server serve
```
Subcommands: `serve | gen-key | healthcheck | version`.
