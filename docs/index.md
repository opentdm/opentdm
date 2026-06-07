---
layout: home

hero:
  name: opentdm
  text: Self-hosted test data & config management
  tagline: Typed config artifacts per project, per environment — edited in a GitHub-style UI and pulled into CI and tests via REST, a CLI, or a GitHub Action.
  actions:
    - theme: brand
      text: Get started
      link: /guide/quickstart
    - theme: alt
      text: Why opentdm
      link: /guide/introduction
    - theme: alt
      text: GitHub
      link: https://github.com/opentdm/opentdm

features:
  - title: Typed, tagged objects
    details: Variables (env / properties / secret) and files (json / csv / xml), each with its own editor — a secret-aware key/value table, a code editor with format + validate, and a CSV table preview.
  - title: Environments & merge
    details: User-managed environment layers; values resolve as base ⊕ environment override (last-writer-wins) with reported collisions. Clone an object or a whole environment across envs.
  - title: Versioned & auditable
    details: Every change is versioned per layer with diff and rollback, and an append-only audit log records who changed/cloned/invited what — without ever recording secret values.
  - title: Roles & invitations
    details: Owner / editor / viewer per project, enforced on every endpoint. Invite teammates by email; SMTP optional (the accept link is logged when unconfigured).
  - title: Consume anywhere
    details: A read-only, project + environment scoped service token plus REST resolve, the opentdm CLI (pull / run), or the GitHub Action.
  - title: Single binary + Postgres
    details: One Go binary with the web UI embedded, AES-256-GCM envelope encryption, and a multi-arch container image. No Redis.
---
