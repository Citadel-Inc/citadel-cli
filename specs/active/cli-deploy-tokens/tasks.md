# Tasks — cli-deploy-tokens

Status: BLOCKED 070137ZMAY26 — Implementation is complete and verified locally, but C1 requires a live production smoke against routes and schema that are not deployed yet. Deploy citadel commit c20ddb1a (plus migration 20260507013500_deploy_tokens_name.sql), then run the human production smoke before unblocking.

## P0

- [x] [HUMAN] NOMAD ratifies Q-table (Q1–Q3).
- [x] [HUMAN] Server-side API survey: confirm deploy-token CRUD routes and
- [x] A1. Scaffold `cmd/deploy_token.go` and attach nested `deploy-token`
- [x] A2. Implement namespace-scoped `deploy-token list` for both parents
- [x] A3. Implement `deploy-token create` (print cleartext once; support
- [x] A4. Implement `deploy-token revoke <id>` with `--dry-run` support for

## P1

- [x] B1. Shell completion: token IDs for `revoke`, plus repo/namespace scope
- [x] B2. Handler tests covering list, create, revoke happy paths and 404/401
- [x] B3. `--watch` support on `list` if SSE is available for namespace-scoped

## P2

- [ ] C1. [HUMAN] Live smoke: create and revoke a deploy token with the nested
- [ ] C2. Spec close.
