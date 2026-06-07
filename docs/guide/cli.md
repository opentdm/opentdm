# CLI

The `opentdm` CLI pulls resolved config into CI and tests, and (with a user PAT) writes config.

## Install

```bash
curl -fsSL https://raw.githubusercontent.com/opentdm/opentdm/main/install.sh | bash
# pin a version / dir:
curl -fsSL https://raw.githubusercontent.com/opentdm/opentdm/main/install.sh | bash -s -- --version v0.1.0 --bin-dir /usr/local/bin
```

The script downloads the matching binary (and verifies its checksum) from GitHub Releases.

## Commands

```
opentdm login --host URL --token TOKEN [--project SLUG]
opentdm pull  --env ENV [--project SLUG] [--format dotenv|json|shell|yaml|properties] [-o FILE]
opentdm run   --env ENV [--project SLUG] -- <command> [args...]
opentdm configs set --env ENV [--secret] CONFIG KEY=VAL [KEY=VAL...]   # needs a user PAT
opentdm push-file   --env ENV --file PATH CONFIG                        # needs a user PAT
opentdm version
```

- **`login`** stores credentials in `~/.opentdm/config.json`.
- **`pull`** prints (or writes with `-o`) the resolved environment in the chosen format.
- **`run`** resolves the environment and injects it as environment variables for the child command.
- **`configs set` / `push-file`** write variables / file content — these require a **user PAT** (`otdmu_…`);
  read-only service tokens (`otdm_…`) cannot write.

## Authentication

A service token (`otdm_…`) is read-only (`pull` / `run`); a user PAT (`otdmu_…`) can also write.

**Precedence:** flags > `OPENTDM_HOST` / `OPENTDM_TOKEN` / `OPENTDM_PROJECT` env > `~/.opentdm/config.json`.
