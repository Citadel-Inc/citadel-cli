# Spec — cli-audit-sessions

| | |
|---|---|
| Status | IN_PROGRESS 052320ZMAY26 — Bastion (J-3) claims execution |
| Authored | 050506ZMAY26 |
| Owner | Bastion (J-3) |
| Carry-forward from | `cli-audit` (DONE) ships **events** only. `internal/api/auditapi` also implements **sessions** list + detail for the same audit gate (`package auditapi` comment references fe-audit-session-drilldown). |

## Why

Session views group related actions for **incident response** and **“what did this token do in one login”** narratives. PoL appendix K lists `audit.session`; parity requires CLI access beside **`audit list` / `audit show`** on events.

## Daemon HTTP contract (`auditapi.Handler.Routes`)

| Route | Handler | Purpose |
|-------|---------|---------|
| `GET /api/audit/sessions` | `handleListSessions` | List session summaries for a namespace — **`ns` query parameter REQUIRED** (`400 ns_required` if absent after decode). Namespace resolved via `auth.ResolveNamespace`; RBAC `audit:read` on resolved namespace — denial surfaces as **`404 audit_sessions_unavailable`** (opaque). |
| `GET /api/audit/sessions/{session_id}` | `handleGetSession` | Session drill-down payload; RBAC + optional operator-only fields (`ShowOperatorConsole` when `operator:audit:read`). Missing session → **`404 audit_session_not_found`**. |

**List query semantics** (`handleListSessions`):

| Param | Behaviour |
|-------|-----------|
| `ns` | Namespace slug (path-decoded like issues API — `%2F` → `/`). **Required.** |
| `since` | RFC3339 **or** shorthand `1h`, `24h`/`1d`, `7d`, `30d`; empty → default **last 24h** window from server clock. Invalid → `400 invalid_since`. |
| `limit` | Optional int, default **20**, positive only parsed. |
| `offset` | Optional int, default **0**, non-negative. |
| `actor_type` | Optional filter string (`actorFilter`). |

**Response shape:** `{"sessions": [...summaries]}` (exact fields — copy from `audit.Service.ListSessions` row struct during implementation).

**Rate limits:** `userBucketLimiter` — `429 rate_limited`.

**Note:** Pagination here uses **`offset`/`limit`**, not necessarily opaque cursor — **do not** blindly reuse `cli-pagination` cursor flags unless extended intentionally (Q-table).

## In scope

**Extend parent:** `citadel-cli audit` — nested commands:

| CLI | HTTP |
|-----|------|
| `audit sessions list` | `GET /api/audit/sessions` |
| `audit sessions show <session_id>` | `GET /api/audit/sessions/{session_id}` |

**Flags**

- `list`: **`--ns`** (required) mapping to `ns=` query; **`--since`** matching server shorthand + RFC3339; **`--limit`**, **`--offset`**; **`--actor-type`** if exposed.
- **cli-output-formats** on both verbs.

## Out of scope

- **SSE tail / follow** — deferred per `cli-audit` retrospective.
- **Mutating audit** — read-only API.

## Decisions

| # | Question | Proposed default | NOMAD |
|---|----------|------------------|-------|
| Q1 | Nested `audit sessions` vs top-level `audit-session`? | Nested `audit sessions list` / `audit sessions show`. | Ratified 052320ZMAY26 |
| Q2 | Add `--namespace` alias for `--ns`? | Yes — either **`--ns`** or **`--namespace`** / **`-n`** (same semantics). | Ratified 052320ZMAY26 |
| Q3 | Offset pagination vs migrate server to cursor later? | CLI exposes **`--limit`** / **`--offset`** matching the daemon until cursors exist. | Ratified 052320ZMAY26 |

## Acceptance

- A1. Both routes wired; tests cover **`ns_required`**, **`invalid_since`**, empty sessions list, show **404** paths.
- A2. Operator-console field suppression respected (tests mock JSON shapes).
- A3. Q-table ratified.
- A4. Optional `CITADEL_TEST_AUDIT_SESSIONS_LIVE=1`.
