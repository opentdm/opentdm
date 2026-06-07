# Introduction

**opentdm** is open-source, self-hosted **test data & configuration management**: typed config artifacts
(`.env` / `properties` / `secret` variables and `json` / `csv` / `xml` fixtures) per project, per environment,
with tags — edited in a GitHub-style UI and pulled into CI and tests via REST, a CLI, or a GitHub Action.

## Why opentdm

Teams scatter test data, per-environment config, and secrets across `.env` files, CI variables, and fixture
files committed to repos. Secret managers (Infisical, Doppler) don't store typed *fixtures*; synthetic-data
generators don't manage config or secrets. **opentdm is the middle**: one typed store, scoped per project per
environment, with a GitHub-style UI and a one-call CI primitive.

## Features

- **Typed, tagged objects** — variables (`env` / `properties` / `secret`) and files (`json` / `csv` / `xml`),
  each with a type-appropriate editor: a secret-aware key/value table, a CodeMirror code editor with **Format**
  and validate for JSON/XML, and a CSV editor with a parsed-table preview.
- **Environments + merge** — user-managed environment layers; a value resolves as **base ⊕ environment
  override** (last-writer-wins), with collisions reported.
- **Cross-environment cloning** — copy an object's content, or a whole environment, from one env to another,
  with or without values.
- **Versioning** — every change is versioned per layer, with **diff** and **rollback**.
- **Per-project roles** — **owner / editor / viewer**, enforced on every endpoint; non-members can't see a
  project exists. See [Access control](/guide/access-control).
- **Email invitations** — invite a teammate by email with a role; they set their own password on accept. SMTP
  optional — the accept link is logged/returned when unconfigured.
- **Audit log** — an append-only [activity feed](/guide/audit-log) (who changed/cloned/invited/tokened what),
  with no secret values recorded.
- **Three auth planes** — session cookies (UI), user PATs (`otdmu_…`), and read-only project + environment
  service tokens (`otdm_…`) for CI.

## Next

- [Quickstart](/guide/quickstart) — self-host in a few commands.
- [Configuration](/guide/configuration) — every environment variable.
- [In CI](/guide/ci) — pull resolved config with the CLI or the GitHub Action.
