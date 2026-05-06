# Tasks — cli-agent-create

Status: DRAFT 061500ZMAY26

Blocked by: none. `POST /api/agents` confirmed live (referenced in cli-oauth-login §21). Requires NOMAD Q-table ratification before implementation.

## P0

- [ ] [HUMAN] NOMAD ratifies Q-table (Q1–Q4).
- [ ] A1. Survey `POST /api/agents` request/response shape. Confirm org-scoped path (if different). Document in plan.md.
- [ ] A2. `cmd/agent.go`: add `createCmd` subcommand with `--org`, `--description`, `--output` flags and `--help` text.
- [ ] A3. Implement `runAgentCreate`: call API, print agent ID + one-time token with save-this-token notice.

## P1

- [ ] B1. Error handling: 409 → "agent name already taken", 403 → "insufficient permission", 422 → field-level hint.
- [ ] B2. `--output json`: structured emit `{ id, name, token, created_at }` only.
- [ ] B3. Handler-level test against httptest fixture: happy path (201), name-conflict (409), forbidden (403).

## P2

- [ ] C1. `make verify` passes.
- [ ] C2. README / HUMANS.md: update agent section with `create` verb.
- [ ] C3. [HUMAN] Operator smoke: `citadel-cli agent create` against a real test instance; confirm agent appears in `agent list`.
- [ ] C4. Spec close.
