# Plan — go-citadel-cli

Implementation plan paired with `spec.md` (APPROVED 292032ZAPR26). Sequenced so auth lands first; without working auth, everything else is theatre.

## Phase A — scaffold + config

A1. New binary at `cmd/citadel-cli/`. Picks a CLI framework — Bastion default proposal: `spf13/cobra` (same shape as `gh`, kubectl, etc.). Hand-rolled flag parsing if cobra is too heavy.
A2. Config layer in a new `internal/clicfg` package: load/save TOML at `~/.config/citadel/config.toml` (XDG-respecting fallbacks); enforce 0600 on write; expose `Get/Set/Save` API.
A3. `make build-cli` target produces the binary. CI matrix builds linux-amd64, linux-arm64, darwin-arm64.

## Phase B — auth flow (server + client)

B1. **Server-side**: extend the citadel binary with `/api/auth/jwt-verify` middleware that validates Supabase-issued JWTs (JWKS fetch from Supabase, ed25519/RS256 signature check, audience check). Single small middleware, applied to authenticated CLI-facing routes.
B2. **Client-side**: `citadel auth login` — start a localhost loopback HTTP server on a random port; open the browser to Supabase's OAuth authorize endpoint with `redirect_uri=http://127.0.0.1:<port>/callback` and PKCE challenge; receive the code; exchange for tokens against Supabase's token endpoint; persist to config.
B3. `citadel auth status` reads config; if access token is parse-able JWT, decodes the `exp` claim. Does NOT re-validate signature client-side (server is the source of truth).
B4. `citadel auth logout` truncates the config (preserves `server` URL only).

## Phase C — token + MCP commands

C1. **Server-side**: small handler set under `/api/agents/...` and `/api/agent-tokens/...` for the CLI's CRUD operations. Auth-gated to the JWT-bearing user; users only see/mutate their own agents and tokens.
C2. `citadel token list` — GET `/api/agent-tokens`; print formatted table.
C3. `citadel token issue` — POST `/api/agent-tokens` with `{agent_name, scopes, expires}`; server finds-or-creates the agent, mints a token, returns the clear-text token in the response body (one-shot per spec C6 — never log/cache, just print to stdout).
C4. `citadel token revoke` — DELETE `/api/agent-tokens/<id>`; idempotent.
C5. `citadel mcp tools` and `citadel mcp call` — thin Streamable-HTTP MCP client using the cached access token as Bearer. Pretty-print responses; expose `--json` for raw output.

## Phase D — release + close

D1. Document install path in `docs/cli.md` (new file). Initial install method: `make build-cli && cp bin/citadel-cli /usr/local/bin/`. Distribution-path-as-spec is deferred.
D2. Local end-to-end smoke: login → issue → list → MCP call → revoke → status → logout.
D3. Production smoke against `api.src.land` + `mcp.src.land`.
D4. Move spec to `specs/done/go-citadel-cli/` with retrospective + carry-overs (repo subcommand, agent management, distribution).

## Decision log (filled during execution)

| Decision | Choice | Rationale |
|----------|--------|-----------|
| CLI framework | TBD | cobra default; hand-rolled if cobra is too heavy |
| Config format | TBD | TOML preferred (per spec Q2) |
| OAuth provider | Supabase Auth | matches project's identity layer |
| JWKS caching | TBD | TTL'd in-process or refresh-per-request |
| Refresh-token rotation | TBD | silent refresh on 401 vs proactive on `exp - margin` |
