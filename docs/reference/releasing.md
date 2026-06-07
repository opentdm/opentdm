# Releasing

Releases are **tag-driven**: pushing a `v*` tag runs
[`.github/workflows/release.yml`](https://github.com/opentdm/opentdm/blob/main/.github/workflows/release.yml),
which publishes the CLI binaries (GoReleaser → GitHub Release + `checksums.txt`), the multi-arch server image
(`ghcr.io/opentdm/opentdm`), and makes the Action usable at `opentdm/opentdm/action@vX.Y.Z`. The version is
injected via `-ldflags -X main.version=<tag>`.

The canonical runbook is [`RELEASING.md`](https://github.com/opentdm/opentdm/blob/main/RELEASING.md). In short:

```bash
git checkout main && git pull
# set the CHANGELOG date, then:
git tag v0.1.0
git push origin v0.1.0
```

Verify afterward:

```bash
curl -fsSL https://raw.githubusercontent.com/opentdm/opentdm/main/install.sh | bash
opentdm version                                          # -> opentdm v0.1.0
docker run --rm ghcr.io/opentdm/opentdm:v0.1.0 version   # -> opentdm-server v0.1.0
```

**Versioning:** SemVer. Pre-1.0 (`0.x`) minors may include breaking schema/API changes — called out in the
[changelog](/reference/changelog). Migrations are embedded and applied on startup under an advisory lock; for
zero-downtime upgrades, set `OPENTDM_MIGRATE_ON_START=false` and migrate before flipping traffic.
