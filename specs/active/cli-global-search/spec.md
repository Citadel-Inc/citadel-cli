# Spec — cli-global-search

| | |
|---|---|
| Status | DRAFT 050506ZMAY26 |
| Authored | 050506ZMAY26 |
| Owner | Bastion (J-3) |
| Carry-forward from | Discovery gap: dashboard Cmd-K search exists server-side (`internal/api/searchapi`) but CLI cannot drive it. |

## Daemon HTTP contract

### Authenticated search — `GET /api/search`

- **Auth:** JWT required (`requireUser` — **401** without claims).
- **Query parsing:** `parseSearchInput(w, r, scoped=true)` (`handler.go`):

| Param | Rule |
|-------|------|
| `q` | **≥ 2 Unicode code points** — else **`400 query_too_short`**. |
| `scope` | `namespaces` \| `repos` \| `all` (default **`all`**). Invalid → **`400 invalid_scope`**. |
| `limit` | Int ≥ 1, capped at **`maxLimit = 25`** server-side; default **`defaultLimit = 10`**; `0`/bad int → **`400 invalid_limit`**. |

**Response**

```json
{"query":"…","scope":"…","results":[{"type":"…","id":"…","slug":"…","kind":"…","parent_slug":"…","display_name":"…","path":"…","score":0,"avatar_url":{…},"gravatar_hash":"…"}, …]}
```

### Public namespaces — `GET /api/search/namespaces/public`

- **Auth:** **None** — handler **does not** call `requireUser` (`parseSearchInput(..., scoped=false)`).
- Same **`q`** length rule (**≥ 2** runes).
- Response: `{"query":"…","results":[{"slug":"…","kind":"…","avatar_url":…,"gravatar_hash":…}]}`.

**Mounting note:** `cmd/citadel/main.go` registers **public** route with **`searchAPI.Routes()` only** (no JWT wrapper) — CLI **`search --public`** can run **without** token **if** we allow unauthenticated client — today **`apiclient.New` requires token** → Q-table must resolve (**optional auth path** vs **dummy anon JWT** impossible).

## In scope

| CLI | Behaviour |
|-----|-----------|
| `search <query>` | Authenticated search — default `scope=all`. |
| Flags | `--scope namespaces|repos|all`, `--limit` (respect server cap), `--json` etc. |
| Public mode | **`search public <query>`** or **`--public`** — hits **`/api/search/namespaces/public`** — **requires implementing token-less GET** (extend `apiclient` with optional auth or raw `http.Client` for this verb only). |

## Out of scope

- **Issue search / KG** — other specs.
- **Frecency / telemetry** endpoints.

## Decision log

| Q | Proposal | Status |
|---|----------|--------|
| Q1 | How to call **public** search given CLI requires JWT today? | **Open** — (a) relax client for this path only; (b) document “requires login” if server later gates — **must verify**. |
| Q2 | Subcommand vs `--public` flag? | **Open** — `--public` keeps one entrypoint. |

## Acceptance

- A1. Authenticated search works + tests for **`query_too_short`**, **`invalid_scope`**, **`invalid_limit`**.
- A2. Public search: resolution per Q1 + documented behaviour.
- A3. Table output maps **`result.type`** sensibly.
- A4. Q-table ratified.
- A5. Optional `CITADEL_TEST_SEARCH_LIVE=1`.
