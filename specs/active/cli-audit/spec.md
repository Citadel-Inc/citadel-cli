# Spec — cli-audit

| | |
|---|---|
| Status | IN_PROGRESS 051143ZMAY26 — Bastion (J-3) claims execution |
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

## Decision log

| Q | Proposal | Status |
|---|----------|--------|
| Q1 | Server endpoint shape: `GET /api/audit/events?since=...&kind=...&namespace=...` returning paginated list? | **Open** — recommended; matches existing API conventions. Daemon-side companion spec needed. |
| Q2 | Time-filter unit: durations (`--since 1h`) vs. absolute timestamps (`--since 2026-05-05T00:00:00Z`)? | **Open** — both: parse a value matching `time.ParseDuration` first, fall back to RFC3339. |
| Q3 | Default `--since`: 24h vs. 1h vs. unbounded? | **Open** — 24h; long enough to catch yesterday-evening events without dumping the whole table. |
| Q4 | `--kind <glob>`: literal `*` glob vs. regex? | **Open** — glob (`fnmatch`-style); operators don't want to escape regex specials in shell. |
| Q5 | Show actor as UUID vs. resolved slug (extra round trip)? | **Open** — slug if cached locally, else UUID; no extra round trip. |
| Q6 | RBAC: are non-operator users allowed to query audit events for namespaces they own? | **Open** — yes, scoped to namespaces they own + actions they performed. Operator role unlocks cross-tenant. |

## Acceptance

- A1. Daemon: `GET /api/audit/events` endpoint with the filter set above. (Companion daemon spec.)
- A2. CLI: `audit list` honors all filters + cli-pagination contract + cli-output-formats contract.
- A3. CLI: `audit show <event-id>` returns the full event row + cascade children.
- A4. Q-table ratified.
- A5. Live integration test (`CITADEL_TEST_AUDIT_LIVE=1`) creates a known event (e.g., agent create) and confirms it surfaces in `audit list --kind agent.created --since 1m`.
- A6. Tail mode (`--follow`) explicitly out of scope at v1; tracked as a follow-on once daemon ships SSE.
