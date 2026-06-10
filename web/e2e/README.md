# opentdm web E2E smoke suite

Visual smoke tests covering critical user journeys in the opentdm React SPA.

## Running locally

From the `web/` directory:

```bash
npm run e2e
```

Or directly:

```bash
bash e2e/run.sh
```

The script:
1. Generates ephemeral secrets (no hardcoded values).
2. Starts a throwaway `postgres:16-alpine` + app container on host port **18099** using Docker Compose project `opentdm-e2e` — completely isolated from any dev stack on :18080.
3. Polls `/readyz` until the app is ready (up to 180 s, including the Docker image build).
4. Captures the first-run setup token from container logs and bootstraps an admin account.
5. Seeds a "Payments API" project with staging/production environments, an `app-config` variable object, and 3+ sequential item PUTs so version history has delta badges.
6. Installs the Playwright Chromium browser if not already present.
7. Runs all journeys in **two projects**: `chromium-light` and `chromium-dark`.
8. Tears down the stack and volumes on exit (whether pass or fail).

## Journeys covered

| File | What is tested |
|---|---|
| `journeys/projects.spec.ts` | Projects grid, project card visibility, meta grid, env pills, objects list |
| `journeys/object.spec.ts` | Code view render, branch/env menu switching, version history with delta badges |
| `journeys/settings-palette.spec.ts` | Settings (Profile, Appearance, Activity, Users), command palette (Cmd+K), dark mode toggle |

## Artifacts

After a run:
- **Screenshots**: `e2e/test-results/*.png`
- **HTML report**: `e2e/playwright-report/index.html` — open with `npx playwright show-report e2e/playwright-report`
- **Traces** (on retry): `e2e/test-results/`

These paths are gitignored.

## CI

This suite is intentionally **not** part of the required CI checks in `.github/workflows/ci.yml`. It is a local developer tool for catching integration regressions before opening a PR. It requires Docker and sufficient RAM to build the multi-stage image.
