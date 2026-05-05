# Tasks — cli-audit

Status: DONE 051145ZMAY26 — Shipped Citadel GET /api/audit/events and GET /api/audit/events/{id} with RBAC, time and kind filters, cli-pagination cursors, cascade linkage from purge, and agent.created audit rows. Delivered citadel-cli audit list/show with standard output modes, live opt-in test, and documentation. Deferred: P1 B6 expanded RBAC HTTP matrix for events; P2 operator smoke, tail-mode carry-forward, and spec hygiene.

## P0

- [x] [HUMAN] NOMAD ratifies Q-table (Q1-Q6).
- [x] A1. [SERVER] `GET /api/audit/events` with filters (since, until, kind, namespace, actor, cursor, limit). Pagination follows cli-pagination shape.
- [x] A2. CLI: `cmd/audit.go` parent + `audit list` subcommand.

## P1

- [x] B1. CLI: `audit show <event-id>` subcommand.
- [x] B2. CLI: time-filter parser accepting both durations (1h) and RFC3339 (2026-05-05T00:00:00Z).
- [x] B3. CLI: glob-kind filter via `path.Match`-style pattern.
- [x] B4. CLI: actor-slug resolver (cached lookup against /agents + /users to avoid N round trips).
- [x] B5. Pagination + output-format integration.
- [ ] B6. Tests: handler-level happy + filter combinations + RBAC scoping (404 vs. 403 vs. 200).

## P2

- [x] C1. Live integration test (`CITADEL_TEST_AUDIT_LIVE=1`) round-trip.
- [x] C2. README + HUMANS.md "Audit log access" section + per-kind list (operator-relevant kinds documented).
- [ ] C3. [HUMAN] Operator smoke: real audit query against prod namespace.
- [ ] C4. Follow-on spec carry-forward: tail mode (`--follow`) once daemon ships SSE.
- [ ] C5. Spec close.
