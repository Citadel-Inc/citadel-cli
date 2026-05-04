# Tasks — cli-mcp-resources

Status: DONE 032359ZMAY26 — Shipped MCP resources/list and resources/read for citadel:// URIs (docs, specs, namespace inventory), prompts/list and prompts/get with three v1 workflows, citadel-cli mcp resources and prompts subcommands, waitlist parity with tools/call, and conformance tests. Production smoke (Claude Desktop) and formal NOMAD Q-table sign-off remain open.

## P0

- [ ] [HUMAN] NOMAD ratifies Q-table.
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
