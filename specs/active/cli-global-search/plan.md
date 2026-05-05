# Plan — cli-global-search

## ORIENT

- **Server:** `internal/api/searchapi/handler.go` — `parseSearchInput`, `HandleSearch`, `HandlePublicNamespaces`.
- **CLI blocker:** `internal/apiclient.New` returns error when **`cfg.AccessToken == ""`** — public search requires an **explicit design** (see spec Q1).

## RECON

- Confirm **`gateSearch`** in `main.go` — authenticated `/api/search` behind JWT.
- Confirm public route has **no JWT** — enables anonymous discovery.

## Implementation options (for Q1 ratification)

1. **`httpx` direct GET** for public endpoint only (no Bearer) — duplicate timeout/trace behaviour cautiously.
2. **`apiclient` extension:** `NewAllowAnonymous(cfg)` or optional token empty only for allow-listed paths (risk: accidental misuse).
3. **Defer public subcommand** until Q1 resolved — ship authenticated search only at v1.

## Risks

- **Marketing vs security:** public namespace enumeration is intentional server-side — CLI should not add extra leaking beyond server.
