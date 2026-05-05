# Plan — cli-kg-extended

## ORIENT

- **Server:** `internal/api/kgapi/handler.go` — comments at top of file list canonical query strings per route.
- **Architecture narrative:** `citadel/docs/architecture.md` (KG section) — long-form parameter semantics.
- **Existing CLI:** `cmd/kg.go` — `runKgImpact`, `resolveSymbolID` hitting **`/kg/{owner}/symbols`** (note: path may differ from `/api/namespaces/...` — verify at P0 A1).

## RECON checklist (P0 A1 — blocking implementation)

1. For each verb, record **exact** path string passed to `apiclient.Get` (including whether `/api` prefix is present).
2. Capture **`HandleSymbols`**, **`HandleWalk`**, **`HandleFulltext`**, **`HandleDiff`** query-param validation errors from handler source (duplicate into plan appendix).
3. **`HandleKGSearch`**: mandatory `scope=cross-namespace`; forbidden `mode=regex`; document `cursor` encoding.

## Implementation sketch

- Extend `cmd/kg.go` or split `cmd/kg_extended.go` if file size grows — keep package `cmd`.
- Shared **`namespaceSlugFromRepoFlag(cmd)`** aligned with `splitRepoArg` / `-R`.

## Risks

- **Dual mount legends** (`/kg/...` vs `/api/namespaces/.../kg/...`) cause subtle 404s — mitigate with Q2 ratification + single doc table.
