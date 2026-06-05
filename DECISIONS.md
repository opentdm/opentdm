# opentdm — Binding Design Decisions

> This file is the **single source of truth** that reconciles the architecture. It exists because the
> initial parallel design forked the data model three ways and disagreed on naming. Do not re-litigate
> these; change them only by editing this file in a PR that explains why.

## Locked product decisions
1. **Security model:** server-side **envelope encryption** at rest. The server *can* read plaintext
   (needed for edit/search/validate/diff/format-conversion). **Not zero-knowledge.**
2. **Stack:** Go (Chi + sqlc + golang-migrate + pgx) backend, React + Vite + @primer/react frontend,
   Go CLI (Cobra + GoReleaser), PostgreSQL 16. Monorepo with `go.work`.
3. **Tenancy:** solo/minimal for v1 (Users, Projects, Environments, Configs, API tokens). Nullable
   `org_id` seams left for future teams/RBAC.
4. **Consumption:** REST API + scoped tokens (core), CLI `pull`/`run`, GitHub Action, SDKs.
5. **CLI scope v1:** read-only consumption (login/pull/run/list via service tokens). Writes + user PATs
   in Phase 2.
6. **Variable model:** multiple named variable configs per project; deterministic merge by immutable
   per-config `sort_order`; cross-config collisions surfaced in `meta.collisions`.

## Data model (config = a named bundle, NOT one-row-per-key)
- `config` = named artifact: `kind` (variable|file), `format` (env/properties/secret | json/csv/xml),
  `tags`, immutable `sort_order`.
- Variables → `config_items(config_id, env_id NULL=base, key, value_ciphertext, nonce, dek_version,
  is_secret, deleted)`. `deleted=true` is a tombstone to unset an inherited base key.
- Files → `config_blobs(config_id, env_id NULL=default, content_ciphertext, nonce, dek_version,
  content_hmac, size_bytes)`. Files shadow (no merge); missing variant falls back to base.
- History → append-only `config_versions(config_id, env_id, version, …)`, cut on explicit save.

## Merge semantics (golden-tested before UI)
- Two layers only in v1: base (`env_id IS NULL`) → target env override. No env→env inheritance chains.
- Last-writer-wins per key; cross-config collisions resolved by `sort_order` (never name) + reported.
- `KEY=` is present-empty; absent key is omitted; `deleted=true` removes an inherited key.

## Encryption (reconciled + hardened)
- master key (KEK, `OPENTDM_MASTER_KEY`, base64 32B) → per-project DEK (wrapped on `projects`) →
  per-value/blob ciphertext. KMS providers are interface stubs in v1.
- Versioned wire format `[1B version][nonce][ciphertext][tag]`. v1 = AES-256-GCM with a mandatory
  per-DEK nonce budget forcing DEK rotation below 2^32; v2 = XChaCha20-Poly1305 (24B nonce) available
  behind the version byte as the recommended default.
- **AAD binds immutable identity only**: `project_id ‖ env_id|"base" ‖ config_id ‖ key`. Never put
  `dek_version` in AAD. Bind `master_key_id` into the wrap AAD (blocks KEK downgrade).
- **Keyed** plaintext-equality hash: `HMAC(HKDF(DEK,"hmac-v1"), plaintext)` — never raw sha256.
- Token & session hashes: `HMAC-SHA256(OPENTDM_TOKEN_PEPPER, token)`. Passwords: argon2id.
- Sessions: opaque, revocable DB rows. **No stateless JWT.**

## Tokens
- `otdm_<base62(32B)>`, shown once, stored as HMAC hash + display prefix.
- Scope = one project + ≥1 explicit environments + read|write via `api_token_environments`.
  **Default-deny**: empty/NULL env set is never a wildcard. No cross-project / tag-scoped tokens in v1.
- `last_used_at` updated via a coalescing map flushed by a ticker (never per-request writes).

## Bootstrap
- DB singleton row + one-time setup token printed to first-boot logs, required to create the first admin.

## Canonical names
| Concern | Value |
|---|---|
| Token prefix | `otdm_` |
| Env var prefix | `OPENTDM_` (every var) |
| Response envelope | `{"data":…,"error":null,"meta":{…}}`; errors RFC 9457 problem+json |
| Resolve path | `GET /api/v1/projects/{project}/resolve?env=staging&format=…` |
| Migrations dir | `server/internal/store/migrations/` (embedded) |
| Seeded envs | `development`, `staging`, `production` (ranks 10/20/30) |
| Core env vars | `DATABASE_URL`, `OPENTDM_MASTER_KEY`, `OPENTDM_SESSION_SECRET`, `OPENTDM_TOKEN_PEPPER` |
| Blob cap | `OPENTDM_MAX_BLOB_BYTES` (default 10 MiB) |

## Input hardening
- Validate every resolved key `^[A-Za-z_][A-Za-z0-9_]*$`; reject `BASH_FUNC_*`, `=`, null, newline.
- XML: reject `<!DOCTYPE`, no external entities. JSON: size/depth caps + reject duplicate keys.
- CSV: per-cell/row caps; sanitize formula-injection on export. shell/dotenv renderers single-quote escape.
- GitHub Action: randomized `$GITHUB_ENV` heredoc delimiter; `::add-mask::` first.

## Data access
- Migrations: **golang-migrate**, embedded (`//go:embed`), applied on startup under a Postgres
  advisory lock. Files in `server/internal/store/migrations/` as `NNNNNN_name.up.sql`/`.down.sql`.
- Queries: **pgx/v5 directly** with hand-written repository methods (not sqlc codegen for v1). Rationale:
  avoids a build-time codegen tool + UUID/enum type-mapping friction for contributors, and gives precise
  control over scanning. IDs use `github.com/google/uuid`. sqlc adoption remains possible later behind
  the same `Store` interface.

## Module path
`github.com/opentdm/opentdm` — submodules `server`, `apiclient`, `cli`, `sdks/go`.
