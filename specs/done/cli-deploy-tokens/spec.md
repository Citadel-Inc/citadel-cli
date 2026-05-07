# Spec — cli-deploy-tokens

| | |
|---|---|
| Status | DONE 043044ZMAY26 |
| Authored | 120000ZMAY26 |
| Owner | Bastion (J-3) |

## Why

Citadel deploy tokens are scoped, short-lived credentials issued against a
repo or namespace for use in CI/CD pipelines and automated scripts. There is
no CLI surface to create, list, or revoke them today; operators must use the
web UI or raw API calls. This spec adds first-class deploy-token management
commands under nested `citadel-cli repo deploy-token` and
`citadel-cli namespace deploy-token` parents so the UX matches the existing
CLI resource hierarchy.

## In scope

- `repo deploy-token {list,create,revoke}` for repo-scoped tokens
- `namespace deploy-token {list,create,revoke}` for namespace-scoped tokens
- Namespace-scoped REST support in `citadel` for list/create/revoke, reused by
  both CLI parents because repos are namespace-backed resources
- `create` supports `--expires <duration>` and `--name <label>`; the cleartext
  token is returned once at creation time
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
| Q1 | Top-level `deploy-token` vs. nested under `repo`/`namespace`? | Ratified 070128ZMAY26 — nested under `repo deploy-token` and `namespace deploy-token` to match existing CLI hierarchy; the user already directed a nested shape |
| Q2 | Cleartext token printed to stdout or stderr? | Ratified 070128ZMAY26 — print the one-time cleartext token in the machine-readable response / stdout path and emit any human warning separately, matching existing token-style commands |
| Q3 | API endpoints — confirm server routes exist before claiming P0 | Ratified 070128ZMAY26 — current `citadel` sources do not expose deploy-token CRUD routes; implement namespace-scoped REST routes in the primary server sources and consume them from the CLI |

## Blocking

Implementation is complete and verified locally, but C1 requires a live production smoke against routes and schema that are not deployed yet. Deploy citadel commit c20ddb1a (plus migration 20260507013500_deploy_tokens_name.sql), then run the human production smoke before unblocking.
