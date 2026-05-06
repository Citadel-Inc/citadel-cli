# Spec — cli-deploy-tokens

| | |
|---|---|
| Status | DRAFT |
| Authored | 120000ZMAY26 |
| Owner | unassigned |

## Why

Citadel deploy tokens are scoped, short-lived credentials issued against a
repo or namespace for use in CI/CD pipelines and automated scripts. There is
no CLI surface to create, list, or revoke them today; operators must use the
web UI or raw API calls. This spec adds first-class deploy-token management
commands under `citadel-cli deploy-token`.

## In scope

- `deploy-token list [--repo <path>] [--namespace <ns>]` — list tokens (table / JSON / YAML)
- `deploy-token create --repo <path> | --namespace <ns> [--expires <duration>] [--name <label>]` — mint a token; print cleartext once
- `deploy-token revoke <id>` — delete a token by ID; support `--dry-run`
- JSON / YAML / table `--output` parity with other list commands
- Shell completion for token IDs (revoke)
- Pagination via `--cursor` / `--all` on list

## Out of scope

- Token rotation (out of scope until server supports it)
- Bulk revoke by repo or namespace filter
- Non-repo / non-namespace scoped tokens (if the server adds them, extend separately)

## Decision log

| Q | Proposal | Status |
|---|----------|--------|
| Q1 | Top-level `deploy-token` vs. nested under `repo`/`namespace`? | OPEN — top-level keeps surface flat and parallels `token`; nesting would match web UI hierarchy |
| Q2 | Cleartext token printed to stdout or stderr? | OPEN — stdout for pipeline-ability (redirect to file); warn on stderr to store securely |
| Q3 | API endpoints — confirm server routes exist before claiming P0 | OPEN — needs server-side API survey |
