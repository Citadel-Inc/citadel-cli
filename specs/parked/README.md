# Parked specifications

Specs in **`specs/parked/`** are **intentionally not pursued**. They are neither in-flight (`specs/active/`) nor delivered.

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
| ------ | -------- | ---------------- |
| [cli-account-export](cli-account-export/spec.md) | 091233ZMAY26 | Rejected — GDPR data export is a web account-settings surface. |
| [cli-billing](cli-billing/spec.md) | 091233ZMAY26 | Rejected — billing management is a web/dashboard surface. |
| [cli-webhook-test](cli-webhook-test/spec.md) | 091233ZMAY26 | Rejected — no CLI mandate to wrap the webhook test-ping. |
