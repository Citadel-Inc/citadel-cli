# Spec — cli-mcp-stream

| | |
|---|---|
| Status | DRAFT 030619ZMAY26 |
| Authored | 030619ZMAY26 |
| Owner | Bastion (J-3) |
| Carry-forward from | `cli-mcp-tools` retro line 80: "`cli-mcp-stream` — SSE upgrade for long-running tool calls." + spec §94 R2 timeout note. |

## Why

`cli-mcp-tools` HTTP transport caps at 60s tool-call timeout (R2). Long-running tools (project-graph walk over a huge repo, KG re-index) blow past that and the client times out. SSE upgrade lets the server stream progress events + final result without holding a connection that triggers the 60s gate.

## In scope

- Server-Sent Events upgrade on `POST /mcp` when client sends `Accept: text/event-stream`.
- Event kinds: `progress` (tool-emitted), `partial-result` (chunked result), `result` (terminal), `error`.
- Client (`cli-mcp-tools`) auto-upgrades when the requested tool is registered with `streaming: true`.
- No timeout cap on streamed tool calls (idle heartbeat every 15s instead).

## Out of scope

- **Bidirectional streaming** — server → client only at v1.
- **Cancellation mid-stream** — client closes the connection; server cancels via context. No `cancel` event verb yet.
- **WebSocket transport** — SSE only (simpler ops, no proxy quirks).

## Decision log

| Q | Proposal | Status |
|---|----------|--------|
| Q1 | Heartbeat interval: 15s vs. 30s? | **Open** — 15s for quick proxy-detection. |
| Q2 | Tool registration: per-tool `streaming: bool` vs. always-eligible? | **Open** — per-tool to keep small tools simple. |
| Q3 | Progress event throttling: cap at 10/sec to avoid flooding? | **Open** — yes; document. |

## Acceptance

- A1. `Accept: text/event-stream` upgrades a tool call to SSE.
- A2. Long-running tools emit progress + terminal result without 60s timeout.
- A3. CLI surfaces progress events to the user (terminal status line).
- A4. Q-table ratified.
