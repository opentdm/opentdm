#!/usr/bin/env bash
# PostToolUse hook: gofmt Go files after they're written/edited.
set -euo pipefail
file="$(cat | python3 -c "import sys,json; print(json.load(sys.stdin).get('tool_input',{}).get('file_path',''))" 2>/dev/null || true)"
[ -n "${file:-}" ] || exit 0
case "$file" in
  *.go)
    if [ -f "$file" ] && command -v gofmt >/dev/null 2>&1; then
      gofmt -w "$file" || true
    fi
    ;;
esac
exit 0
