# Spec — cli-kg-extended

| | |
|---|---|
| Status | IN_PROGRESS 060503ZMAY26 — Bastion (J-3) claims execution |
| Authored | 050506ZMAY26 |
| Owner | Bastion (J-3) |
| Carry-forward from | Phase 0 gap analysis: `citadel-cli kg impact` (+ symbol resolution via `/…/symbols`) covers impact analysis only; `internal/api/kgapi` exposes broader read APIs documented in `citadel/docs/architecture.md` and `go-kg-query` specs. |

## Why

Knowledge-graph queries power discovery (symbols, files, walks), cross-namespace search, fulltext, and structural diff. Agents integrate via **HTTPS MCP** (canonical); humans and CI still need **`citadel kg …`** without composing raw URLs.

**Important:** Today `kg impact` calls **`/kg/{owner}/impact`** (`cmd/kg.go`). Separately, **`kgapi.Routes`** registers **`/api/namespaces/{slug}/kg/...`** and **`/api/kg/search`**. P0 must **reconcile** these path styles against the configured API host (document outcome in plan appendix — either align legacy `/kg/` short paths or standardise on `/api/namespaces/.../kg/...` for new verbs).

## Daemon HTTP contract (baseline RECON)

Registered in `kgapi.Handler.Routes` (`internal/api/kgapi/handler.go`):

| Endpoint | Purpose |
|----------|---------|
| `GET /api/kg/search` | Cross-namespace fulltext search — **`scope=cross-namespace` required** (`400 invalid_scope` otherwise). Params include `q`, `mode` (`fts` default; **`regex` rejected** with `cross_namespace_regex_unsupported`), `path_prefix`, `language`, `cursor`, `limit`. Empty `q` → `400 query_required`. Returns paginated fulltext shape from `QueryFulltextCrossNamespace`. |
| `GET /api/namespaces/{slug}/kg/symbols` | Symbol lookup — substring `q`, `repo`, `kind`, `path_prefix`, `limit`, `cursor` per `go-kg-query` / architecture doc. |
| `GET /api/namespaces/{slug}/kg/files` | File listing with path prefix + language + cursor. |
| `GET /api/namespaces/{slug}/kg/walk` | Bounded graph walk — depth/direction caps per handler (`WalkMaxDepth`, etc.). |
| `GET /api/namespaces/{slug}/kg/impact` | Already used by **`kg impact`** (may use alternate `/kg/{slug}/impact` alias depending on gateway — reconcile). |
| `GET /api/namespaces/{slug}/kg/fulltext` | Per-namespace fulltext (`mode=fts|regex`, cursor, etc.). |
| `GET /api/namespaces/{slug}/kg/diff` + `GET /api/kg/diff` | Structural diff — slug either in path or `?slug=` query (`diff.go`). |

**RBAC:** handlers enforce **`kg:read`** via resolver with **opaque 404** for private namespaces (matches gitwire deny posture).

## In scope

Add subcommands under **`citadel-cli kg`**:

| CLI verb (proposed) | Maps to |
|---------------------|---------|
| `kg search` | `GET /api/kg/search` — expose `scope=cross-namespace` as default or hidden constant; pass through `q`, `mode`, `path_prefix`, `language`, `cursor`, `limit`. |
| `kg symbols` | `GET /api/namespaces/{slug}/kg/symbols` — slug from `-R` / args / cwd context. |
| `kg files` | `GET /api/namespaces/{slug}/kg/files` |
| `kg walk` | `GET /api/namespaces/{slug}/kg/walk` |
| `kg fulltext` | `GET /api/namespaces/{slug}/kg/fulltext` |
| `kg diff` | `GET /api/namespaces/{slug}/kg/diff` or `/api/kg/diff` — Q-table picks invocation shape for repo-scoped workflows. |

**Cross-cutting**

- **cli-cwd-context:** reuse `-R`, `CITADEL_REPO`, origin inference where slug denotes a repo namespace.
- **cli-output-formats:** json/yaml/ndjson/table as applicable.
- **Pagination:** forward opaque `cursor` + `next_cursor` when server returns them.

## Out of scope

- **`kg.decision_log`** (PoL appendix K) — **no REST handler** located under `kgapi` at authoring time; remains MCP-only until server ships HTTP.
- **Changing `kg impact` behaviour** unless required for path unification (track as separate task in plan).
- **Admin KG** (`/api/admin/kg/*`) — operator tooling.

## Decision log

| Q | Proposal | Status |
|---|----------|--------|
| Q1 | Flat `kg search` vs nested `kg query search`? | **Ratified 060500ZMAY26** — flat verbs: `kg search`, `kg symbols`, `kg files`, `kg walk`, `kg fulltext`, `kg diff` (no `kg query` nesting). |
| Q2 | Standardise all verbs on `/api/namespaces/{slug}/kg/...` vs keep `/kg/{owner}/...` for impact only? | **Ratified 060500ZMAY26** — **`kg impact`** migrates to **`GET /api/namespaces/{slug}/kg/impact`** (same query params as today); symbol resolution uses **`/api/namespaces/{slug}/kg/symbols`**. Legacy `/kg/...` short paths are no longer called by the CLI. |
| Q3 | `kg diff` default: namespace path vs global `/api/kg/diff?slug=`? | **Ratified 060500ZMAY26** — Default **`GET /api/namespaces/{namespace}/kg/diff`** with namespace (+ optional `repo` query) from `-R` / `CITADEL_REPO` / CWD; **`/api/kg/diff?slug=`** reserved for explicit scripting if added later (not the default UX). |

## Acceptance

- A1. Each in-scope endpoint implemented + httptest + errmap coverage for `401`, `404` opaque, `429` if rate limited.
- A2. Plan appendix documents **final path convention** + examples with real slugs.
- A3. `make verify` passes.
- A4. `docs/cli.md` updated with examples for cross-namespace search + symbols.
- A5. Q-table ratified.
- A6. Optional `CITADEL_TEST_KG_EXTENDED_LIVE=1`.
