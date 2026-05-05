# Spec — cli-projectgraph

| | |
|---|---|
| Status | DRAFT 050506ZMAY26 |
| Authored | 050506ZMAY26 |
| Owner | Bastion (J-3) |
| Carry-forward from | Phase 0 / PoL gap analysis (2026-05-05): L2 Project-as-Graph is marquee; server exposes `internal/api/projectgraphapi` under `/api/projectgraph/` but `citadel-cli` has no first-class project verbs (operators must use MCP or curl). |

## Why

Citadel’s Project-as-Graph engine (`go-projectgraph`) powers pin chains, dependency walks, status rollups, and edge mutations for multi-repo projects. Phase 0 demo narratives (Bastion meta, submodule pins) assume operators can **inspect and advance** graph state without leaving the terminal or hand-crafting JSON-RPC.

The HTTP surface is stable and JWT-gated (`projectgraph:read` / `:write` / `:admin`). The CLI should mirror the **read-heavy** paths first (pin chain, walk, neighbors, status rollup), then writes (edges, restore, reindex) behind explicit flags / confirmations consistent with other mutating verbs.

## In scope

**Parent command:** `citadel-cli project` (alias acceptable if cobra allows; single noun consistent with `repo`, `namespace`).

**Read verbs** (query params match server; slug path is URL-encoded multi-segment namespace):

| Verb | Maps to |
|------|---------|
| `project pin-chain <slug>` | `GET .../{slug}/pin-chain` |
| `project walk <slug>` | `GET .../{slug}/walk` |
| `project neighbors <slug>` | `GET .../{slug}/neighbors` |
| `project status <slug>` | `GET .../{slug}/status-rollup` |
| `project status drilldown <slug>` | `GET .../{slug}/status-rollup/drilldown` |

**Write / operator verbs** (typed confirmation or `--yes` where aligned with `repo delete` patterns):

| Verb | Maps to |
|------|---------|
| `project edge add …` | `POST .../{slug}/edges` |
| `project edge delete …` | `DELETE .../{slug}/edges/{edge_id}` |
| `project edge restore …` | `POST .../{slug}/edges/{edge_id}/restore` |
| `project reindex <slug>` | `POST .../{slug}/reindex` |

**Admin / recovery** (gate behind obvious naming + doc; may require `projectgraph:admin`):

| Verb | Maps to |
|------|---------|
| `project admin recovery-scan` | `POST .../admin/recovery-scan` |

**Cross-cutting**

- Slug argument accepts org/project paths with `/`; pass-through URL encoding per server dispatcher (`/api/projectgraph/` prefix).
- List-style responses honor **cli-output-formats** (`--output json|yaml|…`). Large graphs default to **json** or concise human summaries (pin-chain as table).
- Pagination/cursors if individual endpoints return `next_cursor` (follow server response shapes in implementation).

## Out of scope

- **Replacing git** for submodule operations — users still run `git submodule`; Citadel records pins via API/MCP.
- **Ingest queue operator HUD** — queue depth belongs to server metrics / admin UI unless a stable JSON endpoint is exposed for CLI later.
- **MCP transport** — HTTPS MCP remains canonical (`specs/parked/README.md` policy); this spec is REST-only via `apiclient`.

## Decision log

| Q | Proposal | Status |
|---|----------|--------|
| Q1 | Top-level `project` vs `namespace project` sub-tree? | **Open** — prefer top-level `project` (short; matches product language). |
| Q2 | Human output for `walk` / `neighbors`: nested text vs always `--json`? | **Open** — summary table + `--json` for full graph. |
| Q3 | Mutations default interactive confirm vs `--yes` only? | **Open** — mirror destructive patterns from `namespace delete`. |

## Acceptance

- A1. Read verbs call live `/api/projectgraph/` routes with correct slug encoding; `make verify` includes handler tests with httptest.
- A2. Mutating verbs require confirmation or `--yes` where applicable.
- A3. RBAC errors surface friendly messages (`403` → permission hint).
- A4. **cli-output-formats** integration on responses that return JSON arrays/objects.
- A5. Q-table ratified.
- A6. Optional live test gated `CITADEL_TEST_PROJECTGRAPH_LIVE=1`.
