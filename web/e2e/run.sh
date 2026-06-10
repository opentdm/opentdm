#!/usr/bin/env bash
# E2E smoke suite runner for opentdm web.
# Spins up a throwaway Postgres + app stack on :18099, seeds data,
# runs Playwright light+dark journeys, then tears down.
# Usage: npm run e2e   (from web/)
#        bash e2e/run.sh  (from web/)
set -euo pipefail

# Change to the e2e directory so docker compose file paths resolve correctly
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
cd "$SCRIPT_DIR"

COMPOSE_PROJECT="opentdm-e2e"
COMPOSE_FILE="compose.e2e.yaml"
WEB_DIR="$(dirname "$SCRIPT_DIR")"

# --- Generate ephemeral secrets if not already set ---
if [ -z "${OPENTDM_MASTER_KEY:-}" ]; then
  OPENTDM_MASTER_KEY="$(openssl rand -base64 32)"
  export OPENTDM_MASTER_KEY
fi
if [ -z "${OPENTDM_TOKEN_PEPPER:-}" ]; then
  OPENTDM_TOKEN_PEPPER="$(openssl rand -base64 32)"
  export OPENTDM_TOKEN_PEPPER
fi
if [ -z "${OPENTDM_SESSION_SECRET:-}" ]; then
  OPENTDM_SESSION_SECRET="$(openssl rand -base64 32)"
  export OPENTDM_SESSION_SECRET
fi
if [ -z "${E2E_PASSWORD:-}" ]; then
  E2E_PASSWORD="$(openssl rand -base64 18)"
  export E2E_PASSWORD
fi

export E2E_BASE_URL="${E2E_BASE_URL:-http://localhost:18099}"
export E2E_USERNAME="${E2E_USERNAME:-e2e-admin}"
export E2E_EMAIL="${E2E_EMAIL:-e2e@example.test}"

echo "[e2e] Starting throwaway stack on ${E2E_BASE_URL}…"
echo "[e2e] Compose project: ${COMPOSE_PROJECT}"

# --- Cleanup trap ---
# Always bring down the stack and remove volumes on exit
cleanup() {
  local exit_code=$?
  echo ""
  echo "[e2e] Tearing down stack (exit code: ${exit_code})…"
  docker compose -f "$COMPOSE_FILE" -p "$COMPOSE_PROJECT" down -v --remove-orphans 2>/dev/null || true
  exit $exit_code
}
trap cleanup EXIT INT TERM

# --- Start the stack ---
docker compose -f "$COMPOSE_FILE" -p "$COMPOSE_PROJECT" up -d --build

# --- Wait for the app to be ready ---
echo "[e2e] Waiting for app to be ready at ${E2E_BASE_URL}/readyz (up to 180s)…"
TIMEOUT=180
ELAPSED=0
until curl -sf "${E2E_BASE_URL}/readyz" >/dev/null 2>&1; do
  if [ "$ELAPSED" -ge "$TIMEOUT" ]; then
    echo "[e2e] ERROR: App did not become ready within ${TIMEOUT}s"
    docker compose -f "$COMPOSE_FILE" -p "$COMPOSE_PROJECT" logs app
    exit 1
  fi
  sleep 2
  ELAPSED=$((ELAPSED + 2))
done
echo "[e2e] App is ready (after ${ELAPSED}s)"

# --- Capture setup token ---
# Give a moment for the log line to appear if the app just started
sleep 1
E2E_SETUP_TOKEN="$(docker compose -f "$COMPOSE_FILE" -p "$COMPOSE_PROJECT" logs app 2>/dev/null \
  | grep 'OPENTDM setup token:' \
  | tail -1 \
  | awk '{print $NF}')"
export E2E_SETUP_TOKEN

if [ -z "$E2E_SETUP_TOKEN" ]; then
  echo "[e2e] WARNING: No setup token found in logs — assuming already bootstrapped or seeded."
else
  echo "[e2e] Setup token captured."
fi

# --- Ensure Playwright browser is installed ---
echo "[e2e] Ensuring Playwright Chromium is installed…"
cd "$WEB_DIR"
npx playwright install chromium

# --- Run the tests ---
echo "[e2e] Running Playwright tests…"
# Run from web/ dir pointing at the e2e config; capture exit code explicitly
set +e
npx playwright test -c e2e/playwright.config.ts
PLAYWRIGHT_EXIT=$?
set -e

echo "[e2e] Playwright finished with exit code: ${PLAYWRIGHT_EXIT}"
exit $PLAYWRIGHT_EXIT
