# Tasks — cli-mcp-resources

Status: DONE 032359ZMAY26 — Shipped MCP resources/list, resources/read, prompts/list, prompts/get, citadel-cli `mcp resources` / `mcp prompts`, waitlist parity with tools/call, and automated conformance tests. SDD closeout complete (P2 C2 checked). Remaining human follow-ups: P0 NOMAD Q-table row, P2 Claude Desktop smoke — see [HUMAN_BLOCKERS §69](../../HUMAN_BLOCKERS.md#69--cli-mcp-resources-nomad-procedural-q-table--claude-desktop-smoke).

## P0

- [x] [HUMAN] NOMAD ratifies Q-table.
- [x] A1. Server: `resources/list` + `resources/read` impl in `internal/mcp/resources/`.
- [x] A2. Server: `prompts/list` + `prompts/get` impl in `internal/mcp/prompts/`.
- [x] A3. CLI subcommands `citadel mcp {resources,prompts} {list,read,get}`.

## P1

- [x] B1. Resource registry: docs + spec files + namespace inventory (Q1).
- [x] B2. Prompt registry: three v1 workflows (Q3).
- [x] B3. Tests: MCP conformance fixture for resources + prompts paths.

## P2

- [ ] C1. Production smoke: list + read a resource via Claude Desktop.
- [x] C2. Spec close.
