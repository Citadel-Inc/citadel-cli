# Specs

## Active

| Slug | State | DTG | Owner |
|------|-------|-----|-------|
| cli-issue-pr | DRAFT | 081550ZMAY26 | Bastion (J-3) |
| cli-oauth-login | DRAFT | 075800ZMAY26 | Bastion (J-3) |

## Parked

Specs we are **not pursuing** (superseded, withdrawn, or cancelled). See [`parked/README.md`](parked/README.md) for the **HTTPS MCP canonical** decision.

| Slug | Parked | Note |
|------|--------|------|
| cli-mcp-stdio | 050505ZMAY26 | Stdio MCP bridge — superseded; customers use HTTPS MCP. |
| cli-mcp-stream | 050505ZMAY26 | SSE streaming upgrade for long calls — superseded; canonical MCP stays HTTPS-only from the CLI product perspective. |

## Done

| Slug | DTG | Note |
|------|-----|------|
| go-citadel-cli | 292032ZAPR26 | shipped (in-line with the B-track ratifications) |
| cli-watch | 051430ZMAY26 | CLI watch shipped end-to-end: Citadel exposes SSE list watches on repos, orgs, agents, OAuth clients, members, pending transfers, and agent tokens (polling snapshot diff + keepalive + Last-Event-ID); citadel-cli streams via `--watch`, ndjson/table modes, reconnect/backoff, and B6 scripted SSE tests. P2 C2 operator smoke stays human-owned. |
| cli-audit | 051145ZMAY26 | Shipped Citadel GET /api/audit/events and GET /api/audit/events/{id} with RBAC, time and kind filters, cli-pagination cursors, cascade linkage from purge, and agent.created audit rows. Delivered citadel-cli audit list/show with standard output modes, live opt-in test, and documentation. Deferred: P1 B6 expanded RBAC HTTP matrix for events; P2 operator smoke, tail-mode carry-forward, and spec hygiene. |
| cli-error-format | 051045ZMAY26 | Error envelope, exit-code map, errmap→CLIError migration, and main.go branching were already landed; Q-table ratified with Retry-After HTTP-date support aligned to apiclient. README/HUMANS document structured errors for json/yaml/ndjson. Live 429 integration and operator exit-code review stay in P2 per HUMAN_BLOCKERS. |
| cli-output-formats | 051045ZMAY26 | Delivered machine-readable list output: json/yaml/ndjson/csv/table with validation, frozen CSV columns per list verb, yaml keyed like json, and cmd-scoped stdout writers. Added httptest coverage for repo csv/yaml, ndjson across pages, and CSV helpers. README documents schemas; operator CSV paste smoke remains in P2. |
| cli-pagination | 051011ZMAY26 | Server list endpoints and citadel-cli list verbs now support opaque cursor pagination (?limit/?cursor, next_cursor), including members-specific cursors, ndjson streaming under --all, human tail hints, bounded completion fetch, and handler-level multi-page tests. P2 leaves the gated live 250-repo walk and operator production smoke as follow-ups. |
| cli-completion-dynamic | 050935ZMAY26 | Delivered dynamic shell completion with a 60s disk cache under XDG, ValidArgsFunction wiring for repos/namespaces/agents/OAuth client UUIDs/agent tokens, async PostRun invalidation on mutating verbs, static --output completion aligned to cli-output-formats, integration tests plus cache TTL tests, and README/HUMANS documentation including CITADEL_NO_COMPLETION_CACHE. Operator latency smoke (C2) remains for a human with a live namespace. |
| cli-cwd-context | 050915ZMAY26 | Implemented CWD git-origin repo resolution: -R/--repo and CITADEL_REPO, optional inference via git remote get-url origin for Citadel hosts (defaults plus CITADEL_GIT_HOSTS), --no-cwd-repo opt-out, TTY inference hint on stderr (respects --quiet and CI). Wired into repo get/delete and kg impact with tests and README/HUMANS guidance. Operator smoke task C2 left for humans. |
| cli-oauth-clients | 040041ZMAY26 | P1 B3: opt-in live integration test (oauth_clients_live_test.go) + §71 runbook. P2 C1 remains operator citadel-cli smoke — see specs/HUMAN_BLOCKERS.md §71. |
| cli-mcp-resources | 032359ZMAY26 | Shipped MCP resources/list, resources/read, prompts/list, prompts/get, citadel-cli `mcp resources` / `mcp prompts`, waitlist parity with tools/call, and automated conformance tests. SDD closeout complete (P2 C2). Remaining operator/NOMAD rows: P0 Q-table sign-off and P2 Claude Desktop smoke — see [specs/HUMAN_BLOCKERS.md §69](../../HUMAN_BLOCKERS.md#69--cli-mcp-resources-nomad-procedural-q-table--claude-desktop-smoke). |
| go-citadel-cli-repo | 032036ZMAY26 | Shipped repo/namespace/agent CRUD CLI verbs against live APIs. repo create/list/get/delete; namespace list/get/members/transfer (with initiate/list-pending/accept/decline/revoke subcommands); agent list/get/delete/rotate-token. All verbs carry --help + --output json (A1). cmd_test.go integration suite covers command-tree structure, flag presence, and destructive-verb --yes gates (A2). Destructive verbs gate on typed-slug confirm (A3). Q-table ratified (A4). repo rename descoped (no server endpoint). namespace transfer org-only for now; personal namespace transfer deferred to server-side follow-on. |
| cli-mcp-tools | 010300ZMAY26 | shipped; HUMAN follow-up = positive prod smoke with real JWT |
