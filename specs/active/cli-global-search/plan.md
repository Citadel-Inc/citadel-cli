# Plan — cli-global-search

## ORIENT

- **Server:** `internal/api/searchapi/handler.go` — `parseSearchInput`, `HandleSearch`, `HandlePublicNamespaces`.
- **CLI:** `internal/apiclient` requires a token; ratified policy keeps it that way for every verb including search.

## RECON

- **`GET /api/search`** — JWT + scope/limit/query validation.
- Anonymous **`GET /api/search/namespaces/public`** — out of scope for this CLI per Q1 ratification.

## Implementation

- **`cmd/search.go`:** build query string; default **`scope=namespaces`**; **`--public`** → **`scope=all`** unless **`--scope`** is set; validate query length (≥ 2 runes) and limit range client-side.
- **Tests:** `cmd/handler_test.go` — happy path + `query_too_short` / `invalid_scope` / `invalid_limit` from mock server; optional live gate **`CITADEL_TEST_SEARCH_LIVE=1`**.
- **Docs:** `docs/cli.md` — login + QoS framing; default vs `--public`.

## Risks

- **Marketing vs security:** broader **`scope=all`** still respects server-side RBAC and rate limits; CLI documents opaque **404** behaviour inherited from the API where applicable.
