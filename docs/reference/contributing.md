# Contributing

The canonical guide is [`CONTRIBUTING.md`](https://github.com/opentdm/opentdm/blob/main/CONTRIBUTING.md). Quick
orientation:

## Layout

A Go workspace (`go.work`) ties `server/`, `apiclient/`, and `cli/`; the React UI lives in `web/` and is built
into the server via `go:embed` (the build output is committed so `go build` works without Node).

## Build, test, format

```bash
go build ./server/... ./cli/... ./apiclient/...
go vet ./server/... && gofmt -l server cli apiclient   # gofmt output must be empty

# unit tests need no setup; the store + httpapi e2e tests are gated on TEST_DATABASE_URL:
docker run -d --name otdm-pg -e POSTGRES_USER=opentdm -e POSTGRES_PASSWORD=opentdm \
  -e POSTGRES_DB=opentdm_test -p 5432:5432 postgres:16-alpine
export TEST_DATABASE_URL="postgres://opentdm:opentdm@localhost:5432/opentdm_test?sslmode=disable"
go test -race ./server/...

# web UI:
cd web && npm install && npm run build     # writes ../server/internal/webui/dist
```

## Conventions

- `gofmt`/`goimports`, small interfaces, wrapped errors, table-driven tests, always `-race`.
- **Conventional Commits** (`feat:` / `fix:` / `refactor:` / `docs:` / `test:` / `chore:`).
- Touching `internal/{crypto,resolve,store,app,httpapi}` or auth? Re-read the invariants in
  [`DECISIONS.md`](https://github.com/opentdm/opentdm/blob/main/DECISIONS.md).
- Security issues: report privately per [`SECURITY.md`](https://github.com/opentdm/opentdm/blob/main/SECURITY.md).

## Docs

This site lives in `docs/` (VitePress) and deploys to GitHub Pages on every push to `main`. Run it locally with
`cd docs && npm install && npm run docs:dev`.
