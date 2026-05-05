# Tasks — cli-audit

Status: IN_PROGRESS 051143ZMAY26 — Bastion (J-3) claims execution

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
