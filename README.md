<h1 align="center">opentdm</h1>

<p align="center">
  <b>Open-source, self-hosted test data &amp; configuration management.</b><br>
  Typed config artifacts (<code>json</code> / <code>csv</code> / <code>xml</code> / <code>.env</code> / <code>properties</code> / secrets)
  per project, per environment, with tags — pulled into your CI and tests via REST, CLI, a GitHub Action, or SDKs.
</p>

---

> **Status: early development (Phase 0/1).** The spine — projects → environments → configs → base/env
> merge → scoped tokens → `opentdm pull` — is being built first. See [`DECISIONS.md`](./DECISIONS.md)
> for the binding architecture and the roadmap below.

## Why opentdm

Teams scatter test data, per-environment config, and secrets across `.env` files, CI variables, and
fixture files committed to repos. Secret managers (Infisical, Doppler) don't store typed *fixtures*;
synthetic-data generators don't manage config or secrets. **opentdm is the middle**: one typed store,
scoped per project per environment, with a GitHub-style UI and a one-call CI primitive.

```bash
# in CI / a test job:
OPENTDM_TOKEN=otdm_xxx opentdm pull --env staging > .env
# or inject directly into a process:
OPENTDM_TOKEN=otdm_xxx opentdm run --env staging -- pytest
```

## Quickstart (self-host)

```bash
git clone https://github.com/opentdm/opentdm && cd opentdm
cp .env.example .env
# generate the master encryption key (KEEP IT SAFE — losing it makes all data unrecoverable):
printf 'OPENTDM_MASTER_KEY=%s\n' "$(openssl rand -base64 32)" >> .env
docker compose up -d
open http://localhost:8080   # first run prints a one-time setup token in `docker compose logs app`
```

## How it works

```
Project ── Environments (development / staging / production)
   └── Configs (a named bundle, tagged, typed)
         ├── variables  → config_items  (base + per-env overrides, merged)
         └── files      → config_blobs  (json/csv/xml fixtures, per-env variants)

resolve(project, env) → base ⊕ env override (last-writer wins) → json | dotenv | shell | yaml | properties
```

## Consumption matrix

| Method | v1 | Use |
|---|----|-----|
| REST API + scoped token | ✅ | the core primitive (`GET /api/v1/projects/{p}/resolve?env=`) |
| CLI `pull` / `run` | ✅ | inject env vars / write `.env` in CI |
| GitHub Action | Phase 2 | `opentdm/inject-action` |
| SDKs (Node/Python/Go) | Phase 3 | fetch resolved values in test code |

## Security model

Server-side **AES-256-GCM envelope encryption** (master key → per-project DEK → per-value ciphertext).
The server **can read plaintext** (by design — it powers editing, search, validation, diff, and format
conversion). This is **not zero-knowledge**: a compromised running server can read secrets. See
[`SECURITY.md`](./SECURITY.md). **Losing `OPENTDM_MASTER_KEY` makes all encrypted data unrecoverable.**

## Roadmap

- **Phase 0** — scaffold, health, docker-compose, CI.
- **Phase 1** — variables end-to-end: auth, projects/envs/configs, envelope crypto, resolve/merge,
  scoped tokens, CLI `pull`/`run`, GitHub-style UI.
- **Phase 2** — file/fixture types, versioning + diff, GitHub Action, user PATs + CLI writes.
- **Phase 3** — SDKs, audit UI, KMS providers, format conversion, docs site.
- **Phase 4** — teams/orgs + RBAC, webhooks, Kubernetes operator.

## License

[MIT](./LICENSE) © opentdm contributors.
