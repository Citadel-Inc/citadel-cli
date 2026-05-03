# Spec — cli-mcp-stdio

| | |
|---|---|
| Status | DRAFT 030619ZMAY26 |
| Authored | 030619ZMAY26 |
| Owner | Bastion (J-3) |
| Carry-forward from | `cli-mcp-tools` retro line 79: "`cli-mcp-stdio` — stdio transport for embedding in other tooling." + spec §95: "Stdio transport (out of scope per spec; tracked as `cli-mcp-stdio` carry-forward)." |

## Why

`cli-mcp-tools` ships HTTP-transport MCP client. Stdio transport is the standard MCP embedding contract — Claude Desktop, IDE integrations, language SDKs all spawn an MCP server on stdio. Citadel exposing only HTTP closes us out of that ecosystem. This spec adds a stdio entrypoint that wraps the existing MCP server.

## In scope

- `citadel mcp serve --stdio` — process speaks JSON-RPC framed messages over stdin/stdout per MCP transport spec.
- Auth: Citadel API token via `CITADEL_TOKEN` env (no interactive auth in stdio mode).
- Reuses the in-process MCP server (`go-mcp-server`); transport switch is the only delta.
- Stderr is the log channel; stdout is reserved for protocol frames only.

## Out of scope

- **Bidirectional sampling** (server-initiated `sampling/createMessage`) — depends on parent client capability advertisement; defer.
- **Multi-session over a single stdio process** — one client per process.
- **Windows named-pipe transport** — stdio only at v1.

## Decision log

| Q | Proposal | Status |
|---|----------|--------|
| Q1 | Token source: env-only vs. also a `--token` flag? | **Open** — env-only (avoids token in process listing). |
| Q2 | Logging level on stderr: `info` default vs. `warn`? | **Open** — `warn` to keep parent client logs clean. |
| Q3 | Graceful shutdown on EOF on stdin: drain in-flight tools or hard-exit? | **Open** — drain with a configurable timeout (default 5s). |

## Acceptance

- A1. `citadel mcp serve --stdio` accepts MCP initialisation + tool calls per transport spec.
- A2. Stdout carries only protocol frames; logs land on stderr.
- A3. Authenticated via `CITADEL_TOKEN` env.
- A4. Q-table ratified.
