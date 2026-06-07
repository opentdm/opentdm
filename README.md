<h1 align="center">opentdm</h1>

<p align="center">
  <b>Open-source, self-hosted test data &amp; configuration management.</b><br>
  Typed config artifacts (<code>.env</code> / <code>properties</code> / secrets and <code>json</code> / <code>csv</code> / <code>xml</code> fixtures)
  per project, per environment, with tags — edited in a GitHub-style UI and pulled into CI and tests via REST, a CLI, or a GitHub Action.
</p>

---

> **Status: v0.1.0 — self-hostable.** A single Go binary (web UI embedded) + PostgreSQL. The full v1 spine is
> in place: typed objects, base→env merge, versioning, per-project roles, email invitations, an audit log,
> scoped tokens, and `opentdm pull`. See [`DECISIONS.md`](./DECISIONS.md) for the binding architecture and
> [`SECURITY.md`](./SECURITY.md) for the security model.

**📖 Documentation:** <https://opentdm.github.io/opentdm>

## Why opentdm

Teams scatter test data, per-environment config, and secrets across `.env` files, CI variables, and fixture
files committed to repos. Secret managers (Infisical, Doppler) don't store typed *fixtures*; synthetic-data
generators don't manage config or secrets. **opentdm is the middle**: one typed store, scoped per project per
environment, with a GitHub-style UI and a one-call CI primitive.

```bash
# in CI / a test job — inject a resolved environment into a process:
OPENTDM_TOKEN=otdm_xxx opentdm run --env staging -- pytest
# …or write a dotenv file:
OPENTDM_TOKEN=otdm_xxx opentdm pull --env staging -o .env
```

## Features

- **Typed, tagged objects** per project — variables (`env` / `properties` / `secret`) and files (`json` /
  `csv` / `xml`), each with **its own editor**: a key/value table (secret-aware), a CodeMirror code editor
  with Format + validate for JSON/XML, and a CSV editor with a parsed-table preview.
- **Environments + merge** — user-managed environment layers; a value resolves as **base ⊕ environment
  override** (last-writer-wins), reported with collisions.
- **Cross-environment cloning** — copy an object's content (or a whole environment) from one env to another,
  with or without values.
- **Versioning** — every change is versioned per layer, with **diff** and **rollback**.
- **Per-project roles** — **owner / editor / viewer**, enforced on every endpoint; non-members can't see a
  project exists. Instance admins are implicit owners everywhere.
- **Email invitations** — invite a teammate by email to a project with a role; they set their own password on
  accept. SMTP optional — when unconfigured, the accept link is logged and shown to the inviter.
- **Audit log** — an append-only activity feed (who changed/cloned/invited/tokened what), per project and
  instance-wide, with no secret values recorded.
- **Three auth planes** — session cookies (UI), **user PATs** (`otdmu_…`, act as the user for CLI/management
  writes), and **service tokens** (`otdm_…`, read-only, project + environment scoped) for CI consumption.
- **Consume anywhere** — REST (`GET /api/v1/projects/{p}/resolve?env=`), the `opentdm` CLI, or the GitHub
  Action.

## Quickstart (self-host)

```bash
git clone https://github.com/opentdm/opentdm && cd opentdm
cp .env.example .env
# generate the three required secrets (KEEP THE MASTER KEY SAFE — losing it makes data unrecoverable):
printf 'OPENTDM_MASTER_KEY=%s\n'    "$(openssl rand -base64 32)" >> .env
printf 'OPENTDM_TOKEN_PEPPER=%s\n'  "$(openssl rand -base64 32)" >> .env
printf 'OPENTDM_SESSION_SECRET=%s\n' "$(openssl rand -base64 32)" >> .env
docker compose up -d
docker compose logs app | grep "setup token"   # one-time first-run admin token
open http://localhost:8080                       # create the first admin with that token
```

The published image is `ghcr.io/opentdm/opentdm` (multi-arch). The container's `HEALTHCHECK` runs the binary's
`healthcheck` subcommand against `/readyz`.

## Use it in CI

Mint a **read-only service token** (project + environment scoped) in the project's Settings, store it as a CI
secret, then pull resolved config.

**CLI** ([install](#cli-install)):

```bash
opentdm pull --host https://opentdm.example.com --token "$OPENTDM_TOKEN" --project payments --env staging -o .env
opentdm run  --env staging -- npm test     # resolves + injects as env vars for the child process
```

**GitHub Action:**

```yaml
- uses: opentdm/opentdm/action@v0.1.0
  with:
    host: https://opentdm.example.com
    token: ${{ secrets.OPENTDM_TOKEN }}
    project: payments
    environment: staging
    format: env          # env (inject) | dotenv-file | json
```

<a id="cli-install"></a>**CLI install:** `curl -fsSL https://raw.githubusercontent.com/opentdm/opentdm/main/install.sh | bash`
(downloads the `opentdm` binary from GitHub Releases). The CLI surface: `login`, `pull`, `run`, and — with a
user PAT — `configs set` / `push-file`. Auth precedence: flags > `OPENTDM_HOST`/`OPENTDM_TOKEN`/`OPENTDM_PROJECT`
> `~/.opentdm/config.json`.

## Configuration

All server config is environment variables (see [`.env.example`](./.env.example)). The authoritative source is
[`server/internal/config/config.go`](./server/internal/config/config.go).

| Variable | Required | Default | Notes |
|---|---|---|---|
| `OPENTDM_MASTER_KEY` | ✅ | — | base64 32 bytes (`openssl rand -base64 32`). Envelope KEK. **Losing it makes all data unrecoverable.** |
| `OPENTDM_TOKEN_PEPPER` | ✅ | — | base64. HMAC pepper for token/session hashing. |
| `OPENTDM_SESSION_SECRET` | ✅ | — | base64. Session secret. |
| `DATABASE_URL` | ✅ | — | `postgres://user:pass@host:5432/db?sslmode=…` |
| `OPENTDM_BIND` | | `0.0.0.0` | bind interface |
| `PORT` | | `8080` | listen port |
| `OPENTDM_HOST` | | `http://localhost:8080` | externally-visible base URL — used to build **invitation links** |
| `OPENTDM_LOG_LEVEL` | | `info` | `debug`/`info`/`warn`/`error` |
| `OPENTDM_LOG_FORMAT` | | `json` | `json` or `text` |
| `OPENTDM_MIGRATE_ON_START` | | `true` | run embedded migrations on boot (advisory-locked) |
| `OPENTDM_MAX_BLOB_BYTES` | | `10485760` | max file/blob size |
| `OPENTDM_WEB_DIR` | | — | serve the UI from disk instead of the embedded build (dev) |
| `OPENTDM_SMTP_HOST` | | — | SMTP host. **If unset, invitation accept links are logged**, not emailed. |
| `OPENTDM_SMTP_PORT` | | `587` | SMTP port |
| `OPENTDM_SMTP_USERNAME` / `OPENTDM_SMTP_PASSWORD` | | — | SMTP auth |
| `OPENTDM_SMTP_FROM` | | — | `From:` address for invitations |
| `OPENTDM_SMTP_TLS` | | `starttls` | `starttls` / `implicit` / `none` |

The first admin is created via a one-time **setup token** printed to the logs on first boot; everyone else
joins via **email invitation**. See [`DECISIONS.md`](./DECISIONS.md) for the role model and invite flow.

## How it works

```
Project ── Environments (e.g. development / staging / production, user-managed)
   └── Configs (a named, tagged, typed "object")
         ├── variables  → config_items  (base + per-env overrides, merged)
         └── files      → config_blobs  (json/csv/xml fixtures, per-env variants)

resolve(project, env) → base ⊕ env override (last-writer-wins) → dotenv | json | shell | yaml | properties
```

## Security

Server-side **AES-256-GCM envelope encryption** (master key → per-project DEK → per-value ciphertext). The
server **can read plaintext** by design — it powers editing, search, validation, diff, and format conversion.
This is **not zero-knowledge**: a compromised running server can read secrets. See [`SECURITY.md`](./SECURITY.md).
**Back up `OPENTDM_MASTER_KEY` out-of-band — losing it makes all encrypted data unrecoverable.**

## Architecture

A Go multi-module workspace (`server/`, `apiclient/`, `cli/`) plus a React/`@primer/react` SPA (`web/`)
embedded into the server binary via `go:embed`. PostgreSQL only (sessions are DB-backed; no Redis in v1).
The binding design record is [`DECISIONS.md`](./DECISIONS.md).

## Roadmap

**Shipped (v0.1.0):** typed objects + per-type editors, environments + merge, cross-env cloning, versioning +
diff + rollback, per-project roles, email invitations, audit log, scoped tokens + user PATs, CLI, GitHub Action.

**Next:** SSO / OIDC, Node/Python/Go SDKs, external KMS (AWS/GCP) DEK providers, DEK rotation tooling, webhooks.

## Contributing

See [`CONTRIBUTING.md`](./CONTRIBUTING.md) for the dev setup, the multi-module test flow (the store + httpapi
e2e tests are gated on `TEST_DATABASE_URL`), and conventions. Releases: [`RELEASING.md`](./RELEASING.md).

## License

[MIT](./LICENSE) © opentdm contributors.
