# Spec — cli-mcp-tools

| | |
|---|---|
| Status | **DONE 010300ZMAY26** — shipped; HUMAN follow-up = positive prod smoke with real JWT |
| Authored | 010200ZMAY26 |
| Owner | Bastion (J-3) |
| Carry-forward from | `go-citadel-cli` (task C5 carry-forward) |

## Why

`go-citadel-cli` shipped auth + token CRUD but explicitly carried `citadel mcp tools` + `citadel mcp call` forward (task C5, struck in close-out). The MCP server (`go-mcp-server`, live at `https://mcp.src.land/mcp`) advertises tools today but the only way to invoke them is `npx @modelcontextprotocol/inspector` — fine for human exploration, painful for scripting. Operators want to call `citadel mcp call get_namespace --arg slug=damon` from the shell or a CI step.

This spec lands the two CLI verbs against the existing MCP transport.

## In scope

### Verbs

- `citadel mcp tools` — calls `tools/list` against the configured MCP endpoint, prints a table of tool name + description + arg schema.
- `citadel mcp call <tool> [--arg key=value ...] [--json]` — calls `tools/call` with the named tool + parsed args; pretty-prints the result, or emits raw JSON with `--json`.
- `--server <url>` flag overrides the default `https://mcp.src.land/mcp` (e.g. for local dev: `http://localhost:8080/mcp`).

### Auth

- Reuses the existing `~/.config/citadel/config.toml` access token from `citadel auth login`. Bearer-token-injects on every Streamable-HTTP request.
- On 401 / `-32001 unauthorized`, surfaces "Run `citadel auth login` to refresh your session." and exits 1.

### Transport

- Streamable-HTTP client over `net/http` + Bearer header. Honors the existing `Mcp-Session-Id` header for session correlation.
- Single round-trip per call — no SSE upgrades needed for v1 (MCP server's `GET /mcp` is 405-stub anyway).

## Out of scope

- **Tool argument validation** beyond JSON parse. The server enforces schemas; the CLI does not pre-validate.
- **Resource / prompt browsing** (`resources/list`, `prompts/list`). MCP server's resource surface is stubbed today; defer to `cli-mcp-resources` follow-on.
- **Session re-use across invocations.** Each CLI call opens a fresh session (`Mcp-Session-Id` not persisted between processes). Re-use is a complication v1 doesn't need.
- **Stdio MCP transport / CLI SSE streaming upgrades.** **Parked** — HTTPS MCP is canonical; see `specs/parked/README.md` and `specs/parked/cli-mcp-stdio/`, `specs/parked/cli-mcp-stream/`.

## Acceptance criteria

A1. `citadel mcp tools` against the production server returns the live tool list (currently `get_namespace`, `resources/list`, `resources/read` + new ones as `go-kg-impact` ships).
A2. `citadel mcp call get_namespace --arg path=damon` returns the matching namespace JSON.
A3. `--json` flag emits raw `tools/call` response (full JSON-RPC envelope).
A4. Missing token → exit 1 + "Run `citadel auth login`" message; no panic, no stack trace.
A5. Invalid tool → server-side `-32601` rendered as "tool not found: <name>"; exit 1.
A6. `--arg foo=bar` parses as `{"foo": "bar"}`; `--arg count=5` parses as `{"count": 5}` (numeric coercion when the value is digit-only); `--arg list=a,b,c` parses as `{"list": ["a","b","c"]}` (CSV → array).
A7. `--server` override works against `http://localhost:8080/mcp` for dev; default points at production.

## Constraints

C1. **Reuse `internal/clicfg`.** No new config storage; access-token comes from the existing TOML.
C2. **No new HTTP client.** Use `net/http` directly; no third-party JSON-RPC lib.
C3. **No retry on auth failure.** The CLI does not auto-refresh the token; surfaces the error and exits.
C4. **Streamable-HTTP only** for the `citadel-cli mcp` client. Stdio bridge and dedicated SSE streaming upgrades are **out of product scope** and **parked** (`specs/parked/`).

## Risks

R1. **Tool-argument coercion ambiguity.** `--arg count=5` becomes `5` (number), but a tool wanting the string `"5"` gets a type mismatch. Mitigation: `--arg-string foo=5` form forces string; document in `docs/cli.md`.
R2. **Long-running tool calls.** v1 has no SSE / streaming response; a tool that takes 30s blocks the CLI. Mitigation: 60s timeout default + `--timeout` flag override. A dedicated SSE streaming track for MCP was **parked** (`specs/parked/cli-mcp-stream/`); address very long calls via server design, proxy timeouts, or async jobs on the HTTPS MCP surface.
R3. **Server-version drift.** MCP server bumps protocol version (`2025-11-25` today); old CLI binaries may break. Mitigation: include the version in every initialize round-trip; surface a clear "version mismatch" error rather than a generic parse failure.

## Resolved questions

| # | Question | Proposed default | NOMAD |
|---|----------|-------------------|-------|
| Q1 | Default server URL | **`https://mcp.src.land/mcp`** | RATIFIED 010230Z |
| Q2 | Argument coercion | **Auto: digits→number, CSV→array, else string** | RATIFIED 010230Z |
| Q3 | Token-refresh on auth failure | **No** — surface error + exit | RATIFIED 010230Z |
| Q4 | Output format | **Pretty by default; `--json` for raw** | RATIFIED 010230Z |
| Q5 | Resource browsing | **Out of scope; `cli-mcp-resources` follow-on** | RATIFIED 010230Z |

Token model clarification (010230Z): default Bearer = `cfg.AccessToken` (Supabase JWT from `citadel auth login`); `--token` / `CITADEL_AGENT_TOKEN` override for agent / CI use. Both work — the MCP server's `verifyBearer` (per `go-mcp-oauth` A2) tries JWT first then falls through to `agent_tokens`.

## Carry-forward

- (none) — `cli-mcp-resources` shipped (`specs/done/cli-mcp-resources/`). Alternate MCP transports (`cli-mcp-stdio`, `cli-mcp-stream`) are **parked** under `specs/parked/`; HTTPS MCP remains canonical per `specs/parked/README.md`.

## Retrospective (010300Z)

**What landed.** `internal/mcpclient` package (Initialize / ToolsList / ToolsCall + typed Error / Kind enum + 6 unit tests). `cmd/citadel-cli/cmd/mcp.go` rewired on top of the client (token precedence: `--token` > `CITADEL_AGENT_TOKEN` > `cfg.AccessToken`; `coerceArg` with leading-zero-preservation; `--arg-string` opt-out; `--timeout` default 60s; auth-failure copy points at `citadel auth login`). `docs/cli.md` MCP section authored end-to-end. Smokes: local + production reachability confirmed via curl with bogus Bearer (→ -32001 + HTTP 401, mapping to KindUnauthorized).

**Latent bug fixed.** Original `cmd/mcp.go` never issued `initialize`, so every call against the live server would 400 with `Mcp-Session-Id header required`. The unit-test suite would never have caught this against the real protocol since the original code didn't reach the server's session-validation branch. The mcpclient.Initialize() handshake is the load-bearing fix; tests cover it directly.

**Token-model deviation.** Original `mcp.go` deliberately rejected `cfg.AccessToken` (citing waitlist + OIDC concerns). Post-go-mcp-oauth A2 the MCP server's `verifyBearer` accepts JWTs first then opaque tokens; the docstring's premise no longer held. Spec ratification clarified this with a token-model note and the new code uses JWT-by-default with agent-token override.

**HUMAN follow-up.** Positive production smoke (real JWT → `tools/list` → `tools/call get_namespace --arg path=damon`) requires interactive `citadel auth login` against the live Supabase, which the agent cannot drive. Tracked in `specs/HUMAN_BLOCKERS.md` §19 (see also **§08** for go-mcp-oauth inspector / consent flows).

**Did NOT do.**
- Token auto-refresh on 401 (Q3 ratified: out of scope).
- SSE / streaming response (R2: 60s timeout + `--timeout`; streaming upgrade **parked** — `specs/parked/cli-mcp-stream/`).
- Stdio transport (**parked** — `specs/parked/cli-mcp-stdio/`).
- Resource / prompt browsing (Q5 ratified: `cli-mcp-resources` follow-on).
