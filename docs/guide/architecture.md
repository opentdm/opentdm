# Architecture

opentdm is a Go multi-module workspace (`server/`, `apiclient/`, `cli/`) plus a React / `@primer/react` SPA
(`web/`) embedded into the server binary via `go:embed`. The only datastore is **PostgreSQL** — sessions are
DB-backed, and the unauthenticated auth endpoints (login, bootstrap, invitation-accept) are rate-limited
**per client IP, in-process** (token bucket, no Redis in v1; tunable via `OPENTDM_AUTH_RATELIMIT_RPM` /
`OPENTDM_AUTH_RATELIMIT_BURST`, set RPM to `0` to disable). Across multiple replicas the limit is
per-replica — front it with a shared limiter for a strict global cap.

## Data model

```
Project ── Environments (e.g. development / staging / production, user-managed)
   └── Configs (a named, tagged, typed "object")
         ├── variables  → config_items  (base + per-env overrides, merged)
         └── files      → config_blobs  (json/csv/xml fixtures, per-env variants)

resolve(project, env) → base ⊕ env override (last-writer-wins) → dotenv | json | shell | yaml | properties
```

- A **config is a named bundle**, not one row per key. `kind` is `variable` or `file`.
- Variables live in `config_items`, where **`env_id IS NULL` is the base layer** and a set `env_id` is an
  environment override; a tombstone unsets an inherited base key.
- The merge engine (`internal/resolve`) is **pure** (no DB, no crypto) and golden-tested: base→env override,
  tombstones, and cross-config collisions resolved by an immutable per-config sort order.
- Every write is **versioned** per layer (`config_versions`), enabling diff and rollback.

## Server layering

Dependencies point inward: `internal/httpapi` (thin handlers + middleware) → `internal/app` (the service layer;
all business rules) → `internal/{store, codec, crypto, resolve}`. Handlers never touch SQL or crypto directly.

## Encryption

Envelope: master key (KEK) → per-project DEK (wrapped, stored on the project) → per-value ciphertext. Wire
format is `[alg][nonce][ciphertext‖tag]` (AES-256-GCM default). See [Security](/guide/security).

## Migrations

Embedded SQL migrations are applied on startup under a Postgres advisory lock by a small custom runner
(idempotent, transactional). Toggle with `OPENTDM_MIGRATE_ON_START`.

The binding design record — the full data-model, merge, crypto, and authorization contracts — is
[`DECISIONS.md`](https://github.com/opentdm/opentdm/blob/main/DECISIONS.md).
