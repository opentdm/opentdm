# Using opentdm in CI

The core consumption primitive is **resolve**: `GET /api/v1/projects/{project}/resolve?env=…&format=…` returns
the merged (base ⊕ environment) config in the format you ask for. In CI you call it with a **read-only service
token** (project + environment scoped) via the CLI or the GitHub Action.

Mint a token in the project's **Settings → Service tokens**, scope it to the environment(s) you need, and store
it as a CI secret (e.g. `OPENTDM_TOKEN`).

## GitHub Action

```yaml
- uses: opentdm/opentdm/action@v0.1.0
  with:
    host: https://opentdm.example.com
    token: ${{ secrets.OPENTDM_TOKEN }}
    project: payments
    environment: staging
    format: env          # env (inject into the job) | dotenv-file | json
    # output-path: .env  # for dotenv-file / json
```

With `format: env`, resolved variables are injected into `$GITHUB_ENV` (secrets masked) for subsequent steps.

## CLI

```bash
# write a dotenv file:
opentdm pull --host https://opentdm.example.com --token "$OPENTDM_TOKEN" \
  --project payments --env staging -o .env

# or inject straight into a process:
opentdm run --env staging -- npm test
```

`pull` supports `--format dotenv|json|shell|yaml|properties`. See the [CLI reference](/guide/cli) for the full
command surface and auth precedence. For the raw HTTP contract, see the [REST API](/reference/rest-api).
