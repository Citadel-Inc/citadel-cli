# Spec — cli-agent-create

| | |
|---|---|
| Status | DONE 070504ZMAY26 |
| Authored | 061500ZMAY26 |
| Owner | Bastion (J-3) |
| Carry-forward from | Gap surfaced in citadel-cli third-pass review (2026-05-05): `agent` subcommand has list / get / delete / rotate-token but no `create`. Server-side `POST /api/agents` is confirmed live (referenced in `cli-oauth-login` spec §21). `live.ts` "Agent ergonomics" is scoped Phase 1. |

## Why

Today a human or CI pipeline that wants to register a new agent must use the web UI or call the HTTP API out-of-band. `citadel-cli agent create` closes the headless registration gap — the same operator who can `agent list` and `agent delete` should be able to provision a new agent without leaving the terminal.

The server's `POST /api/agents` endpoint already exists; the `cli-oauth-login` spec calls it internally for find-or-create. This spec surfaces it as a named CLI verb so operators can provision agents for any namespace (not just the local `citadel-cli@hostname` auto-agent).

## In scope

- `citadel-cli agent create <name> [--org <org-slug>] [--description "..."] [--output json|yaml|...]`
- Name validation mirrors server-side constraints (slug-safe, ≤128 chars).
- On success: print the new agent ID and a one-time initial token, with a clear "save this token — it will not be shown again" notice.
- `--output json` emits a machine-readable struct: `{ "id": "...", "name": "...", "token": "...", "created_at": "..." }`.
- `--org`: creates the agent under the specified org namespace (requires org-scoped agent-create permission). Default: the authenticated user's personal namespace.
- Error paths: name conflict (409 → friendly "name taken" message), insufficient permission (403), validation error (422 with field hints).

## Out of scope

- Scoped-token wizards or default-policy templates (in `live.ts` "Agent ergonomics" — Phase 1 wider feature).
- Fleet-level batch creation or CSV import.
- Automatic rotation scheduling (separate feature).
- `agent update` (rename, description change) — follow-on; no daemon endpoint exists yet.

## Decision log

| Q | Proposal | Status |
|---|----------|--------|
| Q1 | Token display: show initial token inline vs. force `agent rotate-token` as first action? | **Ratified** — inline on create; mirrors how every other tool bootstraps agents. |
| Q2 | `--org` flag vs. positional `<org/name>` syntax? | **Ratified** — `--org` flag; keeps positional arg unambiguous and matches existing verb patterns in the CLI. |
| Q3 | If name already exists (409): error-and-exit vs. return existing agent? | **Ratified** — error-and-exit; find-or-create semantics belong in `cli-oauth-login` auto-agent path, not user-facing `create`. |
| Q4 | One-time token: warn-only vs. require `--confirm-token-shown` acknowledgement flag? | **Ratified** — warn-only at v1; flag gating is friction without clear security benefit for a local CLI session. |

## Acceptance

- A1. `citadel-cli agent create <name>` creates an agent in the authenticated user's namespace; prints agent ID + one-time token.
- A2. `--org <slug>` creates the agent under the specified org (403 if caller lacks permission).
- A3. `--output json` emits `{ id, name, token, created_at }` and nothing else on stdout.
- A4. Name conflict (409) surfaces a clear "agent name already taken" error, not a raw HTTP status.
- A5. Token is printed exactly once with a "save this token" notice; running `agent get` afterward does not show the token.
- A6. `make verify` passes including a handler-level test against an httptest fixture.
- A7. Q-table ratified.

## Resolution

Shipped 070504ZMAY26. `citadel-cli agent create <name>` implemented in `cmd/agent.go` with `--org`, `--description`, `--output` flags. All acceptance criteria met. Server-side composite auth middleware (`auth.AgentOrJWTMiddleware`) added to `citadel` so agent tokens are accepted on REST `/api/agents` routes — fixing a root auth architecture gap that blocked smoke testing. `created_at` field bug (`RETURNING id` → `RETURNING id, created_at`) also fixed. Both server fixes committed and deployed. Smoke test confirmed: `agent create`, `agent list`, `agent delete` all work end-to-end with an opaque agent token session.
