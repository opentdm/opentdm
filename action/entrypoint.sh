#!/usr/bin/env bash
# Composite-action entrypoint: install the opentdm CLI, resolve a project/env,
# and inject the variables into the job (masking secrets) or write a file.
set -euo pipefail

VERSION="${INPUT_VERSION:-latest}"

# Install the CLI unless it's already on PATH (the action-test injects it).
if ! command -v opentdm >/dev/null 2>&1; then
  curl -fsSL "https://raw.githubusercontent.com/opentdm/opentdm/main/install.sh" | bash -s -- --version "$VERSION"
  export PATH="$HOME/.local/bin:$PATH"
fi

PROJECT="$INPUT_PROJECT"
ENVIRONMENT="$INPUT_ENV"

case "${INPUT_FORMAT:-env}" in
  dotenv-file)
    opentdm pull --project "$PROJECT" --env "$ENVIRONMENT" --format dotenv -o "$INPUT_OUTPUT"
    echo "Wrote variables to $INPUT_OUTPUT"
    ;;
  json)
    opentdm pull --project "$PROJECT" --env "$ENVIRONMENT" --format json -o "$INPUT_OUTPUT"
    echo "Wrote variables to $INPUT_OUTPUT"
    ;;
  env|*)
    json="$(opentdm pull --project "$PROJECT" --env "$ENVIRONMENT" --format json)"
    # Randomized heredoc delimiter so a value containing the delimiter can't break
    # out of the $GITHUB_ENV assignment.
    delim="OPENTDM_EOF_$(openssl rand -hex 16)"
    count=0
    while IFS= read -r row; do
      [ -z "$row" ] && continue
      kv="$(printf '%s' "$row" | base64 --decode)"
      key="$(printf '%s' "$kv" | jq -r '.key')"
      val="$(printf '%s' "$kv" | jq -r '.value')"
      # Mask first so the value never appears in subsequent logs.
      echo "::add-mask::$val"
      {
        printf '%s<<%s\n' "$key" "$delim"
        printf '%s\n' "$val"
        printf '%s\n' "$delim"
      } >> "$GITHUB_ENV"
      count=$((count + 1))
    done < <(printf '%s' "$json" | jq -r 'to_entries[] | @base64')
    echo "Injected $count variable(s) into the job environment"
    ;;
esac
