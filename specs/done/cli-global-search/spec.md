# Spec — cli-global-search

| | |
|---|---|
| Status | DONE 060535ZMAY26 — Shipped top-level `citadel-cli search` with authenticated GET /api/search, default scope=namespaces, --public for scope=all, httptest coverage for query_too_short/invalid_scope/invalid_limit, optional CITADEL_TEST_SEARCH_LIVE=1, and docs/cli.md QoS framing. |
| Authored | 050506ZMAY26 |
| Owner | Bastion (J-3) |
| Carry-forward from | Discovery gap: dashboard Cmd-K search exists server-side (`internal/api/searchapi`) but CLI cannot drive it. |

## Daemon HTTP contract

### Authenticated search — `GET /api/search`

- **Auth:** authenticated user session required (`requireUser` — **401** without claims). The CLI documents that **every** command, including search, requires login so Citadel can enforce per-user rate limits and protect quality of service.
- **Query parsing:** `parseSearchInput(w, r, scoped=true)` (`handler.go`):

| Param | Rule |
|-------|------|
| `q` | **≥ 2 Unicode code points** — else **`400 query_too_short`**. |
| `scope` | `namespaces` \| `repos` \| `all` (default **`all`** on the server if omitted). Invalid → **`400 invalid_scope`**. |
| `limit` | Int ≥ 1, capped at **`maxLimit = 25`** server-side; default **`defaultLimit = 10`**; `0`/bad int → **`400 invalid_limit`**. |

**Response**

```json
{"query":"…","scope":"…","results":[{"type":"…","id":"…","slug":"…","kind":"…","parent_slug":"…","display_name":"…","path":"…","score":0,"avatar_url":{…},"gravatar_hash":"…"}, …]}
```

### Public namespaces — `GET /api/search/namespaces/public`

- **Auth:** **None** on the wire — unauthenticated discovery exists server-side for other clients.
- **CLI policy:** The CLI does **not** call this route. Operators remain logged in; **`--public`** widens authenticated search via **`scope=all`** on **`GET /api/search`** (broader than membership-scoped results), not via the anonymous handler.

## In scope

| CLI | Behaviour |
|-----|-----------|
| `search <query>` | Authenticated **`GET /api/search`** — default CLI **`scope=namespaces`** (user-accessible namespaces first). |
| Flags | `--scope namespaces|repos|all`, `--limit` (1–25; respect server cap), `--public` (implies **`scope=all`** unless `--scope` overrides), `--output` / `--json`. |
| `--public` | Opt-in to broader discovery (unrelated namespaces), still session-authenticated; not the anonymous public HTTP route. |

## Out of scope

- **Issue search / KG** — other specs.
- **Frecency / telemetry** endpoints.

## Decision log

| Q | Proposal | Status |
|---|----------|--------|
| Q1 | How to call **public** search given CLI requires JWT today? | **Ratified 050545ZMAY26** — Document that the CLI requires login for all commands (including search), framed as a service-protection mechanism so Citadel can enforce per-user rate limits and preserve quality of service. Do not ship anonymous `GET /api/search/namespaces/public` from this CLI. |
| Q2 | Subcommand vs `--public` flag? | **Ratified 050545ZMAY26** — **`--public`** flag; search user-accessible namespaces first, then **`--public`** opts in to broader discovery (`scope=all`). |

## Acceptance

- A1. Authenticated search works + tests for **`query_too_short`**, **`invalid_scope`**, **`invalid_limit`**.
- A2. Documented behaviour: login required; `--public` = authenticated wider scope (not anonymous endpoint).
- A3. Table output maps **`result.type`** sensibly.
- A4. Q-table ratified.
- A5. Optional `CITADEL_TEST_SEARCH_LIVE=1`.
