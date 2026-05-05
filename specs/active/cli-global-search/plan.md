# Plan — cli-global-search

## ORIENT

- Server: `internal/api/searchapi/handler.go` — `HandleSearch`, `HandlePublicNamespaces`.
- Main mounts gate — verify `gateSearch` vs public route auth in `cmd/citadel/main.go` during implementation.

## RECON

- Parse rules from `pure_test.go` URL cases (`q` length, scope enum, limit bounds).
- Response JSON shape: `[]result` with fields from handler.

## Implementation sketch

- New `cmd/search.go`; thin GET wrapper + query param builder.
- Default scope `all` if server supports it.

## Risks

- **Public endpoint without JWT** — if middleware still requires JWT, document that `search --public` needs login for consistency with server reality.
