# REST API

All endpoints are under `/api/v1`. Successful responses use the envelope `{"data": …, "error": null, "meta": …}`;
errors are RFC 9457 `application/problem+json` (`type`, `title`, `status`, `detail`, `code`).

## Authentication

| Plane | How |
|---|---|
| Session | `otdm_session` cookie + `X-CSRF-Token` (double-submit) on unsafe methods |
| User PAT | `Authorization: Bearer otdmu_…` (acts as the user; CSRF-exempt) |
| Service token | `Authorization: Bearer otdm_…` (read-only `resolve`; project + env scoped) |

Management routes require a **session or PAT**; `resolve` accepts a **session or a service token**. Non-members
of a project receive **404**; members lacking the required role receive **403**.

## Health

| Method | Path | Notes |
|---|---|---|
| GET | `/healthz` | liveness |
| GET | `/readyz` | readiness (dependency checks) |

## Auth & setup

| Method | Path | Notes |
|---|---|---|
| GET | `/auth/setup` | whether first-run setup is needed |
| POST | `/auth/bootstrap` | create the first admin (setup token) |
| POST | `/auth/login` / `/auth/logout` | session lifecycle |
| GET | `/auth/me` | current user |

## Consumption

| Method | Path | Notes |
|---|---|---|
| GET | `/projects/{project}/resolve?env=&format=` | merged config; session **or** service token. `format`: `dotenv` \| `json` \| `shell` \| `yaml` \| `properties` |

The response also carries an `X-OpenTDM-Collisions` header with the count of cross-config key collisions.
Add `&meta=true` to get the canonical JSON envelope instead of the raw rendered body — `data` is the merged
key/value object and `meta.collisions` lists each collision (`{key, winning_config, losing_config}`). In meta
mode `format` is ignored.

## Projects, environments, configs

| Method | Path | Min role |
|---|---|---|
| GET / POST | `/projects` | (list is per-user) |
| GET | `/projects/{project}` | viewer |
| GET / POST | `/projects/{project}/environments` | viewer / editor |
| POST | `/projects/{project}/environments/reorder` | editor |
| PATCH / DELETE | `/projects/{project}/environments/{environment}` | editor |
| GET / POST | `/projects/{project}/configs` | viewer / editor |
| GET / PATCH / DELETE | `/projects/{project}/configs/{config}` | viewer / editor / editor |

## Values, files, versions, clone

| Method | Path | Min role |
|---|---|---|
| GET / PUT | `/projects/{project}/configs/{config}/items` | viewer / editor |
| GET / PUT | `/projects/{project}/configs/{config}/blob` | viewer / editor |
| GET | `/projects/{project}/configs/{config}/versions[/{version}]` | viewer |
| GET | `/projects/{project}/configs/{config}/diff` | viewer |
| POST | `/projects/{project}/configs/{config}/rollback` | editor |
| POST | `/projects/{project}/configs/{config}/clone` | editor |
| POST | `/projects/{project}/clone-environment` | editor |

## Tokens, members, invitations, audit

| Method | Path | Min role |
|---|---|---|
| GET / POST / DELETE | `/projects/{project}/tokens[/{token}]` | viewer / editor / editor |
| GET / POST | `/projects/{project}/members` | viewer / owner |
| PATCH / DELETE | `/projects/{project}/members/{user}` | owner |
| GET / POST / DELETE | `/projects/{project}/invitations[/{invitation}]` | owner |
| GET | `/projects/{project}/audit` | viewer |

## Public invitation accept

| Method | Path | Notes |
|---|---|---|
| GET | `/invitations/{token}` | invitation details for the accept page |
| POST | `/invitations/{token}/accept` | create account + membership, log in |

## Personal access tokens (session only)

| Method | Path | Notes |
|---|---|---|
| GET / POST / DELETE | `/pats[/{pat}]` | a PAT cannot mint/revoke PATs |

## Instance admin

| Method | Path | Notes |
|---|---|---|
| GET | `/users` / `/audit` | user directory / instance-wide audit |
| PATCH | `/users/{user}` | toggle `is_active` / `is_admin` |
