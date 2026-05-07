# Tasks — cli-agent-create

Status: DONE 070504ZMAY26

Blocked by: none. `POST /api/agents` confirmed live (referenced in cli-oauth-login §21). Requires NOMAD Q-table ratification before implementation.

## P0

- [x] [HUMAN] NOMAD ratifies Q-table (Q1–Q4).
- [x] A1. Survey `POST /api/agents` request/response shape. Confirm org-scoped path (if different). Document in plan.md.
- [x] A2. `cmd/agent.go`: add `createCmd` subcommand with `--org`, `--description`, `--output` flags and `--help` text.
- [x] A3. Implement `runAgentCreate`: call API, print agent ID + one-time token with save-this-token notice.

## P1

- [x] B1. Error handling: 409 → "agent name already taken", 403 → "insufficient permission", 422 → field-level hint.
- [x] B2. `--output json`: structured emit `{ id, name, token, created_at }` only.
- [x] B3. Handler-level test against httptest fixture: happy path (201), name-conflict (409), forbidden (403).

## P2

- [x] C1. `make verify` passes.
- [x] C2. README / HUMANS.md: update agent section with `create` verb.
- [x] C3. [HUMAN] Operator smoke: `citadel-cli agent create` against a real test instance; confirm agent appears in `agent list`.
- [x] C4. Spec close.
