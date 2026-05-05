# Spec — cli-kg-extended

| | |
|---|---|
| Status | DRAFT 050506ZMAY26 |
| Authored | 050506ZMAY26 |
| Owner | Bastion (J-3) |
| Carry-forward from | Phase 0 gap analysis: `citadel-cli kg impact` (+ symbol resolution) covers only impact analysis; `internal/api/kgapi` exposes search, symbols, files, walk, fulltext, diff — operators lack CLI parity. |

## Why

The knowledge-graph query API (`kgapi`) implements cross-namespace search, symbol/file listings, dependency walks, fulltext, and structural diff. PoL appendix K lists multiple `kg.*` MCP tools; **HTTPS MCP is canonical** for agents, but humans and scripts still benefit from **`citadel kg …`** without composing `mcp call`.

This spec **extends** the existing `kg` command tree; it does not replace `kg impact`.

## In scope

Add subcommands under **`citadel-cli kg`**:

| CLI verb | HTTP surface (citadel) |
|----------|-------------------------|
| `kg search --query …` | `GET /api/kg/search?q=…&scope=…&path_prefix=…&language=…&limit=…&cursor=…` |
| `kg symbols [<owner>[/<repo>]] --query …` | `GET /api/namespaces/{slug}/kg/symbols?…` |
| `kg files [<owner>[/<repo>]] …` | `GET /api/namespaces/{slug}/kg/files?…` |
| `kg walk [<owner>[/<repo>]] …` | `GET /api/namespaces/{slug}/kg/walk?…` |
| `kg fulltext [<slug>] …` | `GET /api/namespaces/{slug}/kg/fulltext?…` |
| `kg diff …` | `GET /api/namespaces/{slug}/kg/diff?…` **or** `GET /api/kg/diff?slug=…&…` (two entry points exist) |

**Cross-cutting**

- Reuse **repo resolution** from `cli-cwd-context` (`-R`, `CITADEL_REPO`, origin inference) where the API is namespace/repo scoped.
- **cli-output-formats** on all commands; default human summaries where feasible (search + symbols may table well).
- Document caps (`WalkMaxDepth`, pagination cursors) referencing server limits from `kgapi` constants.

## Out of scope

- **`kg.decision_log` / vector “decision” narrative** — no matching REST handler found in `kgapi` at RECON time; defer until server exposes an endpoint or MCP-only remains documented elsewhere.
- **Changing impact behaviour** — stays as implemented; only additive subcommands.
- **Admin KG mode / reindex** — operator surfaces belong to server/admin specs, not this CLI spec.

## Decision log

| Q | Proposal | Status |
|---|----------|--------|
| Q1 | Flat `kg search` vs `kg query search`? | **Open** — flat subcommands (`kg search`, `kg symbols`, …). |
| Q2 | Namespace slug for cross-namespace search: required flag vs default from context? | **Open** — `--namespace` when needed; `/api/kg/search` may omit slug. |

## Acceptance

- A1. Each in-scope endpoint has a CLI mapping + httptest coverage.
- A2. `make verify` passes.
- A3. `docs/cli.md` documents new verbs + examples.
- A4. Q-table ratified.
- A5. Optional live test `CITADEL_TEST_KG_EXTENDED_LIVE=1`.
