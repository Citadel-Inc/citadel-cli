# Specs

This directory holds [SDD](https://github.com/Rethunk-AI/citadel-sdd) specifications for `citadel-cli`.

## Layout

- `active/` — open specs (DRAFT, APPROVED, IN_PROGRESS, BLOCKED)
- `done/` — completed specs (DONE)

## Lifecycle

Use the `mcp__citadel-sdd__*` MCP tools for all spec lifecycle operations (claim, approve, close, block, task-check). Never hand-edit status fields, DTG stamps, or `tasks.md` checkbox state lines — the tools enforce lint rules and stamp accurate timestamps.

See [`citadel-sdd/docs/mcp-tools.md`](https://github.com/Rethunk-AI/citadel-sdd/blob/main/docs/mcp-tools.md) for the full tool reference.
