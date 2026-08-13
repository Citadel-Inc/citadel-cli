# TODO — follow-on and carry-forward

Planning backlog only. Parked/rejected specs (`specs/parked/`) and live-smoke-only residuals stay out unless reopened by product decision.

Do **not** restore `account passkey` / `account device` — removed deliberately in `b1871a6` (settings-panel, not a developer workflow). Phase B MFA recovery under that tree is obsolete with the removal.

---

## Shipped 130333ZAUG26 (fenced wave 32)

| # | Item | Notes |
| --- | --- | --- |
| 133 | Exact `assertRepoTopicBadOutput` | Get-format full-string; list/set/popular hermetics |
| 134 | Mutation `--output` flag help | `addMutationOutputFlag`; json-only completer; split mixed registrations |
| 143 | Notification residual Contains | Exact no-auth + not-found |
| 144 | Deduplicate project edge guard tests | Dropped handler withServer+Contains; hermetic to-namespace-id |
| 155 | Deduplicate ns deploy-token list guards | Shared `validateDeployTokenListFlags` before path |
| 156 | Exact `assertNamespaceBadOutput` | List-format full-string |
| — | Should-fix closeout | Tag delete output-before-path + BadOutput_NoRepo |

---

## Shipped 130248ZAUG26 (fenced wave 31)

| # | Item | Notes |
| --- | --- | --- |
| 131 | Issue create/comment/label/close-refs output-before-path | Output validators before `resolveIssueNamespacePath`; BadOutput_NoRepo + MissingRepo |
| 133 | Issue create/close-refs exact BadOutput | Full-string + four-clear; topics helper still open |
| 135 | Project empty-path hermetics | status rollup/drilldown, edge add/delete/restore |
| 145 | Label mutate MissingRepo hermetics | create/edit/delete `--no-cwd-repo` |
| 151 | Namespace deploy-token BadOutput hermetics | Wrapper guards + exact list/create/revoke |
| 153 | Namespace delete dry-run + bad `--output` | Exact mutation error; no Would DELETE |
| 154 | `setNamespaceHermeticEnv` `CITADEL_REPO` | Four-clear parity |
| — | Should-fix closeout | `issue view` output-before-path; exact view/comment-edit BadOutput; four-clear assign/close/reopen |

---

## Shipped 130225ZAUG26 (fenced wave 30)

| # | Item | Notes |
| --- | --- | --- |
| 139 | Namespace delete output-before-client | `validateMutationOutput` before dry-run/client; exact BadOutput hermetic |
| 140 | Repo delete output-before-path | Mutation output before `resolveRepoFromPosOrFlag`; BadOutput + NoRepo exact |
| 141 | KG paginated cursor + MissingRepo | `validateDescCursor` before path/client on search/symbols/files/fulltext |
| 150 | Tag list BadOutput_NoRepo hermetic | Exact list format; `--no-cwd-repo` |
| 152 | Milestone `--state` before path | State guard before `resolveIssueNamespacePath`; BadState hermetics |
| — | Should-fix closeout | `CITADEL_REPO` clear; `--no-cwd-repo` on NoRepo/cursor/state hermetics |

---

## Shipped 130214ZAUG26 (fenced wave 29)

| # | Item | Notes |
| --- | --- | --- |
| 146 | PR collab + view output-before-path | check/comment/reviewer/review/view; BadOutput_NoRepo exact |
| 147 | Milestone output-before-path | all five handlers; BadOutput_NoRepo exact |
| 148 | Deploy-token repo wrappers output-before-path | list/create/revoke; pagination/watch/cursor before path |
| 149 | Branch delete/set-default output-before-path | mutation output before `parseRepoScopedNameArgs` |
| — | Should-fix closeout | Exact `-R` BadOutput + env clears; branch list NoRepo hermetic |

---

## Shipped 130046ZAUG26 (fenced wave 28)

| # | Item | Notes |
| --- | --- | --- |
| 132 | Label mutate output-before-path | create/edit/delete `validateMutationOutput` before path; BadOutput_NoRepo exact |
| 136 | Clone whitespace-only hermetic | `" "` → exact `argument must be <namespace>/<repo>` |
| 137 | MCP prompts-get bad `--arg` hermetic | `requireMcpError` same string as `call` |
| 138 | Agent delete/rotate output-before-client | `validateMutationOutput` before client; `agent_handler_test.go` |
| 142 | Webhook repo wrappers output-before-path | list/create/get/edit/delete/deliveries; BadOutput_NoRepo + list MissingRepo |
| — | Should-fix closeout | Remaining webhook verbs hoist; exact legacy BadOutput + env clears |

---

## Open

| # | Item | Notes |
| --- | --- | --- |
| 157 | Mutation `--output` completion golden | Mirror `TestCompleteOutputFormats_GoldenList` for `completeMutationOutputFormats` (json-only) |
| 158 | Get-verb `--output` help vs `validateGetOutput` | `addOutputFlag` still advertises ndjson/csv on get-style verbs (e.g. release create/edit) |
| 159 | Broader mutation Usage spot-checks | `TestMutationOutputFlagUsage` only repo delete vs list |
| 160 | Agent list/create exact BadOutput | `handler_test.go` still `strings.Contains` |
| 161 | Drop duplicate deploy-token mutation validate | Wrapper + inner `validateMutationOutput` on create/revoke |

---

## Blocked on server SSE

### 3. Audit event tail / `--watch`

**Feature.** `cli-audit` deferred `--follow` until SSE; `cli-watch` now owns SSE for many list verbs, but `audit list` has no watch flag.

| | |
| --- | --- |
| **Packages / files** | `cmd/audit.go`, `cmd/watch.go`, `internal/sseclient`, `docs/cli.md` / `HUMANS.md` (audit) |
| **Carry-from** | `specs/done/cli-audit/tasks.md` C4; `specs/done/cli-audit/spec.md` A6 |
| **Blocked** | Daemon `auditapi` has **no** `listwatch` / `Accept: text/event-stream` path (verified against Citadel-Inc/citadel). Do not wire CLI until server ships SSE. |
| **Acceptance** | `audit list --watch` (and ndjson mode) streams events until interrupt; missing server SSE fails with a clear error, not a hang |

### 4. Extend `--watch` to high-churn workflow lists

**Feature.** Watch is on repo/agent/token/oauth/namespace lists; not on `issue list`, `pr list`, or `notification list`.

| | |
| --- | --- |
| **Packages / files** | `cmd/issue.go`, `cmd/pr.go`, `cmd/notification.go`, `cmd/watch.go` / `watch_table.go` |
| **Blocked** | Daemon issues/PR/notification APIs have **no** listwatch SSE (same as #3). |
| **Acceptance** | Documented `--watch` on issue/PR/notification lists when server supports them; behaviour matches `repo list --watch` |

---

## Deferred

### 14b. Homebrew formula (optional)

**Feature.** `make install` shipped; Homebrew tap still deferred until a second non-operator adopter.

| | |
| --- | --- |
| **Packages / files** | optional `rethunk-tech/tap` formula (out of tree), `docs/cli.md` Distribution |
| **Traps** | Do not force tap early; keep docs demand-gated |
| **Acceptance** | Formula installs current release binary; docs link the tap when published |

### 16. Replace Phase 0 placeholder LICENSE

**Feature.** Root `LICENSE` still says terms may change; releases publish under that text.

| | |
| --- | --- |
| **Packages / files** | `LICENSE`, `NOTICE`, README license badge/blurb if any |
| **Traps** | Legal decision — not an agent fiat; keep NOTICE attributions intact |
| **Acceptance** | Chosen license text committed; README/HUMANS link matches; no "Phase 0 placeholder" language remains |

### 21. Optional migrate MCP client to official go-sdk

**Feature.** Retries shipped in hand-rolled client; SDK `StreamableClientTransport` + `ReconnectOptions` still a larger swap.

| | |
| --- | --- |
| **Packages / files** | `internal/mcpclient`, `cmd/mcp.go`, `cmd/doctor.go` |
| **Refs** | Context7 `/modelcontextprotocol/go-sdk` |
| **Traps** | Keep protocol pin; never auto-retry `tools/call`; doctor must still initialize |
| **Acceptance** | Behaviour parity with current client + reconnection options documented |

---

## Explicitly not tracked here

| Item | Why |
| ------ | ----- |
| `cli-webhook-test`, billing, avatar, privacy, account-export, mcp-stdio/stream | `specs/parked/` — rejected or superseded |
| Restoring `account *` security verbs | Deliberate removal `b1871a6` |
| Live-smoke / C1-only residuals on done specs | Env-gated operator work, not product features |
| shadcn / web component work | This repo is a Go Cobra CLI; browser UX lives in Citadel web |
