## Summary

What this change does and why.

## Test plan

- [ ] `go build ./server/... ./cli/... ./apiclient/...` + `go vet` + `gofmt -l` clean
- [ ] `go test -race ./...` (set `TEST_DATABASE_URL` for the store + httpapi e2e tests)
- [ ] `cd web && npx tsc --noEmit && npm run build` (if the web UI changed)
- [ ]

## Checklist

- [ ] Touches `internal/{crypto,resolve,store,app,httpapi}` or auth? Re-read the invariants in `DECISIONS.md`.
- [ ] No secrets, `.env`, or `OPENTDM_MASTER_KEY` committed.
- [ ] Docs / `.env.example` / `CHANGELOG.md` updated if behavior or config changed.
