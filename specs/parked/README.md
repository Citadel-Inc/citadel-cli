# Parked specifications

Specs in **`specs/parked/`** are **intentionally not pursued**. They are neither in-flight (`specs/active/`) nor delivered (`specs/done/`).

Use **parked** when:

- The idea was scoped and written up, but a **product or architecture decision** retires it (often **superseded** by a simpler canonical path).
- Work should remain **discoverable** for history and rationale, without implying a backlog commitment.

This is the repo-local answer until `@rethunk/citadel-sdd` grows an automated **`spec_park`** (or equivalent) tool; moves into this directory are **hand-edited** and should be called out in commit messages.

## Program decision — HTTPS MCP is canonical

**Customers and agents integrate with Citadel’s MCP over HTTPS** (Streamable HTTP to the Citadel MCP URL, e.g. production `https://mcp.src.land/mcp`), with bearer auth as documented. IDEs and agent hosts should be configured to use that endpoint **directly**.

We are **not** investing in:

- A **stdio** MCP bridge in `citadel-cli` (would duplicate transport and maintenance).
- An **SSE streaming upgrade** path dedicated to long-running MCP tool calls in the CLI client (also parked; long calls remain a timeout / server-design concern).

**Operational note:** Very long tool executions may still need **server or proxy timeout tuning**, **async job patterns**, or product limits—address those on the HTTPS MCP surface, not via a second transport.

## Index

| Slug | Parked | Reason (short) |
|------|--------|----------------|
| [cli-mcp-stdio](cli-mcp-stdio/spec.md) | 050505ZMAY26 | Superseded — HTTPS MCP only; no stdio server in CLI. |
| [cli-mcp-stream](cli-mcp-stream/spec.md) | 050505ZMAY26 | Superseded — no parallel SSE streaming client track; canonical MCP stays HTTPS. |
| [cli-account-avatar](cli-account-avatar/spec.md) | 072200ZMAY26 | Not a dev-loop workflow; no GitHub CLI analogue. Avatar management is a browser/UI concern. |
| [cli-account-privacy](cli-account-privacy/spec.md) | 072200ZMAY26 | Not a dev-loop workflow; no GitHub CLI analogue. Privacy toggles are a settings-panel concern. |
