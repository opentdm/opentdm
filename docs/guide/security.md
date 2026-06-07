# Security

The full policy lives in [`SECURITY.md`](https://github.com/opentdm/opentdm/blob/main/SECURITY.md); this is the
summary.

## Threat model

opentdm uses **server-side AES-256-GCM envelope encryption** (master key → per-project DEK → per-value
ciphertext). It protects **data at rest** — a stolen database, disk, or backup is ciphertext without the master
key.

It is **not zero-knowledge**: the running server can read plaintext by design (that's what powers editing,
search, validation, diff, and format conversion). **A compromised running server can read secrets.**

## Key handling

- `OPENTDM_MASTER_KEY` is a base64 32-byte key, validated at boot and never logged.
- **Back it up out-of-band.** Losing it makes all encrypted data unrecoverable.
- A backup is only useful with **both** the Postgres data **and** the three secrets
  (`OPENTDM_MASTER_KEY`, `OPENTDM_TOKEN_PEPPER`, `OPENTDM_SESSION_SECRET`).

## How it's encrypted

- Per-project DEK wrapped by the master key; values sealed with AES-256-GCM and a fresh random nonce.
- AAD binds each value to its immutable identity (project ‖ env ‖ config ‖ key) — ciphertext can't be relocated.
- Service/session tokens are stored as `HMAC-SHA256(pepper)`; passwords use argon2id.

## Reporting a vulnerability

Please report privately via GitHub Security Advisories — see
[`SECURITY.md`](https://github.com/opentdm/opentdm/blob/main/SECURITY.md). Do not open a public issue for
security problems.
