# Parked specifications

Specs in **`specs/parked/`** are **intentionally not pursued**. They are neither in-flight (`specs/active/`) nor delivered.

Use **parked** when:

- The idea was scoped and written up, but a **product or architecture decision** retires it (often **superseded** by a simpler canonical path).

Moves into this directory are hand-edited and called out in commit messages.

## HTTPS MCP is canonical

**Customers and agents integrate with Citadel’s MCP over HTTPS** (Streamable HTTP to the Citadel MCP URL, e.g. production `https://mcp.src.land/mcp`), with bearer auth as documented. IDEs and agent hosts should be configured to use that endpoint **directly**.

Not built:

- A **stdio** MCP bridge in `citadel-cli` (would duplicate transport and maintenance).
- An **SSE streaming upgrade** path dedicated to long-running MCP tool calls in the CLI client (also parked; long calls remain a timeout / server-design concern).

**Operational note:** Very long tool executions may still need **server or proxy timeout tuning**, **async job patterns**, or product limits—address those on the HTTPS MCP surface, not via a second transport.

Rejected proposals and their reasons: the Parked table in [../README.md](../README.md).
