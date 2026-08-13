# TODO — follow-on and carry-forward

Planning backlog only. Parked/rejected specs (`specs/parked/`) and live-smoke-only residuals stay out unless reopened by product decision.

Do **not** restore `account passkey` / `account device` — removed deliberately in `b1871a6` (settings-panel, not a developer workflow). Phase B MFA recovery under that tree is obsolete with the removal.

---

## Open

| # | Item | Notes |
| --- | --- | --- |
| 131 | Issue create/comment/label/close-refs output-before-path | `validateMutationOutput` / `validateGetOutput` before `resolveIssueNamespacePath`; BadOutput_NoRepo + MissingRepo hermetics |
| 132 | Label mutate output-before-path | create/edit/delete + NoRepo hermetics |
| 133 | Exact older BadOutput hermetics | `TestIssueCreate_BadOutput_Hermetic`, `TestIssueCloseRefs_BadOutput_Hermetic`, `assertRepoTopicBadOutput` |
| 134 | Mutation `--output` flag help | `addOutputFlag` lists json/yaml/ndjson/csv/table on verbs that only allow json/default |
| 135 | Project empty-path hermetics for remaining verbs | status rollup/drilldown, edge add/delete/restore |
| 136 | Clone whitespace-only hermetic | `" "` → exact `argument must be <namespace>/<repo>` (trim already in `runRepoClone`) |
| 137 | MCP prompts-get bad `--arg` hermetic | `parseArgPairs` shared; only `call` covered today |
| 138 | Agent delete/rotate output-before-client | `validateMutationOutput` before `newAPIClient`; BadOutput hermetics |
| 139 | Namespace delete output-before-client | Mirror rename `validateMutationOutput`; BadOutput hermetic |
| 140 | Repo delete output-before-client | Mirror create `validateMutationOutput`; BadOutput hermetic |
| 141 | KG paginated cursor + MissingRepo | `validateDescCursor` before client on symbols/files/search/fulltext; hermetics |
| 142 | Webhook repo wrappers output-before-path | `validateListOutput`/`validateGetOutput` before `resolveRepoFromPosOrFlag`; BadOutput_NoRepo |
| 143 | Notification residual Contains | `TestNotificationList_NoAuth` and remaining Contains asserts |
| 144 | Deduplicate project edge guard tests | `handler_test.go` withServer+Contains vs `project_handler_test.go` hermetics |

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
