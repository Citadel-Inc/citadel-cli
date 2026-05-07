# Tasks — go-citadel-cli

Status: **DONE 292032ZAPR26** — shipped; all P0/P1/P2 closed.

## P0

- [x] A1. Scaffold `cmd/citadel-cli/`; pick CLI framework (cobra default).
- [x] A2. `internal/clicfg` package — TOML config at `~/.config/citadel/config.toml`, 0600 enforced on every write.
- [x] A3. `make build-cli` target; CI matrix builds linux-amd64 / linux-arm64 / darwin-arm64 / windows-amd64.

## P1

- [x] B1. Server-side JWT-verify middleware — JWKS fetch from Supabase, signature + audience check.
- [x] B2. `citadel auth login` — loopback HTTP receiver + PKCE flow against Supabase OAuth + token persist.
- [x] B3. `citadel auth status` — decode `exp` claim from cached access token.
- [x] B4. `citadel auth logout` — config truncation preserving server URL only.
- [x] C1. Server-side `/api/agents/...` + `/api/agent-tokens/...` handlers, JWT-auth gated.
- [x] C2. `citadel token list` — GET + table format.
- [x] C3. `citadel token issue --agent <name> [--scopes ...] [--expires ...]` — find-or-create agent + mint token + one-shot clear-text print.
- [x] C4. `citadel token revoke <id>` — DELETE; idempotent.
- [ ] ~~C5. `citadel mcp tools` + `citadel mcp call <tool> --arg k=v` — Streamable-HTTP client over Bearer access token.~~ — carry-forward to `cli-mcp-tools` follow-on.

## P2

- [x] D1. Document install path in `docs/cli.md`.
- [x] D2. Local end-to-end smoke (login → issue → list → MCP call → revoke → logout). MCP-call leg deferred with C5; rest exercised live.
- [x] D3. Production smoke against `api.src.land` + `mcp.src.land`. Bearer-rejection paths verified end-to-end (mcp-server D2 evidence).
- [x] D4. Move spec to `specs/done/` with retrospective.
- [ ] ~~D5. Draft follow-on `go-citadel-cli-repo` spec for repo / namespace / agent CRUD verbs (carry-over from spec §Out of scope).~~ — carry-forward to future-spec batch.
