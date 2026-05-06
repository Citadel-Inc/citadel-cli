# Plan — cli-kg-extended

## ORIENT

- **Server:** `internal/api/kgapi/handler.go` — comments at top of file list canonical query strings per route.
- **Architecture narrative:** `citadel/docs/architecture.md` (KG section) — long-form parameter semantics.
- **Existing CLI:** `cmd/kg.go` — `runKgImpact`, `resolveSymbolID` hit **`/api/namespaces/{slug}/kg/...`** (extended verbs live in `cmd/kg_extended.go`).

## RECON checklist (P0 A1 — blocking implementation)

1. For each verb, record **exact** path string passed to `apiclient.Get` (including whether `/api` prefix is present).
2. Capture **`HandleSymbols`**, **`HandleWalk`**, **`HandleFulltext`**, **`HandleDiff`** query-param validation errors from handler source (duplicate into plan appendix).
3. **`HandleKGSearch`**: mandatory `scope=cross-namespace`; forbidden `mode=regex`; document `cursor` encoding.

## Implementation sketch

- Extend `cmd/kg.go` or split `cmd/kg_extended.go` if file size grows — keep package `cmd`.
- Shared **`namespaceSlugFromRepoFlag(cmd)`** aligned with `splitRepoArg` / `-R`.

## Risks

- **Dual mount legends** (`/kg/...` vs `/api/namespaces/.../kg/...`) cause subtle 404s — mitigate with Q2 ratification + single doc table.

## Appendix: Final path convention (CLI shipped)

The CLI uses **`GET https://<api-host>/…`** paths **without** an extra `/api` prefix in code strings because the configured server URL already targets the API host (same pattern as `ssh-key`, `oauth clients`, etc.).

| Verb | `apiclient.Get` path |
|------|----------------------|
| Cross-namespace search | `/api/kg/search?scope=cross-namespace&…` |
| Symbols | `/api/namespaces/<ns>/kg/symbols?repo=<repo>&…` |
| Files | `/api/namespaces/<ns>/kg/files?…` |
| Walk | `/api/namespaces/<ns>/kg/walk?seed_id=<uuid>&…` |
| Fulltext | `/api/namespaces/<ns>/kg/fulltext?…` |
| Diff | `/api/namespaces/<ns>/kg/diff?…` |
| Impact | `/api/namespaces/<ns>/kg/impact?symbol=<uuid>&…` |

`<ns>` / `<repo>` derive from **`-R ns/repo`**, **`CITADEL_REPO`**, optional positional **`namespace/repo`**, or git **`origin`** for hosts in **`CITADEL_GIT_HOSTS`** (same helpers as repo commands).

**Examples:** namespace **`acme`**, repo **`billing`** → symbols path `/api/namespaces/acme/kg/symbols?repo=billing&q=…`.

**Note:** Query parameter names for **`kg diff`** (`from_ref` / `to_ref`) follow the HTTP handler’s expected spellings; adjust if the daemon renames them.
