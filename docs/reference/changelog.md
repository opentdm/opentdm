# Changelog

The full, maintained changelog is [`CHANGELOG.md`](https://github.com/opentdm/opentdm/blob/main/CHANGELOG.md);
each tag also has auto-generated notes on the
[GitHub Releases](https://github.com/opentdm/opentdm/releases) page.

## v0.1.0

First self-hostable release — a single Go binary (web UI embedded) + PostgreSQL.

- **Object-centric UI** — typed, tagged objects with per-type editors (key/value table, JSON/XML code editor,
  CSV preview).
- **Managed environments** + base ⊕ override merge, and **cross-environment cloning** (with or without values).
- **Versioning** — per-layer history with diff and rollback.
- **Per-project authorization** — owner/editor/viewer roles; non-members get 404; admin user directory.
- **Email invitations** — invite by email with a role (SMTP optional; accept link logged when unconfigured).
- **Audit log** — append-only activity feed, per project and instance-wide, with no secret values recorded.
- **Consumption** — REST `resolve`, the `opentdm` CLI, and a GitHub Action; read-only project+env service
  tokens and user PATs.
- **Crypto** — AES-256-GCM envelope encryption (master key → per-project DEK → per-value ciphertext).
