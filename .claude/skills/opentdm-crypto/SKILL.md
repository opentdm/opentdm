---
name: opentdm-crypto
description: Conventions and invariants for opentdm's server-side envelope encryption. Use when touching server/internal/crypto, encrypting/decrypting config values or file blobs, adding or changing AAD, version snapshots, DEK handling, or token/password hashing.
---

# opentdm crypto invariants

Envelope encryption: master key (KEK, `OPENTDM_MASTER_KEY`) → per-project DEK (wrapped on `projects`) → per-value/blob ciphertext. Code in `server/internal/crypto`; usage in `server/internal/app`. Break these and data becomes silently undecryptable or insecure — always run the tests at the bottom after changing anything here.

## Cardinal rule: AAD must match at write AND read
Every `Seal` binds Additional Authenticated Data identifying the row; `Open` recomputes it and fails closed on mismatch. If you change how AAD is built at write time, change it identically at read time.
- Variables → `crypto.ItemAAD(projectID, envID, configID, key)` (envID `""` = base layer).
- File blobs → `crypto.BlobAAD(projectID, envID, configID)`.
- Version snapshots → `crypto.VersionAAD(projectID, envID, configID, kind)`. The leading tag (`"item"`/`"blob"`/`"version"`) keeps the three kinds non-interchangeable.
- **Never put `dek_version` (or any mutable field) in AAD** — rotation mutates it; using the wrong DEK already fails GCM auth.
- Base (`""`) and a named env are NOT interchangeable — `baseOr("")` maps to the literal `"base"`.

## DEK handling
- `service.cipherFor(project)` → cached `*DEKCipher`, zeroizes the raw DEK. Use for read/encrypt where you don't need the raw key.
- `service.cipherAndDEK(project)` → cipher **plus** a transient raw DEK copy. Only use when you need `crypto.ContentHMAC` (which needs raw key bytes). `defer zero(dek)` immediately after.
- Wire format: `[1-byte alg][nonce][ciphertext||tag]`. AES-256-GCM default, XChaCha20 behind the alg byte. Nonce is random per `Seal` — never reuse a nonce with a key.

## Hashing (one-way, not encryption)
- Tokens/sessions → `crypto.TokenHash(pepper, raw)` (HMAC-SHA256; pepper from env, not DB). Prefixes: service `otdm_`, user PAT `otdmu_` (mutually non-prefixing — dispatch on prefix).
- Content equality/dedup → `crypto.ContentHMAC(dek, plaintext)` (HKDF-keyed). **Never** a raw `sha256` of secret plaintext (it's a confirmation oracle).
- Passwords → argon2id (`HashPassword`/`VerifyPassword`).

## Never
Log secret values, ciphertext, DEKs, or the master key. Store unkeyed hashes of secrets. Reuse a nonce.

## After changing crypto
```bash
go test -race ./server/internal/crypto/...
TEST_DATABASE_URL=... go test -race ./server/internal/httpapi/...   # encrypt→resolve round-trips
```
