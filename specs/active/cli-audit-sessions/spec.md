# Spec — cli-audit-sessions

| | |
|---|---|
| Status | DRAFT 050506ZMAY26 |
| Authored | 050506ZMAY26 |
| Owner | Bastion (J-3) |
| Carry-forward from | `cli-audit` shipped **events** (`audit list` / `audit show`). Same server package registers **sessions** (`GET /api/audit/sessions`, `GET /api/audit/sessions/{session_id}`) for drill-down / replay narratives per PoL appendix K `audit.session`. |

## Why

Operators investigating incidents need **session-scoped** views (related events grouped by session id) as well as flat event timelines. The daemon exposes session list + detail with RBAC aligned to audit events.

## In scope

Extend **`citadel-cli audit`** (same parent command as events):

| Verb | HTTP |
|------|------|
| `audit sessions list` | `GET /api/audit/sessions` |
| `audit sessions show <session_id>` | `GET /api/audit/sessions/{session_id}` |

**Cross-cutting**

- Filters on list endpoint must match server query params (survey `auditapi` — `since`, `until`, `namespace`, pagination if present).
- **cli-pagination** + **cli-output-formats** parity with `audit list`.
- Human output: session summary table; show verb prints ordered steps or JSON.

## Out of scope

- **SSE live tail** — deferred in original cli-audit retrospective; unchanged.
- **Cross-tenant operator bypass** — follows existing audit RBAC; no new semantics.

## Decision log

| Q | Proposal | Status |
|---|----------|--------|
| Q1 | Subcommand shape: `audit sessions list` vs top-level `audit-session`? | **Open** — nested under `audit sessions` for cohesion. |
| Q2 | Show output: full JSON only vs pretty narrative? | **Open** — json/yaml + compact human list of contained events if API returns nested structure. |

## Acceptance

- A1. Both endpoints wired; httptest coverage mirroring `audit` event tests.
- A2. Flags documented; `make verify` passes.
- A3. Q-table ratified.
- A4. Optional `CITADEL_TEST_AUDIT_SESSIONS_LIVE=1`.
