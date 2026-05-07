# Specs

## Active

*In-flight specifications (DRAFT through BLOCKED) under `specs/active/`. Claim, block, unblock, park, and close via [`@rethunk/citadel-sdd`](https://github.com/Rethunk-AI/citadel-sdd).*

| Slug | State | DTG | Owner |
|------|-------|-----|-------|
| cli-git-wrappers | DRAFT |  | unassigned |
| cli-webhooks | DRAFT |  | unassigned |

## Done

*Completed work (**DONE**) after `spec_close`; directories live under `specs/done/`. Lifecycle semantics and tools: [`@rethunk/citadel-sdd`](https://github.com/Rethunk-AI/citadel-sdd).*

| Slug | DTG | Note |
|------|-----|------|
| cli-error-format | 071629ZMAY26 | Structured error output remains shipped: README/HUMANS document the json/yaml/ndjson envelope, and the remaining P2 live 429 integration plus operator exit-code review stay as out-of-band follow-up work rather than a HUMAN_BLOCKERS dependency. |
| cli-oauth-clients | 071629ZMAY26 | OAuth client CLI support remains shipped: the opt-in live integration test and runbook landed, while any additional operator citadel-cli smoke stays as an out-of-band follow-up instead of a HUMAN_BLOCKERS entry. |
| cli-watch | 071628ZMAY26 | Live repo watch smoke now passed on rethunk-ai after fixing SSE timeout inheritance in the CLI: repo list --watch stayed connected long enough to observe add/remove events from a temporary repository create/delete cycle. |
| cli-branch-tag | 070526ZMAY26 |  |
| cli-oauth-login | 070526ZMAY26 |  |
| cli-agent-create | 070504ZMAY26 |  |
| cli-issues | 070107ZMAY26 | Operator smoke completed via citadel-cli against live namespace rethunk: created issue #1, added a comment, closed it, and verified close-refs. REST routing fix landed in citadel-cli so the live API resolves through api.src.land while OAuth/MCP remain on mcp.src.land. |
| cli-projectgraph | 060539ZMAY26 | Delivered top-level `citadel-cli project` with URL-encoded multi-segment namespace paths, read verbs (pin-chain, walk, neighbors, status rollup/drilldown), write verbs (edge add/delete/restore, reindex) with typed confirm/--yes, httptest matrix incl. multi-segment pin-chain + read/write 404 paths, docs/cli.md, optional live test behind CITADEL_TEST_PROJECTGRAPH_LIVE + CITADEL_TEST_PROJECTGRAPH_SLUG. Q4 recovery-scan intentionally unimplemented; P1 B2 remains open. |
| cli-global-search | 060535ZMAY26 | Shipped top-level `citadel-cli search` with authenticated GET /api/search, default scope=namespaces, --public for scope=all, httptest coverage for query_too_short/invalid_scope/invalid_limit, optional CITADEL_TEST_SEARCH_LIVE=1, and docs/cli.md QoS framing. |
| cli-kg-extended | 060504ZMAY26 | Shipped extended KG HTTP verbs (search, symbols, files, walk, fulltext, diff); migrated kg impact + symbol resolution to /api/namespaces/{slug}/kg/*; added httptest (401/404/429), docs/cli.md section, plan appendix, and opt-in live test. P1 pagination/table polish remains open. |
| cli-account-security | 060459ZMAY26 | Phase A delivered: account passkey list/rename/delete, device list/revoke, PATCH client support, httptest + opt-in live tests (CITADEL_TEST_ACCOUNT_SECURITY_LIVE), docs and CSV contracts. Phase B MFA recovery verbs intentionally deferred (P1 B1 remains open). |
| cli-ssh-keys | 060441ZMAY26 | SSH key surface complete: list/add/delete against /account/ssh-keys, private-key rejection, output modes, httptest coverage, docs/cli.md, live opt-in list test, and shell tab completion for delete UUIDs via cached GET /account/ssh-keys (KeySSHKeys) with PostRun invalidation after add/delete. |
| cli-audit-sessions | 052320ZMAY26 | Added `citadel-cli audit sessions list` and `audit sessions show` backed by `/audit/sessions` with required namespace (`--ns` or `--namespace`/`-n`), `since`/limit/offset pagination (no cursor), output formats on list, JSON/YAML/table passthrough on show, CSV projection types, httptest coverage for client and server `ns_required`/`invalid_since`/404 paths and minimal drill-down JSON without operator-console fields, user docs + plan appendix, and opt-in live list behind `CITADEL_TEST_AUDIT_SESSIONS_LIVE` + `CITADEL_TEST_AUDIT_SESSIONS_NS`. P1 B1 (cross-link cli-audit retrospective) left open until sibling directory lands under specs/done. |
| cli-org-invitations | 052317ZMAY26 | Delivered `citadel-cli org invitation` (pending, list, create, revoke, accept) with output formats, TTY email prompt, token-file accept, httptest matrix for 409/404/400 paths, docs/cli.md, plan RECON appendix, and opt-in live pending test behind CITADEL_TEST_ORG_INVITATIONS_LIVE=1. |
| cli-audit | 051145ZMAY26 | Shipped Citadel GET /api/audit/events and GET /api/audit/events/{id} with RBAC, time and kind filters, cli-pagination cursors, cascade linkage from purge, and agent.created audit rows. Delivered citadel-cli audit list/show with standard output modes, live opt-in test, and documentation. Deferred: P1 B6 expanded RBAC HTTP matrix for events; P2 operator smoke, tail-mode carry-forward, and spec hygiene. |
| cli-output-formats | 051045ZMAY26 | Delivered machine-readable list output: json/yaml/ndjson/csv/table with validation, frozen CSV columns per list verb, yaml keyed like json, and cmd-scoped stdout writers. Added httptest coverage for repo csv/yaml, ndjson across pages, and CSV helpers. README documents schemas; operator CSV paste smoke remains in P2. |
| cli-pagination | 051011ZMAY26 | Server list endpoints and citadel-cli list verbs now support opaque cursor pagination (?limit/?cursor, next_cursor), including members-specific cursors, ndjson streaming under --all, human tail hints, bounded completion fetch, and handler-level multi-page tests. P2 leaves the gated live 250-repo walk and operator production smoke as follow-ups. |
| cli-completion-dynamic | 050935ZMAY26 | Delivered dynamic shell completion with a 60s disk cache under XDG, ValidArgsFunction wiring for repos/namespaces/agents/OAuth client UUIDs/agent tokens, async PostRun invalidation on mutating verbs, static --output completion aligned to cli-output-formats, integration tests plus cache TTL tests, and README/HUMANS documentation including CITADEL_NO_COMPLETION_CACHE. Operator latency smoke (C2) remains for a human with a live namespace. |
| cli-cwd-context | 050915ZMAY26 | Implemented CWD git-origin repo resolution: -R/--repo and CITADEL_REPO, optional inference via git remote get-url origin for Citadel hosts (defaults plus CITADEL_GIT_HOSTS), --no-cwd-repo opt-out, TTY inference hint on stderr (respects --quiet and CI). Wired into repo get/delete and kg impact with tests and README/HUMANS guidance. Operator smoke task C2 left for humans. |
| cli-mcp-resources | 032359ZMAY26 | Shipped MCP resources/list, resources/read, prompts/list, prompts/get, citadel-cli `mcp resources` / `mcp prompts`, waitlist parity with tools/call, and automated conformance tests. SDD closeout complete (P2 C2). Remaining follow-up: P2 HTTPS-MCP client smoke against a live server (automation-capable; not a HUMAN_BLOCKERS item). |
| go-citadel-cli-repo | 032036ZMAY26 | Shipped repo/namespace/agent CRUD CLI verbs against live APIs. repo create/list/get/delete; namespace list/get/members/transfer (with initiate/list-pending/accept/decline/revoke subcommands); agent list/get/delete/rotate-token. All verbs carry --help + --output json (A1). cmd_test.go integration suite covers command-tree structure, flag presence, and destructive-verb --yes gates (A2). Destructive verbs gate on typed-slug confirm (A3). Q-table ratified (A4). repo rename descoped (no server endpoint). namespace transfer org-only for now; personal namespace transfer deferred to server-side follow-on. |
| cli-mcp-tools | 010300ZMAY26 | shipped; HUMAN follow-up = positive prod smoke with a real authenticated session |
| go-citadel-cli | 292032ZAPR26 | shipped (in-line with the B-track ratifications) |
| cli-deploy-tokens | 043044ZMAY26 |  |

## Parked

*Deliberately not pursued (**PARKED**); superseded or withdrawn specs under `specs/parked/`. Use `spec_park` from [`@rethunk/citadel-sdd`](https://github.com/Rethunk-AI/citadel-sdd).*

| Slug | DTG | Note |
|------|-----|------|
| cli-mcp-stdio | 050505ZMAY26 | superseded by HTTPS MCP canonical policy ([`../README.md`](../README.md)). |
| cli-mcp-stream | 050505ZMAY26 | superseded by HTTPS MCP canonical policy ([`../README.md`](../README.md)). |
