# Releasing opentdm

Releases are tag-driven. Pushing a `v*` tag triggers
[`.github/workflows/release.yml`](./.github/workflows/release.yml), which:

- builds the **CLI** for linux/darwin/windows × amd64/arm64 via GoReleaser and uploads the archives +
  `checksums.txt` to the GitHub Release (notes auto-generated from merged PRs);
- builds and pushes the **multi-arch server image** to `ghcr.io/opentdm/opentdm` (tags `vX.Y.Z`, `latest`,
  and `sha-…`);
- makes the **GitHub Action** usable at `opentdm/opentdm/action@vX.Y.Z` (it installs the matching CLI).

The version string is injected at build time via `-ldflags -X main.version=<tag>` (Dockerfile + GoReleaser),
so `opentdm version` / `opentdm-server version` print the tag.

## Prerequisite: the repo must live under the `opentdm` org

The release refs are hardcoded to **`opentdm/opentdm`**: GoReleaser `release.github.owner`
([`.goreleaser.yaml`](./.goreleaser.yaml)), the GHCR image ([`release.yml`](./.github/workflows/release.yml)),
and `REPO` in [`install.sh`](./install.sh). A tag pushed from a different namespace (e.g. a personal fork)
will **fail to publish** — the GHCR push has no permission to the `opentdm` namespace and GoReleaser targets
the wrong repo.

**Before the first release, move the repository under the `opentdm` GitHub org** (or, if you keep a different
home, retarget those three refs + the README/Action URLs to match). No code change is needed for the org move;
`GITHUB_TOKEN` in Actions then has `packages: write` for `ghcr.io/opentdm/opentdm`.

## Cut a release

1. Land all changes on `main`; make sure CI is green.
2. Update [`CHANGELOG.md`](./CHANGELOG.md): set the `[0.1.0]` date and link.
3. Tag and push:
   ```bash
   git checkout main && git pull
   git tag v0.1.0
   git push origin v0.1.0
   ```
4. Watch the **Release** workflow finish (CLI + image jobs).

## Verify the release

```bash
# CLI install + version
curl -fsSL https://raw.githubusercontent.com/opentdm/opentdm/main/install.sh | bash
opentdm version            # -> opentdm v0.1.0

# Server image
docker run --rm ghcr.io/opentdm/opentdm:v0.1.0 version   # -> opentdm-server v0.1.0
```

In a consumer repo, pin the Action: `uses: opentdm/opentdm/action@v0.1.0`.

## Dry-run locally (no publish)

Validate the pipeline without tagging:

```bash
# CLI archives (requires goreleaser; GOWORK=off because cli/ is a standalone module)
cd cli && GOWORK=off goreleaser release --snapshot --clean

# Server image + version injection
docker build --build-arg VERSION=v0.0.0-dryrun -t opentdm:dryrun .
docker run --rm opentdm:dryrun version
```

## Versioning & migrations

- SemVer. Pre-1.0 (`0.x`) may include breaking schema/API changes between minors; call them out in the
  changelog.
- Migrations are embedded and applied on startup under a Postgres advisory lock (`OPENTDM_MIGRATE_ON_START`,
  default `true`). For zero-downtime upgrades, set it to `false` and run the new version once to migrate before
  flipping traffic.
