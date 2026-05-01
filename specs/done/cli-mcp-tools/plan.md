# Plan — cli-mcp-tools

## Phase A — transport client

`cmd/citadel-cli/internal/mcpclient/client.go`:
- `Client` struct holds server URL + access token + http.Client.
- `Initialize(ctx)` POSTs `initialize` JSON-RPC; captures returned `Mcp-Session-Id` header.
- `ToolsList(ctx)` POSTs `tools/list`; returns parsed tool descriptors.
- `ToolsCall(ctx, name, args)` POSTs `tools/call`; returns the raw result.

All three round-trip Bearer-token-authed. Errors surface as `mcpclient.Error{ Code, Message }` with the JSON-RPC error code preserved.

## Phase B — verbs

`cmd/citadel-cli/cmd/mcp.go`:
- `tools` subcommand: build client → Initialize → ToolsList → render table.
- `call <tool>` subcommand: parse `--arg` flags into a `map[string]any`; build client → Initialize → ToolsCall → render result.
- `--server` global flag on the `mcp` subcommand group.

## Phase C — argument coercion

Helper `coerceArg(value string) any`:
- All-digits (and optional minus) → int64.
- Decimal → float64.
- `true`/`false` (case-insensitive) → bool.
- Contains comma → split into []string (no nested coercion).
- Otherwise → string.

`--arg-string foo=5` form bypasses coercion; always passes string.

## Phase D — error rendering

`mcpclient.Error` with code `-32001` (unauthorized) → "Run `citadel auth login` to refresh your session." + exit 1.
Code `-32601` (method not found) → "tool not found: <name>" + exit 1.
Other codes → "MCP error <code>: <message>" + exit 1.
Network errors → "could not reach MCP server at <url>: <err>" + exit 2.

## Decision log

| # | Decision | DTG |
|---|----------|-----|
| 1 | Streamable-HTTP only; stdio deferred | 010200Z |
| 2 | Auto-coerce arg types (digit→number, CSV→array); `--arg-string` opt-out | 010200Z |
| 3 | No auto-refresh on auth failure | 010200Z |
| 4 | Default server URL `https://mcp.src.land/mcp` | 010200Z |

## Open questions

See spec.md resolved-questions table. Q1-Q5 await NOMAD ratification.
