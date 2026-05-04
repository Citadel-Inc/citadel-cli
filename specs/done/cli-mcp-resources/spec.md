# Spec — cli-mcp-resources

| | |
|---|---|
| Status | DONE 032359ZMAY26 — Shipped MCP resources/list, resources/read, prompts/list, prompts/get, citadel-cli `mcp resources` / `mcp prompts`, waitlist parity with tools/call, and automated conformance tests. SDD closeout complete (P2 C2). Remaining operator/NOMAD rows: P0 Q-table sign-off and P2 Claude Desktop smoke — see [specs/HUMAN_BLOCKERS.md §69](../../HUMAN_BLOCKERS.md#69--cli-mcp-resources-nomad-procedural-q-table--claude-desktop-smoke). |
| Authored | 030619ZMAY26 |
| Owner | Bastion (J-3) |
| Carry-forward from | `cli-mcp-tools` Q5 (RATIFIED OOS) + retro line 78: "`cli-mcp-resources` — `resources/list` + `resources/read` verbs." + spec §37: "Resource / prompt browsing (`resources/list`, `prompts/list`). MCP server's resource surface is stubbed today; defer to `cli-mcp-resources` follow-on." |

## Why

`cli-mcp-tools` ships tool-call verbs only. MCP also defines `resources/*` (read-addressable artefacts: spec docs, CLAUDE.md, generated reports) and `prompts/*` (server-supplied templates). Other MCP clients consume these. Citadel's MCP server stubs them today; this spec lights them up so CLI users can browse / read them and other clients (Claude Desktop, IDEs) get the full surface.

## In scope

- Server-side `resources/list` + `resources/read` implementations: serve documentation files, generated reports, and selected substrate-derived artefacts (e.g. namespace inventory).
- Server-side `prompts/list` + `prompts/get`: ship a curated set of common workflows (`commit-message-from-diff`, `spec-scaffold`, `triage-pr`).
- CLI verbs: `citadel mcp resources list`, `citadel mcp resources read <uri>`, `citadel mcp prompts list`, `citadel mcp prompts get <name>`.
- Resource URI scheme: `citadel://<kind>/<id>` (e.g. `citadel://doc/architecture`, `citadel://spec/active/fe-web-editor-wysiwyg`).

## Out of scope

- **Resource subscription** (`resources/subscribe` for change notifications) — pull only at v1.
- **Prompt arguments validation beyond JSON parse** — strict schemas land in a follow-on if abuse is observed.
- **Operator-tunable resource gating** — hard-coded resource set at v1.

## Decision log

| Q | Proposal | Status |
|---|----------|--------|
| Q1 | Resource-set v1: docs + spec files + active-namespace inventory — anything else? | **RE-RATIFIED 032100ZMAY26 — also includes `citadel://kg-search/{query}`** (NOMAD override). v1 ships `citadel://doc/*`, root primers, `citadel://spec/{active,done}/<slug>`, `citadel://inventory/namespaces`, plus a parameterised `citadel://kg-search/{query}` resource that runs cross-namespace KG fulltext gated to the caller's readable namespaces. |
| Q2 | Auth: per-resource ACL vs. all-or-nothing tied to MCP session token? | **RATIFIED** — all-or-nothing at v1; same waitlist gate as `tools/call`. |
| Q3 | Prompt set v1: which workflows? | **RE-RATIFIED 032100ZMAY26 — adds `review-pr` + `summarize-issue`** (NOMAD override). v1 prompts: `commit-message-from-diff`, `spec-scaffold`, `triage-pr`, `review-pr`, `summarize-issue`. |

## Acceptance

- A1. `resources/list` + `resources/read` return non-stub responses.
- A2. `prompts/list` + `prompts/get` return non-stub responses.
- A3. CLI surfaces both via dedicated subcommands.
- A4. Q-table ratified.
