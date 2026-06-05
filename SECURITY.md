# Security Policy

## Threat model (read this before trusting opentdm with secrets)

opentdm uses **server-side envelope encryption**, not zero-knowledge encryption. The server holds the
master key and decrypts data in memory in order to edit, search, validate, diff, and convert it.

- **Protects against:** a stolen database dump, disk, backup, or read replica. Ciphertext columns are
  useless without `OPENTDM_MASTER_KEY`, which lives in the environment (or a KMS), never in the DB.
- **Does NOT protect against:** a compromised running server process — it can read plaintext by design.
  Do not assume secrets are safe from an attacker who has popped the app server.

### Key handling
- `OPENTDM_MASTER_KEY` is a base64-encoded 32-byte key. Generate with `openssl rand -base64 32`.
- It is validated at boot (length-checked) and **never logged**.
- **Losing the master key makes all encrypted data permanently unrecoverable.** Back it up
  out-of-band (a password manager / secret store), separate from database backups.
- `OPENTDM_TOKEN_PEPPER` and `OPENTDM_SESSION_SECRET` must also be backed up; rotating them invalidates
  existing tokens/sessions.

### Backup & restore
- Back up the Postgres volume **and** the three secrets above. A DB backup alone is undecryptable; the
  secrets alone are useless. You need both to restore.

### Encryption details
- Per-project DEK, wrapped by the master key (envelope). Values/blobs encrypted with AES-256-GCM,
  random nonce per operation, AAD bound to immutable row identity. Plaintext-equality hashes are keyed
  HMACs (never raw hashes of secret values). Service/session tokens are stored as `HMAC-SHA256(pepper)`;
  passwords as argon2id.

## Supported versions

Pre-1.0: only the latest tagged release receives security fixes.

## Reporting a vulnerability

**Do not open a public issue for security vulnerabilities.** Report privately via GitHub Security
Advisories ("Report a vulnerability" on the repo's Security tab). We aim to acknowledge within 72 hours.
