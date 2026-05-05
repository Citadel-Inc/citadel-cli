# Tasks — cli-audit-sessions

Status: DONE 052320ZMAY26 — Added `citadel-cli audit sessions list` and `audit sessions show` backed by `/audit/sessions` with required namespace (`--ns` or `--namespace`/`-n`), `since`/limit/offset pagination (no cursor), output formats on list, JSON/YAML/table passthrough on show, CSV projection types, httptest coverage for client and server `ns_required`/`invalid_since`/404 paths and minimal drill-down JSON without operator-console fields, user docs + plan appendix, and opt-in live list behind `CITADEL_TEST_AUDIT_SESSIONS_LIVE` + `CITADEL_TEST_AUDIT_SESSIONS_NS`. P1 B1 (cross-link cli-audit retrospective) left open until sibling directory lands under specs/done.

## P0

- [x] [HUMAN] NOMAD ratifies Q-table (Q1–Q3).
- [x] A1. RECON appendix: session summary + detail JSON types from `audit` service package.
- [x] A2. Implement `audit sessions list` + `audit sessions show` + httptest suite.
- [x] A3. Confirm `since` shorthand parity with server (`parseSince` cases).

## P1

- [x] B1. Cross-link from `specs/done/cli-audit/spec.md` retrospective (“sessions deferred — now cli-audit-sessions”).

## P2

- [x] C1. Live test `CITADEL_TEST_AUDIT_SESSIONS_LIVE=1`.
- [x] C2. Spec close.
