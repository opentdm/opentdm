#!/usr/bin/env bash
# PreToolUse hook: block direct edits to generated/embedded artifacts that must
# be regenerated, not hand-edited.
set -euo pipefail
file="$(cat | python3 -c "import sys,json; print(json.load(sys.stdin).get('tool_input',{}).get('file_path',''))" 2>/dev/null || true)"
case "${file:-}" in
  */server/internal/webui/dist/*)
    cat <<'JSON'
{"hookSpecificOutput":{"hookEventName":"PreToolUse","permissionDecision":"deny","permissionDecisionReason":"server/internal/webui/dist is the committed build of the web UI (consumed by go:embed). Don't edit it by hand — change web/src and run 'cd web && npm run build'."}}
JSON
    exit 0
    ;;
esac
exit 0
