# Tasks — cli-deploy-tokens

Status: DRAFT

## P0

- [ ] [HUMAN] NOMAD ratifies Q-table (Q1–Q3).
- [ ] [HUMAN] Server-side API survey: confirm deploy-token CRUD routes and
  response shapes are stable; update spec Decision Log Q3.
- [ ] A1. Scaffold `cmd/deploy_token.go` with `deploy-token` top-level cobra
  command and three subcommands (list, create, revoke).
- [ ] A2. Implement `deploy-token list` (pagination, --output table/json/yaml/csv/ndjson).
- [ ] A3. Implement `deploy-token create` (print cleartext once; --dry-run no-op for create).
- [ ] A4. Implement `deploy-token revoke <id>` with `--dry-run` support.

## P1

- [ ] B1. Shell completion: token IDs for `revoke`, repo/namespace names for `list`/`create`.
- [ ] B2. Handler tests covering list, create, revoke happy paths and 404/401 error branches.
- [ ] B3. `--watch` support on `list` if SSE events are available for deploy-token mutations.

## P2

- [ ] C1. [HUMAN] Live smoke: create and revoke a deploy token against the live production host (`https://mcp.src.land` as of 2026-05-06).
- [ ] C2. Spec close.
