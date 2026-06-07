# Configuration

All server configuration is environment variables, validated at boot. The authoritative source is
[`server/internal/config/config.go`](https://github.com/opentdm/opentdm/blob/main/server/internal/config/config.go);
copy [`.env.example`](https://github.com/opentdm/opentdm/blob/main/.env.example) to `.env` to start.

## Required

| Variable | Notes |
|---|---|
| `OPENTDM_MASTER_KEY` | base64 32 bytes (`openssl rand -base64 32`). Envelope KEK. **Losing it makes all data unrecoverable.** |
| `OPENTDM_TOKEN_PEPPER` | base64. HMAC pepper for token/session hashing. |
| `OPENTDM_SESSION_SECRET` | base64. Session secret. |
| `DATABASE_URL` | `postgres://user:pass@host:5432/db?sslmode=…` |

## Server (optional)

| Variable | Default | Notes |
|---|---|---|
| `OPENTDM_BIND` | `0.0.0.0` | bind interface |
| `PORT` | `8080` | listen port |
| `OPENTDM_HOST` | `http://localhost:8080` | externally-visible base URL — used to build **invitation links** |
| `OPENTDM_LOG_LEVEL` | `info` | `debug` / `info` / `warn` / `error` |
| `OPENTDM_LOG_FORMAT` | `json` | `json` or `text` |
| `OPENTDM_MIGRATE_ON_START` | `true` | run embedded migrations on boot (advisory-locked) |
| `OPENTDM_MAX_BLOB_BYTES` | `10485760` | max file/blob size (bytes) |
| `OPENTDM_WEB_DIR` | — | serve the UI from disk instead of the embedded build (dev) |

## SMTP (optional — for invitation emails)

If `OPENTDM_SMTP_HOST` / `OPENTDM_SMTP_FROM` are unset, invitations still work but the **accept link is logged**
(and returned to the inviter) instead of being emailed.

| Variable | Default | Notes |
|---|---|---|
| `OPENTDM_SMTP_HOST` | — | SMTP host |
| `OPENTDM_SMTP_PORT` | `587` | SMTP port |
| `OPENTDM_SMTP_USERNAME` / `OPENTDM_SMTP_PASSWORD` | — | SMTP auth |
| `OPENTDM_SMTP_FROM` | — | `From:` address for invitations |
| `OPENTDM_SMTP_TLS` | `starttls` | `starttls` / `implicit` / `none` |

::: tip Production base URL
Set `OPENTDM_HOST` to your real external URL (e.g. `https://opentdm.example.com`) so invitation accept links
point to the right place.
:::
