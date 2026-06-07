# opentdm — Binding Design Decisions

> This file is the **single source of truth** that reconciles the architecture. It exists because the
> initial parallel design forked the data model three ways and disagreed on naming. Do not re-litigate
> these; change them only by editing this file in a PR that explains why.

## Locked product decisions
1. **Security model:** server-side **envelope encryption** at rest. The server *can* read plaintext
   (needed for edit/search/validate/diff/format-conversion). **Not zero-knowledge.**
2. **Stack:** Go (Chi + golang-migrate + pgx, hand-written queries — see §Storage) backend, React + Vite
   + @primer/react frontend, Go CLI (stdlib `flag` + GoReleaser), PostgreSQL 16. Monorepo with `go.work`.
3. **Tenancy:** solo/minimal for v1 (Users, Projects, Environments, Configs, API tokens). Nullable
   `org_id` seams left for future teams/RBAC.
4. **Consumption:** REST API + scoped tokens (core), CLI `pull`/`run`, GitHub Action, SDKs.
5. **CLI scope:** read-only consumption (login/pull/run/list via read-only service tokens) **and** config
   writes (configs-set / push-file via a user PAT) — both shipped.
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
- Versioned wire format `[1B version][nonce][ciphertext][tag]`. v1 default = AES-256-GCM with random
  96-bit nonces; a per-DEK nonce/seal budget to force DEK rotation below 2^32 is **planned, not yet
  enforced** (see the roadmap). XChaCha20-Poly1305 (24B nonce) is implemented and selectable behind the
  version byte, slated to become the recommended default.
- **AAD binds immutable identity only**: `project_id ‖ env_id|"base" ‖ config_id ‖ key`. Never put
  `dek_version` in AAD. Bind `master_key_id` into the wrap AAD (blocks KEK downgrade).
- **Keyed** plaintext-equality hash: `HMAC(HKDF(DEK,"hmac-v1"), plaintext)` — never raw sha256.
- Token & session hashes: `HMAC-SHA256(OPENTDM_TOKEN_PEPPER, token)`. Passwords: argon2id.
- Sessions: opaque, revocable DB rows. **No stateless JWT.**

## Tokens
- `otdm_<base62(32B)>`, shown once, stored as HMAC hash + display prefix.
- Scope = one project + ≥1 explicit environments (via `api_token_environments`). The token `scope`
  column allows `read|write`, but v1 ships **read-only** service tokens (write is a reserved seam; the
  CLI's write commands use a user PAT). **Default-deny**: empty/NULL env set is never a wildcard. No
  cross-project / tag-scoped tokens in v1.
- `last_used_at` updated via a coalescing map flushed by a ticker (never per-request writes).

## Authorization (per-project roles + membership)
- Roles per project: **owner > editor > viewer** (`project_members(project_id,user_id,role)`, compared via
  `model.RoleRank`). Viewer reads; editor reads+writes configs/values/envs/clone/tokens; owner also manages
  members + invitations. Instance admins (`users.is_admin`) **bypass membership** as implicit owners everywhere.
- Single choke point: `httpapi.resolveProject` (viewer) / `resolveProjectRole(minRole)` gate every
  `/projects/{project}/*` management route. **Non-member → 404** (existence hidden, GitHub-style);
  **member below the required role → 403**. `ListProjects` is per-user (admins see all); the project DTO
  carries `your_role` for UI gating (server is the authority).
- Auth planes: a **PAT inherits its user's roles** (it authenticates *as* the user); a **service token is its
  own grant** (`/resolve` via token is independent of membership; via session it requires viewer+).
- Keep **≥1 owner**: demoting/removing the last owner → 422.
- Onboarding is by **email invitation** (`project_invitations`, token hashed like other tokens, single-use,
  7-day expiry): an owner invites an email+role; accept creates a *new* account + the membership and logs in.
  Existing accounts are added directly by username/email. SMTP is optional — when unconfigured the accept link
  is logged and returned to the inviter (mirrors the setup-token-to-logs pattern). Backfill on migrate makes
  each existing project's `created_by` its owner.

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
