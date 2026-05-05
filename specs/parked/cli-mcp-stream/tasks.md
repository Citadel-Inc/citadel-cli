# Tasks — cli-mcp-stream

**Spec PARKED 050505ZMAY26** — see [`spec.md`](./spec.md) and [`../README.md`](../README.md). Tasks below are frozen; do not execute. _(Was DRAFT 030619ZMAY26.)_

| | |
|---|---|
| Status | PARKED 050505ZMAY26 — superseded by HTTPS MCP canonical policy ([`../README.md`](../README.md)). |

## P0

- [ ] [HUMAN] NOMAD ratifies Q-table.
- [ ] A1. SSE upgrade in `internal/mcp/transport/http.go`.
- [ ] A2. Tool runtime hook for emitting `progress` events.
- [ ] A3. Heartbeat ticker (Q1).

## P1

- [ ] B1. CLI consumer: detect `streaming: true`, set Accept header, render progress to terminal.
- [ ] B2. Per-tool `streaming` flag in registration metadata (Q2).
- [ ] B3. Tests: SSE round-trip; idle heartbeat; client-disconnect cancels server context.

## P2

- [ ] C1. Smoke: re-index a large KG via streaming MCP call; observe progress events.
- [ ] C2. Spec close.
