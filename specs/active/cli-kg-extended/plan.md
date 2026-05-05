# Plan — cli-kg-extended

## ORIENT

- Existing: `cmd/kg.go` — `kg impact` uses `GET /kg/{owner}/impact` (apiclient path style — verify whether base URL prepends `/api` via client).
- Server: `internal/api/kgapi/handler.go` registers routes on `/api/...` — confirm `apiclient` path conventions in `internal/apiclient` (prefix `/api` or raw).

## RECON

Registered routes:

- `GET /api/kg/search`
- `GET /api/namespaces/{slug}/kg/symbols`
- `GET /api/namespaces/{slug}/kg/files`
- `GET /api/namespaces/{slug}/kg/walk`
- `GET /api/namespaces/{slug}/kg/impact` (existing)
- `GET /api/namespaces/{slug}/kg/fulltext`
- `GET /api/namespaces/{slug}/kg/diff`
- `GET /api/kg/diff` (slug query param variant)

## Implementation notes

- Share flag helpers with `kg impact` for `--repo`, `--json`, depth where relevant.
- For `fulltext`/`diff`, expose query params verbatim with sane defaults documented in `--help`.

## Risks

- **Path prefix drift**: if CLI today calls `/kg/` without `/api`, align all kg verbs to one convention to avoid subtle bugs.
