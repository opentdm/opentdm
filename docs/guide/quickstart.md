# Quickstart (self-host)

opentdm is a single Go binary (web UI embedded) plus PostgreSQL. The fastest way to run it is the bundled
`docker compose` stack.

## Run it

```bash
git clone https://github.com/opentdm/opentdm && cd opentdm
cp .env.example .env

# generate the three required secrets:
printf 'OPENTDM_MASTER_KEY=%s\n'     "$(openssl rand -base64 32)" >> .env
printf 'OPENTDM_TOKEN_PEPPER=%s\n'   "$(openssl rand -base64 32)" >> .env
printf 'OPENTDM_SESSION_SECRET=%s\n' "$(openssl rand -base64 32)" >> .env

docker compose up -d
```

::: warning Back up your master key
`OPENTDM_MASTER_KEY` is the root of the encryption envelope. **If you lose it, all encrypted data is
unrecoverable.** Store it (and the other two secrets) out-of-band. See [Security](/guide/security).
:::

## Create the first admin

The first run prints a one-time **setup token** to the logs. Use it to create the first admin:

```bash
docker compose logs app | grep "setup token"
open http://localhost:8080        # paste the token, set the admin username + password
```

Everyone else joins via [email invitation](/guide/access-control#invitations).

## Use the published image

Instead of building, you can run the released multi-arch image directly:

```bash
docker run --rm ghcr.io/opentdm/opentdm:v0.1.0 version
```

The container's `HEALTHCHECK` runs the binary's `healthcheck` subcommand against `/readyz`. Migrations run
automatically on startup (advisory-locked); set `OPENTDM_MIGRATE_ON_START=false` for controlled upgrades.

## Next

- [Configuration](/guide/configuration) — all environment variables (including SMTP for invitations).
- [Access control](/guide/access-control) — roles and invitations.
- [In CI](/guide/ci) — pull resolved config into your pipelines.
