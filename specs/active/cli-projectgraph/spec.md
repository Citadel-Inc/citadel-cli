# Spec — cli-projectgraph

| | |
|---|---|
| Status | IN_PROGRESS 060539ZMAY26 — Bastion (J-3) claims execution |
| Authored | 050506ZMAY26 |
| Owner | Bastion (J-3) |
| Carry-forward from | Phase 0 / PoL gap analysis (2026-05-05): L2 Project-as-Graph is marquee; server exposes `internal/api/projectgraphapi` under `/api/projectgraph/` but `citadel-cli` had no first-class project verbs (operators had to use MCP or curl). |

## Why

Citadel’s Project-as-Graph engine (`go-projectgraph`) powers pin chains, dependency walks, status rollups, and edge mutations for multi-repo projects. Namespace paths may look like `org/project/repo` or `org/repo`; the discriminated namespace model treats organizations and projects uniformly — **the graph is namespace-scoped**, regardless of whether the slug denotes an org, project container, or repo.

Server companion context: `citadel/specs/active/go-projectgraph/spec.md` (daemon); RBAC atoms **`projectgraph:read`** (browse) and **`projectgraph:manage`** (edge writes / restore / reindex — verify exact atom names in `internal/auth` at implementation time).

## Daemon HTTP contract (authoritative for CLI mapping)

**Mount:** subtree handler at **`/api/projectgraph/`** (`cmd/citadel/main.go` wraps with JWT middleware).  
**Slug encoding:** the server uses a **manual dispatcher** — path tail after `/api/projectgraph/` supports **multi-segment** namespace paths (`org/repo`, nested projects). CLI MUST mirror `decodeSlugPath` semantics: path-unescape, normalize `%2F` → `/`, trim slashes — **do not** assume a single `{slug}` mux segment.

| CLI intent | Method | Path pattern | Notes |
|------------|--------|--------------|--------|
| Pin chain | GET | `…/{slug}/pin-chain` | Namespace resolved from path must be **`kind == repo`** else `400 repo_namespace_required`. Response: JSON array of pin rows (filtered for read). |
| Walk | GET | `…/{slug}/walk` | **Required query:** `kind` — empty → `400 kind_required`. Optional: `max_depth` (positive int; server defaults cap). |
| Neighbors | GET | `…/{slug}/neighbors` | Query: `ns` (optional; defaults from path slug), `kind`, `direction`, `include_deleted=true` for tombstones. |
| Status rollup | GET | `…/{slug}/status-rollup` | See plan appendix for RECON notes; v1 CLI issues plain GET (filters added when daemon stabilizes query contracts). |
| Drilldown | GET | `…/{slug}/status-rollup/drilldown` | Same namespace gate as rollup. |
| Create edge | POST | `…/{slug}/edges` | JSON body `postEdgeBody`: `from_namespace_id`, `from_kind`, `to_namespace_id?`, `to_kind`, `to_external_id?`, `edge_type`, `attrs`, `source` (must be `"manual"` at v1 or `400 manual_source_only`). Requires **`projectgraph:manage`** on **from** namespace. Success: `201` + `{"status":"ok"}`. |
| Delete edge | DELETE | `…/{slug}/edges/{edge_id}` | UUID `edge_id`; resolve `from_namespace_id` for RBAC. |
| Restore edge | POST | `…/{slug}/edges/{edge_id}/restore` | Same RBAC pattern as delete. |
| Reindex | POST | `…/{slug}/reindex` | Ingest hook — CLI confirmation UX mirrors destructive-pattern parity. |

**Opaque deny:** on missing RBAC the API often returns **`404 not_found`** (not `403`) to avoid leaking namespace existence — CLI copy must not promise “does not exist” when read failure may be permission-shaped.

**Rate limits:** per-user bucket (`userLimiter`) — CLI surfaces `429` + `rate_limited` via existing errmap patterns.

## In scope

**Parent command:** **`citadel-cli project`** (top-level; see Q1).

**Read verbs** — implemented before writes.

**Write verbs** — typed confirmation or **`--yes`**; **`PermProjectgraphManage`** paths explain missing scope in user-facing text where practical.

**Recovery scan** — **deferred** (Q4): no CLI surface in v1.

**Cross-cutting**

- **cli-output-formats** on JSON-bearing responses; default human: compact summaries for large arrays (`walk`, `neighbors`) with **`--output json`** for full payloads.
- Large payloads: recommend **`--output json`** + jq in docs.

## Out of scope

- **Replacing git** for submodule operations — users still run `git submodule`; Citadel records pins via API/MCP.
- **Ingest queue HUD** — worker stats belong elsewhere unless exposed as stable JSON.
- **MCP** — HTTPS MCP canonical (`specs/parked/README.md`); this spec is REST via `apiclient` only.
- **Admin recovery-scan** — deferred entirely for v1 (Q4).

## Decision log

| Q | Proposal | Status |
|---|----------|--------|
| Q1 | Top-level `project` vs `namespace project`? | **Ratified 060545ZMAY26** — Top-level **`project`**. Projects remain a first-class Citadel concept; paths may be `org/project/repo` or `org/repo`. The graph operates at the **namespace** level using the discriminated namespace model (org vs project vs repo kinds). |
| Q2 | Human output for `walk` / `neighbors` vs mandatory JSON? | **Ratified 060545ZMAY26** — Summary human preview + **`--output json`** for full payloads. |
| Q3 | Mutations: interactive confirm vs `--yes` only? | **Ratified 060545ZMAY26** — Mirror **`namespace delete`** / repo destructive verbs (typed confirm unless **`--yes`**). |
| Q4 | Admin `recovery-scan`: ship in v1 or defer? | **Ratified 060545ZMAY26** — **Defer entirely** for v1 (no `project admin recovery-scan`). |

## Acceptance

- A1. Every **in-scope** route has a CLI mapping + **httptest** covering **happy path** + at least one **404** denial path per read vs write verb family.
- A2. Multi-segment slug (`org/project/repo`) works end-to-end in tests (encode path tail exactly as server expects).
- A3. Mutating verbs match confirmation policy ratified in Q3.
- A4. **cli-output-formats** honoured.
- A5. Q-table ratified (incl. Q4 deferral).
- A6. Optional live test `CITADEL_TEST_PROJECTGRAPH_LIVE=1` (+ `CITADEL_TEST_PROJECTGRAPH_SLUG`).
