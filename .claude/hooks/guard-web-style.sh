#!/usr/bin/env bash
# PostToolUse hook: advisory web style/discipline checks on the edited file.
# Non-blocking (exit 0 + stderr notes) — ESLint is the hard CI gate; this is a
# fast nudge. Reads tool_input.file_path from stdin JSON, like gofmt.sh.
set -euo pipefail
file="$(cat | python3 -c "import sys,json; print(json.load(sys.stdin).get('tool_input',{}).get('file_path',''))" 2>/dev/null || true)"
[ -n "${file:-}" ] || exit 0
[ -f "$file" ] || exit 0

case "$file" in
  */web/src/*) ;;            # only the SPA source
  *) exit 0 ;;
esac

warn() { echo "ℹ️  guard-web-style: $1" >&2; }

case "$file" in
  *.tsx|*.ts)
    case "$file" in
      */ui/primer.tsx) ;;    # the shim is the one allowed @primer/react importer
      *)
        if grep -Eq "from ['\"]@primer/react" "$file"; then
          warn "$file imports @primer/react directly — import from the ui/primer shim instead (preserves sx)."
        fi
        ;;
    esac
    # Heuristic: non-static sx (pseudo-selector / nesting / responsive array).
    if grep -Eq "sx=\{\{[^}]*(:hover|:focus|:active|&|\[)" "$file"; then
      warn "$file has a possibly non-static sx (pseudo/&/array) — sx must be static; use a className + CSS."
    fi
    ;;
esac

# Hardcoded hex colors outside the token files / theme-preview swatches.
case "$file" in
  */ui/tokens.css|*/ui/primitives.css|*/settings/AppearancePanel.tsx) ;;
  *.tsx|*.css)
    if grep -Eiq "#[0-9a-f]{3}([0-9a-f]{3})?\b" "$file"; then
      warn "$file has a hardcoded hex color — use a Primer functional var (--fgColor-*/--bgColor-*) or an sx token."
    fi
    ;;
esac
exit 0
