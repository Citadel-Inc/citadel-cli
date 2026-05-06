# Tasks — cli-mcp-tools

Status: **DONE 010300ZMAY26** — all P0/P1/P2 landed; HUMAN follow-up = positive prod smoke against a real authenticated session (operator-side login).

## P0
- [x] [HUMAN] NOMAD ratifies scope + 5 decision-log defaults — RATIFIED 010230Z.
- [x] A1. `cmd/citadel-cli/internal/mcpclient/client.go` — `Initialize` / `ToolsList` / `ToolsCall` over Streamable-HTTP. Captures Mcp-Session-Id; resends on every subsequent call.
- [x] A2. `cmd/citadel-cli/internal/mcpclient/error.go` — typed error mapping JSON-RPC codes (-32700/-32600/-32601/-32602/-32001) + HTTP 401 + version-mismatch.

## P1
- [x] B1. `cmd/citadel-cli/cmd/mcp.go` — `tools` subcommand.
- [x] B2. `cmd/citadel-cli/cmd/mcp.go` — `call <tool>` subcommand with `--arg` parsing.
- [x] B3. `--server` global override on the `mcp` group (inherits root `--server`).
- [x] B4. Coercion helper `coerceArg` (digits→int64, decimals→float64, true/false→bool, CSV→array) + `--arg-string` opt-out. Leading-zero numerics stay as strings to preserve IDs.

## P2
- [x] C1. Update `docs/cli.md` MCP section with verb examples + arg coercion table + flags + exit codes + auth-failure copy.
- [x] C2. Local smoke — `/tmp/citadel mcp tools` (no token → friendly error), `/tmp/citadel mcp tools --token bogus` (→ "unauthorized: run `citadel auth login`"). Confirms transport + auth-gate + Initialize semantics.
- [x] C3. Production smoke partial — `curl https://mcp.src.land/mcp` with bogus Bearer returns `{"error":{"code":-32001,"message":"unauthorized"}}` + HTTP 401. Confirms reachability + auth-gate operative. Positive smoke (real authenticated session → `tools/list` → `tools/call get_namespace`) is HUMAN follow-up since it requires interactive `citadel auth login`.
- [x] C4. Spec close — moved to `specs/done/cli-mcp-tools/`; retrospective in `spec.md`; `specs/README.md` Active → Done.
