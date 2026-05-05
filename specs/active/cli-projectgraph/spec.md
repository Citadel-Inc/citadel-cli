# Spec — cli-projectgraph

| | |
|---|---|
| Status | DRAFT 050506ZMAY26 |
| Authored | 050506ZMAY26 |
| Owner | Bastion (J-3) |
| Carry-forward from | Phase 0 / PoL gap analysis (2026-05-05): L2 Project-as-Graph is marquee; server exposes `internal/api/projectgraphapi` under `/api/projectgraph/` but `citadel-cli` has no first-class project verbs (operators must use MCP or curl). |

## Why

Citadel’s Project-as-Graph engine (`go-projectgraph`) powers pin chains, dependency walks, status rollups, and edge mutations for multi-repo projects. Phase 0 demo narratives (Bastion meta, submodule pins) assume operators can **inspect and advance** graph state without leaving the terminal or hand-crafting JSON-RPC.

Server companion context: `citadel/specs/active/go-projectgraph/spec.md` (daemon); RBAC atoms **`projectgraph:read`** (browse) and **`projectgraph:manage`** (edge writes / restore / reindex — verify exact atom names in `internal/auth` at implementation time).

## Daemon HTTP contract (authoritative for CLI mapping)

**Mount:** subtree handler at **`/api/projectgraph/`** (`cmd/citadel/main.go` wraps with JWT middleware).  
**Slug encoding:** the server uses a **manual dispatcher** — path tail after `/api/projectgraph/` supports **multi-segment** namespace paths (`org/repo`, nested projects). CLI MUST mirror `decodeSlugPath` semantics: path-unescape, normalize `%2F` → `/`, trim slashes — **do not** assume a single `{slug}` mux segment.

| CLI intent | Method | Path pattern | Notes |
|------------|--------|--------------|--------|
| Pin chain | GET | `…/{slug}/pin-chain` | Namespace resolved from path must be **`kind == repo`** else `400 repo_namespace_required`. Response: JSON array of pin rows (filtered for read). |
| Walk | GET | `…/{slug}/walk` | **Required query:** `kind` — empty → `400 kind_required`. Optional: `max_depth` (positive int; server defaults cap). |
| Neighbors | GET | `…/{slug}/neighbors` | Query: `ns` (optional; defaults from path slug), `kind`, `direction`, `include_deleted=true` for tombstones. |
| Status rollup | GET | `…/{slug}/status-rollup` | (Exact query params — capture from `handleStatusRollup` during P0 A1.) |
| Drilldown | GET | `…/{slug}/status-rollup/drilldown` | Same namespace gate as rollup. |
| Create edge | POST | `…/{slug}/edges` | JSON body `postEdgeBody`: `from_namespace_id`, `from_kind`, `to_namespace_id?`, `to_kind`, `to_external_id?`, `edge_type`, `attrs`, `source` (must be `"manual"` at v1 or `400 manual_source_only`). Requires **`projectgraph:manage`** on **from** namespace. Success: `201` + `{"status":"ok"}`. |
| Delete edge | DELETE | `…/{slug}/edges/{edge_id}` | UUID `edge_id`; resolve `from_namespace_id` for RBAC. |
| Restore edge | POST | `…/{slug}/edges/{edge_id}/restore` | Same RBAC pattern as delete. |
| Reindex | POST | `…/{slug}/reindex` | Ingest hook — confirm confirmation UX with destructive-pattern parity. |
| Recovery scan | POST | `…/admin/recovery-scan` | Admin-only — document server gate; may be operator-only. |

**Opaque deny:** on missing RBAC the API often returns **`404 not_found`** (not `403`) to avoid leaking namespace existence — CLI copy must not promise “does not exist” when read failure may be permission-shaped.

**Rate limits:** per-user bucket (`userLimiter`) — CLI should surface `429` + `rate_limited` via existing errmap patterns.

## In scope

**Parent command:** `citadel-cli project` (see Q-table for naming).

**Read verbs** — implement rows in contract table **before** writes.

**Write verbs** — require typed confirmation or `--yes`; **`PermProjectgraphManage`** paths must explain missing scope in user-facing text.

**Cross-cutting**

- **cli-output-formats** on all JSON responses; default human: compact tables for pin-chain / rollup where feasible.
- Large payloads (`walk`, `neighbors`): recommend `--output json` + jq in docs.

## Out of scope

- **Replacing git** for submodule operations — users still run `git submodule`; Citadel records pins via API/MCP.
- **Ingest queue HUD** — worker stats belong elsewhere unless exposed as stable JSON.
- **MCP** — HTTPS MCP canonical (`specs/parked/README.md`); this spec is REST via `apiclient` only.

## Decision log

| Q | Proposal | Status |
|---|----------|--------|
| Q1 | Top-level `project` vs `namespace project`? | **Open** — top-level `project`. |
| Q2 | Human output for `walk` / `neighbors` vs mandatory JSON? | **Open** — summary + `--output json` for full payloads. |
| Q3 | Mutations: interactive confirm vs `--yes` only? | **Open** — mirror `namespace delete` / repo destructive verbs. |
| Q4 | Admin `recovery-scan`: ship in v1 or defer? | **Open** — hide behind `project admin` + doc operator-only. |

## Acceptance

- A1. Every **in-scope** route has a CLI mapping + **httptest** covering **happy path** + at least one **404/403-shaped** denial path per verb family.
- A2. Multi-segment slug (`org/project/repo`) works end-to-end in tests (encode path tail exactly as server expects).
- A3. Mutating verbs match confirmation policy ratified in Q3.
- A4. **cli-output-formats** honoured.
- A5. Q-table ratified (incl. Q4).
- A6. Optional live test `CITADEL_TEST_PROJECTGRAPH_LIVE=1`.
