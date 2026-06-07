# Changelog

All notable changes to opentdm are documented here. The format follows
[Keep a Changelog](https://keepachangelog.com/en/1.1.0/), and the project aims to follow
[Semantic Versioning](https://semver.org/spec/v2.0.0.html). The GitHub Release for each tag also carries
auto-generated notes from merged PRs.

## [0.1.0] — 2026-06-07

First self-hostable release: a single Go binary (web UI embedded) + PostgreSQL.

### Added
- **Object-centric UI** — typed, tagged config objects per project, each with a type-appropriate editor:
  a secret-aware key/value table (`env` / `properties` / `secret`), a CodeMirror code editor with Format +
  validate (`json` / `xml`), and a CSV editor with a parsed-table preview.
- **Managed environments** — create / rename / reorder / set-default / delete environment layers; values
  resolve as base ⊕ environment override with reported collisions.
- **Cross-environment cloning** — copy an object's layer, or a whole environment, from one env to another,
  with or without values.
- **Versioning** — per-layer version history with diff and rollback.
- **Per-project authorization** — owner / editor / viewer roles enforced at every endpoint (non-members get
  404); instance admins are implicit owners. Per-user project listing and an admin user directory.
- **Email invitations** — invite by email to a project with a role; the invitee sets their own password on
  accept. SMTP optional — the accept link is logged/returned when SMTP is unconfigured.
- **Audit log** — append-only activity feed of resource mutations (no secret values recorded), viewable per
  project (members) and instance-wide (admins).
- **Auth planes** — session cookies (UI), user PATs (`otdmu_…`), and read-only project+environment service
  tokens (`otdm_…`).
- **Consumption** — REST `resolve`, the `opentdm` CLI (`login` / `pull` / `run`, plus `configs set` /
  `push-file` with a PAT), and a GitHub Action.
- **Crypto** — AES-256-GCM envelope encryption (master key → per-project DEK → per-value ciphertext) with
  AAD bound to immutable identity.

### Security
- See [`SECURITY.md`](./SECURITY.md). Not zero-knowledge by design; back up `OPENTDM_MASTER_KEY` out-of-band.

[0.1.0]: https://github.com/opentdm/opentdm/releases/tag/v0.1.0
