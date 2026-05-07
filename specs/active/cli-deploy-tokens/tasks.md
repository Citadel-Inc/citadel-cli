# Tasks — cli-deploy-tokens

Status: IN_PROGRESS 070123ZMAY26 — Bastion (J-3) claims execution

## P0

- [x] [HUMAN] NOMAD ratifies Q-table (Q1–Q3).
- [x] [HUMAN] Server-side API survey: confirm deploy-token CRUD routes and
  response shapes are stable; update spec Decision Log Q3.
- [ ] A1. Scaffold `cmd/deploy_token.go` and attach nested `deploy-token`
  command groups beneath both `repo` and `namespace`.
- [ ] A2. Implement namespace-scoped `deploy-token list` for both parents
  (pagination, --output table/json/yaml/csv/ndjson).
- [ ] A3. Implement `deploy-token create` (print cleartext once; support
  `--expires` and `--name`; `--dry-run` remains a no-op for create).
- [ ] A4. Implement `deploy-token revoke <id>` with `--dry-run` support for
  both parent command paths.

## P1

- [ ] B1. Shell completion: token IDs for `revoke`, plus repo/namespace scope
  completion for the nested command paths.
- [ ] B2. Handler tests covering list, create, revoke happy paths and 404/401
  error branches across the server and CLI slices.
- [ ] B3. `--watch` support on `list` if SSE is available for namespace-scoped
  deploy-token mutations.

## P2

- [ ] C1. [HUMAN] Live smoke: create and revoke a deploy token with the nested
  repo / namespace CLI paths against the live production host
  (`https://mcp.src.land` as of 2026-05-06).
- [ ] C2. Spec close.
