# Spec — cli-audit

| | |
|---|---|
| Status | DONE 051145ZMAY26 — Shipped Citadel GET /api/audit/events and GET /api/audit/events/{id} with RBAC, time and kind filters, cli-pagination cursors, cascade linkage from purge, and agent.created audit rows. Delivered citadel-cli audit list/show with standard output modes, live opt-in test, and documentation. Deferred: P1 B6 expanded RBAC HTTP matrix for events; P2 operator smoke, tail-mode carry-forward, and spec hygiene. |
| Authored | 081550ZMAY26 |
| Owner | Bastion (J-3) |
| Carry-forward from | third-pass review of `citadel-cli` (2026-05-05): operators need terminal-side access to the Citadel audit event bus for compliance + incident-response. |

## Why

The Citadel server emits structured audit events on every state change (visible in repo-delete, agent-create, oauth-revoke, etc. cascade comments — `go-telemetry-server-emit` instrumented this; `go-issues-webhooks` consumes the same bus). Operators currently have no CLI surface for read access; they must drop into the database (operator-only) or wait for a webhook.

Add `citadel-cli audit list` + `audit show` so operators can answer "what happened in my namespace in the last hour?" without leaving the terminal.

## In scope

- **`citadel-cli audit list`**: filterable list of recent audit events. Filters:
  - `--since <duration>` (e.g., `--since 1h`, `--since 30m`)
  - `--until <duration>`
  - `--kind <glob>` (e.g., `--kind repo.*`, `--kind oauth.client.revoked`)
  - `--namespace <slug>` (filter by affected namespace)
  - `--actor <user-uuid-or-slug>` (filter by initiator)
  - `--limit N` / `--cursor X` / `--all` (cli-pagination contract)
  - `--output json|ndjson|csv|yaml` (cli-output-formats contract)
- **`citadel-cli audit show <event-id>`**: full event with payload, attribution, request-id, IP (operator-only), and any cascade-child events linked to it.
- **`--follow` (`-f`) on `audit list`**: tail-f-style live stream when supported by the server (server-sent events or long-poll). v2 if SSE not available; document as out-of-scope at v1.

## Out of scope

- **Audit log retention configuration** — operator/admin surface, not user.
- **Audit log purge / redaction** — not user-facing.
- **Cross-tenant audit correlation** — single user's view only at v1.
- **Visualisation / charting** — terminal output only; ndjson exports for downstream tooling.
- **Tail mode (`--follow`)** — depends on server-side stream. Defer until SSE / long-poll endpoint exists.
- **Audit sessions (grouped views)** — shipped as follow-on [`cli-audit-sessions`](./cli-audit-sessions/spec.md) (`citadel-cli audit sessions list` / `show`).

## Decision log

| Q | Proposal | Status |
|---|----------|--------|
| Q1 | Server endpoint shape: `GET /api/audit/events?since=...&kind=...&namespace=...` returning paginated list? | **Ratified 051200ZMAY26** — `GET /api/audit/events` (+ `GET /api/audit/events/{id}`) with `next_cursor` pagination; mounted under existing JWT + audit gate like `/api/audit/sessions`. |
| Q2 | Time-filter unit: durations (`--since 1h`) vs. absolute timestamps (`--since 2026-05-05T00:00:00Z`)? | **Ratified 051200ZMAY26** — both: `time.ParseDuration` first, else RFC3339, for `since` and `until`. |
| Q3 | Default `--since`: 24h vs. 1h vs. unbounded? | **Ratified 051200ZMAY26** — 24h default server-side when `since` omitted; `until` defaults to now. |
| Q4 | `--kind <glob>`: literal `*` glob vs. regex? | **Ratified 051200ZMAY26** — dot-separated glob: `*` one segment, `**` multi-segment tail; translated to an anchored POSIX regexp server-side. |
| Q5 | Show actor as UUID vs. resolved slug (extra round trip)? | **Ratified 051200ZMAY26** — server enriches `actor_slug` in one pass (user root namespace + agent name); else UUID; CLI does not add resolver round trips. |
| Q6 | RBAC: are non-operator users allowed to query audit events for namespaces they own? | **Ratified 051200ZMAY26** — yes: rows where the caller is `actor_id`, or `namespace_id` is non-null and the caller has `audit:read` on that namespace (owner/grant walk); `operator:audit:read` bypasses scope. Operator-only payload fields stripped server-side for others. |

## Acceptance

- A1. Daemon: `GET /api/audit/events` endpoint with the filter set above. (Companion daemon spec.)
- A2. CLI: `audit list` honors all filters + cli-pagination contract + cli-output-formats contract.
- A3. CLI: `audit show <event-id>` returns the full event row + cascade children.
- A4. Q-table ratified.
- A5. Live integration test (`CITADEL_TEST_AUDIT_LIVE=1`) creates a known event (e.g., agent create) and confirms it surfaces in `audit list --kind agent.created --since 1m`.
- A6. Tail mode (`--follow`) explicitly out of scope at v1; tracked as a follow-on once daemon ships SSE.
