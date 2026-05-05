# Specs

## Active

| Slug | State | DTG | Owner |
|------|-------|-----|-------|
| cli-audit | DRAFT | 081550ZMAY26 | Bastion (J-3) |
| cli-issue-pr | DRAFT | 081550ZMAY26 | Bastion (J-3) |
| cli-output-formats | DRAFT | 080800ZMAY26 | Bastion (J-3) |
| cli-oauth-login | DRAFT | 075800ZMAY26 | Bastion (J-3) |
| cli-watch | IN_PROGRESS | 050953ZMAY26 | Bastion (J-3) |
| cli-error-format | DRAFT | 050900ZMAY26 | Bastion (J-3) |
| cli-mcp-stdio | DRAFT | 030619ZMAY26 | Bastion (J-3) |
| cli-mcp-stream | DRAFT | 030619ZMAY26 | Bastion (J-3) |

## Done

| Slug | DTG | Note |
|------|-----|------|
| go-citadel-cli | 292032ZAPR26 | shipped (in-line with the B-track ratifications) |
| cli-pagination | 051011ZMAY26 | Server list endpoints and citadel-cli list verbs now support opaque cursor pagination (?limit/?cursor, next_cursor), including members-specific cursors, ndjson streaming under --all, human tail hints, bounded completion fetch, and handler-level multi-page tests. P2 leaves the gated live 250-repo walk and operator production smoke as follow-ups. |
| cli-completion-dynamic | 050935ZMAY26 | Delivered dynamic shell completion with a 60s disk cache under XDG, ValidArgsFunction wiring for repos/namespaces/agents/OAuth client UUIDs/agent tokens, async PostRun invalidation on mutating verbs, static --output completion aligned to cli-output-formats, integration tests plus cache TTL tests, and README/HUMANS documentation including CITADEL_NO_COMPLETION_CACHE. Operator latency smoke (C2) remains for a human with a live namespace. |
| cli-cwd-context | 050915ZMAY26 | Implemented CWD git-origin repo resolution: -R/--repo and CITADEL_REPO, optional inference via git remote get-url origin for Citadel hosts (defaults plus CITADEL_GIT_HOSTS), --no-cwd-repo opt-out, TTY inference hint on stderr (respects --quiet and CI). Wired into repo get/delete and kg impact with tests and README/HUMANS guidance. Operator smoke task C2 left for humans. |
| cli-oauth-clients | 040041ZMAY26 | P1 B3: opt-in live integration test (oauth_clients_live_test.go) + §71 runbook. P2 C1 remains operator citadel-cli smoke — see specs/HUMAN_BLOCKERS.md §71. |
| cli-mcp-resources | 032359ZMAY26 | Shipped MCP resources/list, resources/read, prompts/list, prompts/get, citadel-cli `mcp resources` / `mcp prompts`, waitlist parity with tools/call, and automated conformance tests. SDD closeout complete (P2 C2). Remaining operator/NOMAD rows: P0 Q-table sign-off and P2 Claude Desktop smoke — see [specs/HUMAN_BLOCKERS.md §69](../../HUMAN_BLOCKERS.md#69--cli-mcp-resources-nomad-procedural-q-table--claude-desktop-smoke). |
| go-citadel-cli-repo | 032036ZMAY26 | Shipped repo/namespace/agent CRUD CLI verbs against live APIs. repo create/list/get/delete; namespace list/get/members/transfer (with initiate/list-pending/accept/decline/revoke subcommands); agent list/get/delete/rotate-token. All verbs carry --help + --output json (A1). cmd_test.go integration suite covers command-tree structure, flag presence, and destructive-verb --yes gates (A2). Destructive verbs gate on typed-slug confirm (A3). Q-table ratified (A4). repo rename descoped (no server endpoint). namespace transfer org-only for now; personal namespace transfer deferred to server-side follow-on. |
| cli-mcp-tools | 010300ZMAY26 | shipped; HUMAN follow-up = positive prod smoke with real JWT |
