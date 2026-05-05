# Plan — cli-audit-sessions

## ORIENT

- Reference implementation: `cmd/audit.go` + `specs/done/cli-audit/spec.md`.
- Server: `internal/api/auditapi/handler.go` — `Routes()` registers sessions + events on same mux.

## RECON

Survey list handler query params (`handleListSessions`) for filters, pagination keys (`next_cursor`), and session detail JSON shape from `handleGetSession`.

## Implementation sketch

- Add `auditSessionsListCmd`, `auditSessionsShowCmd` under a `sessions` parent (`audit sessions`).
- Reuse `addPaginationFlags`, `addOutputFlag`, error mapping from events.

## Risks

- **Schema drift** between events and sessions — tests lock minimal JSON shapes.
