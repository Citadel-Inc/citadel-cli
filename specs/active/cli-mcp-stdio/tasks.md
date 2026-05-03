# Tasks — cli-mcp-stdio

Status: DRAFT 030619ZMAY26

## P0

- [ ] [HUMAN] NOMAD ratifies Q-table.
- [ ] A1. `citadel mcp serve` subcommand with `--stdio` flag.
- [ ] A2. Stdio transport adapter wrapping the existing MCP server.
- [ ] A3. Stderr-only log routing.

## P1

- [ ] B1. Graceful shutdown on EOF (Q3).
- [ ] B2. Tests: MCP transport conformance fixture (init, tools/list, tools/call).
- [ ] B3. Claude Desktop config snippet in CLI README.

## P2

- [ ] C1. End-to-end smoke: register `citadel mcp serve --stdio` in Claude Desktop config; invoke a tool from chat.
- [ ] C2. Spec close.
