# Spec — go-citadel-cli

| | |
|---|---|
| Status | **APPROVED** by NOMAD 292032ZAPR26 (in-line with the B-track ratifications) |
| Authored | 292032ZAPR26 |
| Approved | 292032ZAPR26 |
| Owner | Bastion (J-3) |
| Dossier refs | XIII (architecture), `gh`-style CLI surface |
| Parallel-safe | Yes — touches a new package and a new build target. No file overlap with the other active specs. |

## Why

`go-mcp-server` and any future authenticated surface need a way for humans to authenticate and mint agent tokens that doesn't require `psql` access. The `gh` CLI is the closest model — a thin client that owns auth + a small set of operational verbs, not a kitchen sink. This spec lands the smallest CLI shape that unblocks `go-mcp-server` (token issuance) and gives operators a real authentication path.

## Scope

### In scope

- A new binary at `cmd/citadel-cli/`, distinct from the daemon at `cmd/citadel/`. Built static, single binary, distributed by build artefact.
- Subcommands (modeled on `gh`):
  - `citadel auth login` — opens an OAuth/PKCE flow against Supabase Auth (via the existing `auth.users` identity layer); stores the resulting access + refresh token in a config file under `~/.config/citadel/` (mode 0600).
  - `citadel auth status` — prints whether a session is active, the bound user UUID, the access-token expiry, and the configured server URL.
  - `citadel auth logout` — clears the local config.
  - `citadel token list` — lists `agent_tokens` for agents owned by the authenticated user; columns: token id, agent name, scopes, created_at, expires_at, revoked_at.
  - `citadel token issue --agent <name> [--scopes <...>] [--expires <duration>]` — finds-or-creates an `agents` row under the authenticated user with the given name, mints a new `agent_tokens` row, prints the clear-text token ONCE on stdout (sha256 stored in DB).
  - `citadel token revoke <token-id>` — sets `revoked_at` on the row; idempotent on already-revoked.
  - `citadel mcp tools` — calls `tools/list` against the configured MCP server using the active access token; prints tool names + descriptions.
  - `citadel mcp call <tool> [--arg key=value ...]` — invokes `tools/call`; pretty-prints the response.
- Server URL resolution: defaults to `https://api.src.land`; overridable via `CITADEL_SERVER` env var or `--server` flag.
- Config: `~/.config/citadel/config.toml` (or JSON; pick in plan.md). Fields: server URL, access token, refresh token, token expiry, user UUID. Mode 0600.

### Out of scope

- Repo / namespace operations (`citadel repo create`, `citadel namespace ls`, etc.) — defers to `go-citadel-cli-repo` follow-on once gitwire and the namespace API surface are real.
- Agent management beyond find-or-create-on-issue (`citadel agent list/delete`, etc.) — defers to a follow-on.
- Configuration UX beyond the basic config file (no `citadel config set`).
- Shell completions — defer until the surface is stable enough to be worth maintaining.
- Plugin / extension architecture (à la `gh extension`).
- Windows-specific UX polish (works on Windows; not a primary target).
- **Citadel as third-party OAuth provider** ("Login with src.land" on external sites). Future spec — uses Supabase's OAuth-Server primitives once they're in scope; NOMAD 292218ZAPR26 confirmed this is the eventual public direction. Does not affect v1 CLI shape (which consumes Supabase Auth as an OAuth client; the future server posture is additive, not breaking).

## Acceptance criteria

A1. `citadel auth login` opens a browser to the Supabase OAuth endpoint with PKCE, completes the round-trip, and writes a 0600 config.

A2. `citadel auth status` after a successful login prints user UUID + access-token expiry; after `citadel auth logout` prints "not authenticated".

A3. `citadel token issue --agent test1 --scopes mcp:read` prints exactly one line of clear-text token (no debug noise) on stdout, exits 0; the token validates against `https://mcp.src.land/`.

A4. `citadel token list` shows the just-issued token; `citadel token revoke <id>` sets `revoked_at`; subsequent MCP calls with that token return the protocol's unauthorised error.

A5. `citadel mcp tools` against `https://mcp.src.land/` lists the three tools shipped by `go-mcp-server`.

A6. The CLI never logs token clear-text. After `token issue` prints the token, the value is forgotten — re-running `token list` shows only the metadata, not the secret.

A7. CI builds the CLI for linux-amd64, linux-arm64, darwin-arm64; release artefacts go to `bin/`.

## Constraints

C1. **Refresh-token rotation must work.** Access tokens expire in minutes; the CLI silently refreshes on stale-token detection during a command. If refresh fails, prompt the user to `citadel auth login` again.

C2. **Config file is 0600 always.** Any code path that creates or rewrites the config file MUST `chmod 0600`. Verified in tests where possible.

C3. **No CLI flag accepts a literal token on the command line.** Tokens come from `auth login` flow only; no `--token <value>` shortcut that would land in shell history. (Env-var override `CITADEL_TOKEN` is acceptable for CI use; that path doesn't hit shell history.)

C4. **`gh`-shape command structure.** Verb-noun-target where applicable (`citadel auth login`, `citadel token issue`). One subcommand level deep before the action; consistent across the tree.

C5. The CLI is a thin client. Validation, business logic, scope enforcement all live server-side; the CLI never trusts its own state.

## Risks

R1. **Server-side OAuth surface is the load-bearing dep.** This spec assumes Supabase Auth's OAuth endpoints exist (they do — Supabase ships them) and that the citadel server can verify Supabase-issued JWTs. The verification path is small (JWKS fetch + signature check) but counts as scope creep into the server. Mitigation: lay it down here as a small server-side change rather than a new spec; document in plan.md.

R2. **The CLI duplicates state with the server (cached agent list, cached MCP tool list).** Default posture: zero client-side caching; every command hits the server. A future "fast mode" with cache lands its own spec.

R3. **Distribution mechanism.** GitHub Releases vs Homebrew tap vs static download from src.land. Not in scope to pick today; the binary builds, `make build-cli` produces it, distribution lands as a follow-on.

## Resolved questions (decision log)

| # | Question | Decision | Source |
|---|----------|----------|--------|
| Q0 | Spec exists? | **Yes** — required by `go-mcp-server` Q3 token-issuance deferral. NOMAD authored the directive 292032ZAPR26. | NOMAD 292032ZAPR26 |
| Q1 | CLI surface? | `auth login/status/logout`, `token list/issue/revoke`, `mcp tools/call`. Everything else deferred. | NOMAD 292032ZAPR26 |
| Q2 | Config format? | TBD — Bastion picks during Phase A (TOML preferred for human readability + good Go libs). | Bastion-delegated |
| Q3 | OAuth provider? | Supabase Auth (already the project's identity layer). PKCE flow. | Bastion default; NOMAD overrides if otherwise |
