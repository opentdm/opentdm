#!/usr/bin/env bash
# Stop hook: advisory reminder if the web source changed but the committed
# embedded build (server/internal/webui/dist) was not rebuilt. Mirrors the CI
# "Web build" stale-embed gate locally so it's caught before the push.
set -euo pipefail
cd "${CLAUDE_PROJECT_DIR:-.}" 2>/dev/null || exit 0
command -v git >/dev/null 2>&1 || exit 0

changed="$(git status --porcelain 2>/dev/null | awk '{print $2}')" || exit 0
echo "$changed" | grep -Eq '^web/src/' || exit 0          # no web source change → nothing to check
echo "$changed" | grep -Eq '^server/internal/webui/dist/' && exit 0  # embed also changed → assume rebuilt

echo "⚠️  web/src changed but server/internal/webui/dist was not rebuilt." >&2
echo "    Run: cd web && npm run build   (Node 22 — see web/.nvmrc), then commit the dist/ result." >&2
echo "    CI's 'Web build' job fails on a stale embed (git diff --quiet server/internal/webui/dist)." >&2
exit 0
