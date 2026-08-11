# TODO — follow-on and carry-forward

Planning backlog only. Parked/rejected specs (`specs/parked/`) and live-smoke-only residuals stay out unless reopened by product decision.

Do **not** restore `account passkey` / `account device` — removed deliberately in `b1871a6` (settings-panel, not a developer workflow). Phase B MFA recovery under that tree is obsolete with the removal.

---

## Shipped 110430ZAUG26 (fenced wave 23)

| # | Item | Notes |
| --- | --- | --- |
| 102 | Namespace OutOrStdout + transfer decline guard | Human empty/hint/success via writers; decline `validateMutationOutput` before client |
| 103 | Token list OutOrStdout + issue secret capture | Empty/pagination writers; EmptyHuman/PaginationHint + issue cleartext assert |
| 104 | KG impact OutOrStdout + empty-symbol guard | `printImpactTree` writers; empty symbol before auth; hermetic + human capture |
| 105 | Org invitation revoke path-before-auth | Trim slug/id before client; empty-id/slug hermetics + happy capture |
| 106 | Audit empty + repo create/delete OutOrStdout | Empty list + mutation success via writers; stdout capture |
| 107 | Search + repo-topics bad-output hermetics | Empty-XDG hermetics; topics exact empty assert |
| — | Should-fix closeout | Hermetic API-env clears (ns/org/audit/repo); exact org/repo asserts; kg writer discard + broader human assert |

---

## Shipped 110347ZAUG26 (fenced wave 22)

| # | Item | Notes |
| --- | --- | --- |
| 98 | Release asset list path-before-client | Namespace + tag before client; path-guards hermetic |
| 99 | Release asset ID completion path-before-client | `resolveIssueNamespacePath` before client |
| 100 | Agent / OAuth list OutOrStdout | Empty + cursor hint via writers; EmptyHuman + PaginationHint asserts |
| 101 | OAuth rotate-secret clipboard stdout capture | `rootForOut` + exact `sek\n` |
| — | Should-fix closeout | OAuth pagination hint test; agent empty-list capture; agent delete OutOrStdout + happy assert |

---

## Shipped 110340ZAUG26 (fenced wave 21)

| # | Item | Notes |
| --- | --- | --- |
| 93 | Gist create/edit dry-run hermetics | Full-string empty-XDG; delete hermetics cleared + exact asserts |
| 95 | Agent rotate-token OutOrStdout | `rootForOut` capture in Success/Happy |
| 96 | OAuth create/rotate secret OutOrStdout | Command writers; HumanSecretOutput + handler happy capture |
| 97 | Release list/view/latest/download path-before-client | Path guards hermetic; exact namespace-path assert |
| — | Should-fix closeout | Revoke ErrOrStderr; revoke/gist delete API-env clear; handler happy capture; path-guard exact assert |

---

## Shipped 110324ZAUG26 (fenced wave 20)

| # | Item | Notes |
| --- | --- | --- |
| 89 | Auth provider link normalize before client | Empty-XDG hermetic |
| 90 | Release create/edit/asset-upload dry-run before auth | Empty-XDG full-string hermetics |
| 91 | Org invitation create/accept auth-before-guard | Mutation output; invitee/token before client; hermetics |
| 92 | Deploy-token create `--expires` before auth | Empty-XDG invalid-expires hermetic |
| 94 | Token issue expires + OutOrStdout | Shared `parseExpiresFlag`; empty-XDG hermetic |
| — | Should-fix closeout | Clear inherited API env on hermetics; mutation output on inv create/accept; empty-list OutOrStdout |

---

## Shipped 110316ZAUG26 (fenced wave 19)

| # | Item | Notes |
| --- | --- | --- |
| 82 | Org member remove/set-permissions auth-before-guard | Confirm before client; set-permissions mutation output + json; hermetics |
| 83 | PR collab output guards | List/mutate `--output` before client; diff/check left alone; bad-output hermetics |
| 84 | Auth provider unlink normalize+confirm before client | Empty-XDG hermetics |
| 85 | OAuth revoke OutOrStdout | Dry-run via command writer; hermetic uses `rootForOut` |
| 86 | Retire weak DryRun handler stubs | Removed mock-server no-message duplicates |
| 87 | Agent rotate-token empty-XDG hermetic | `--yes` expects auth/config error |
| 88 | Rename token revoke dry-run hermetic | `TestTokenRevoke_DryRun_Hermetic` |
| — | Should-fix closeout | Collab lists use `validateGetOutput`; comment-add output before body; tighten rotate/unlink hermetic asserts |

---

## Shipped 110300ZAUG26 (fenced wave 18)

| # | Item | Notes |
| --- | --- | --- |
| 75 | Namespace transfer revoke (+ delete) auth-before-guard | Dry-run/confirm before client; OutOrStdout; hermetics |
| 76 | Dry-run hermetics (oauth revoke, gist delete, ns rename) | Empty-XDG; full oauth dry-run message assert |
| 77 | Agent delete/rotate auth-before-guard | Dry-run skips lookup; confirm before client; hermetic preview |
| 78 | Token revoke dry-run before auth | + empty-XDG dry-run test |
| 79 | SSH key delete dry-run before auth | + hermetic |
| 80 | Repo delete dry-run/confirm before auth | OutOrStdout; hermetic message assert |
| 81 | Deploy-token revoke dry-run before auth | + hermetics |
| — | Should-fix closeout | OutOrStdout on repo/agent/ns dry-run; oauth full-string assert; ns stdout `t.Cleanup` |

---

## Shipped 110232ZAUG26 (fenced wave 17)

| # | Item | Notes |
| --- | --- | --- |
| 68 | SSH key add auth-before-guard | Key/output via `validateMutationOutput` before client; hermetic |
| 69 | Notification unread/prefs output-before-auth | `validateGetOutput` before client; hermetics |
| 70 | Repo create auth-before-guard | Namespace/slug/visibility + mutation output before client; hermetics |
| 71 | KG extended list output/pagination-before-auth | All six verbs before client; broadened hermetics |
| 72 | Namespace transfer/rename output-before-auth | Initiate/accept/rename mutation output before client; hermetics |
| 73 | OAuth rotate/revoke + gist delete | Output before client; gist keeps yaml allowlist |
| 74 | PR list bad-output hermetic | Empty-XDG `TestPRList_BadOutput_Hermetic` |
| — | Should-fix closeout | Shared SSH helper; full mutation error wording; repo local-flag hermetics; gist yaml restore |

---

## Shipped 110218ZAUG26 (fenced wave 16)

| # | Item | Notes |
| --- | --- | --- |
| 65 | Issue reopen bad-output hermetic | Empty-XDG `TestIssueReopen_BadOutput_Hermetic` |
| 66 | Agent/token list hermetic depth | `_Hermetic` rename; `--all`+json; watch yaml/csv hermetics |
| 67 | Issue mutate hermetic assert strings | Full `--output for <verb> supports json…` wording |
| — | Release delete/asset-delete auth-before-guard | Mutation-output + empty args before client; hermetics |
| — | Label create/edit/delete auth-before-guard | Mutation-output before client; hermetics |
| — | Should-fix closeout | Label delete confirm before `newAPIClient` |

---

## Shipped 110204ZAUG26 (fenced wave 15)

| # | Item | Notes |
| --- | --- | --- |
| 61b | Agent/token list(+create) auth-before-guard | List/pagination/watch/cursor + create mutation before client; hermetics |
| 62 | Issue mutate auth-before-guard | create/assign/state/label mutation-output before client; hermetics |
| 63 | Milestone create/edit/delete auth-before-guard | Mutation-output (+ title/UUID) before client; hermetics |
| 64 | OAuth create + repo tag delete auth-before-guard | Create mutation+flags before client; tag delete already ordered + hermetic |
| — | Should-fix closeout | Empty issue title + whitespace OAuth `--name` before auth + hermetics |

---

## Shipped 110131ZAUG26 (fenced wave 14)

| # | Item | Notes |
| --- | --- | --- |
| 57 | Label list auth-before-guard | `validateListOutput` before client + hermetic |
| 58 | Notification list auth-before-guard | Output + pagination + `--all`/json before client; normalize + hermetics |
| 59 | Org member list auth-before-guard | Output + pagination + member cursor before client; hermetics |
| 60 | Repo branch/tag list(+create) | List/pagination + tag create mutation before client; hermetics |
| 61a | Repo list + inv list/pending + SSH list + recovery-scan | Full list/watch/cursor before client; dedicated hermetic files |
| — | Should-fix closeout | Notification output normalize; all/json + bad-cursor hermetics |

---

## Shipped 110123ZAUG26 (fenced wave 13)

| # | Item | Notes |
| --- | --- | --- |
| 54 | Release list/asset-list auth-before-guard | `validateListOutput` before client + hermetic |
| 55 | Repo commit list auth-before-guard | `validateListOutput` before client + hermetic |
| 56a | Deploy-token list/create/revoke | Output (+ watch/pagination) before client; hermetic list/create/revoke/watch |
| 56b | OAuth clients list + UUID guards | List/watch + show/rotate/revoke UUID before client; hermetic |
| 56c | Namespace list/members/transfers | List + watch/pagination before client; hermetic |
| — | Watch-output should-fix closeout | `validateWatchOutput` (+ pagination/cursor) before auth on 56a–c |

---

## Shipped 102354ZAUG26 (fenced wave 12)

| # | Item | Notes |
| --- | --- | --- |
| 49 | Audit list/show auth-before-guard | Flags/output before client; hermetic RBAC tests |
| 50 | Release auth-before-guard | get/create/edit/upload + empty-edit before auth |
| 51 | Issue/milestone/comment auth-before-guard | View/close-refs/milestone view+list/comment list |
| 52 | PR view/check auth-before-guard | Output before client + hermetic |
| 53 | Repo commit get auth-before-guard | Output before client + hermetic |
| — | Webhook delete output-before-auth | Scout leftover from #48 |

---

## Shipped 102348ZAUG26 (fenced wave 11)

| # | Item | Notes |
| --- | --- | --- |
| 47 | Keep-split limit validation docs | `docs/cli.md` List pagination; HUMANS pagination notes |
| 48 | Webhook create/edit/get/redeliver auth-before-guard | Flag/output/UUID before `newAPIClient` + hermetic tests |
| — | Audit sessions show output-before-auth | `validateGetOutput` + hermetic bad-output |
| — | Search negative `--limit` hermetic | Local guard coverage |

---

## Shipped 102337ZAUG26 (fenced wave 10)

| # | Item | Notes |
| --- | --- | --- |
| 44 | Hermetic gist + webhook deliveries negative-offset | `executeGistTestCommand` XDG; deliveries test XDG |
| 45 | Gist list negative `--limit` | Guard + test; validation before auth |
| 46 | Webhook list `strconv.Itoa` | List + deliveries query builders |
| — | Webhook list/deliveries auth-before-guard | Flags before `newAPIClient` |
| — | Audit sessions auth-before-guard + hermetic negatives | Handler + both tests |
| — | Deliveries negative `--limit` test | `readPagination` range wording |

---

## Shipped 102245ZAUG26 (fenced wave 9)

| # | Item | Notes |
| --- | --- | --- |
| 41 | Delivery completion goldens pin `DefaultLimit` | Three mocks use `strconv.Itoa(pagination.DefaultLimit)` |
| 42 | Delivery completion `strconv.Itoa` | `fetchWebhookDeliveryIDs` limit query |
| 43 | Gist list negative `--offset` | Shared `--offset cannot be negative` + test |

---

## Shipped 102238ZAUG26 (fenced wave 8)

| # | Item | Notes |
| --- | --- | --- |
| 37 | Delivery completion `DefaultLimit` | `fetchWebhookDeliveryIDs` uses `pagination.DefaultLimit` |
| 38 | Positional-repo delivery golden | `TestCompleteRepoWebhookDeliveryIDs_UsesPositionalRepo` get+redeliver |
| 39 | Namespace deliveries `--offset` docs | Example beside namespace list block in `docs/cli.md` |
| 40 | Align negative-offset copy | Audit split to `--limit`/`--offset cannot be negative`; webhook kept; tests |

---

## Shipped 091705ZAUG26 (fenced wave 7)

| # | Item | Notes |
| --- | --- | --- |
| 34 | Delivery ID shell completion | `get`/`redeliver` ValidArgsFunction; bounded `limit=50` fetch; golden tests |
| 35 | Deliveries list `--offset` | First-page query only; flag contracts; docs; negative + `--all` page-2 coverage |
| 36 | Audit show 403 httptest | `TestAuditRBAC_ShowForbidden` mirrors list surface |

---

## Shipped 091652ZAUG26 (fenced wave 6)

| # | Item | Notes |
| --- | --- | --- |
| 30 | Cobra group test harness | `addTestRootGroups` on completion/man/watch SSE roots |
| 31 | Webhook edit cmd_test | Tree + flag contracts for repo/namespace `edit` |
| 32 | Audit list/show RBAC tests | Happy, namespace filter, 404, 403 (`cli-audit` B6) |
| 33 | Webhook deliveries | `list`/`get`/`redeliver`; envelope unwrap; delivery-specific errors |

---

## Shipped 091637ZAUG26 (fenced wave 5)

| # | Item | Notes |
| --- | --- | --- |
| 26 | Shared binary/TTY download helpers | `cmd/download_tty.go`; release/repo raw/gist call sites |
| 27 | YAML TTY allowlist | `application/yaml`, `application/x-yaml` (+ `text/*`) |
| 28 | Deploy-token list `--watch` tests | Repo + namespace ndjson smoke; asserts payload + `limit=` query |
| 29 | Webhook edit (PATCH) | `repo`/`namespace webhook edit`; rotate-secret; dry-run |

---

## Shipped 091627ZAUG26 (fenced wave 4)

| # | Item | Notes |
| --- | --- | --- |
| 1 | OAuth refresh-token exchange | Persist refresh; one-shot 401 refresh when no agent; flock `clicfg.Update`; skip when `CITADEL_ACCESS_TOKEN` set |
| 7 | Repo raw blob download | `repo browse raw`; stream + TTY binary guard; output files `0600` |
| 17 | Doctor host + git remote | REST vs MCP bases; WARN on non-Citadel / missing origin |
| 23 | Gist raw stream + TTY | `io.Copy` + binary TTY refusal |
| 24 | KG `--all` handler tests | Multi-page httptest for search/symbols/files/fulltext |
| 25 | Soft PKCE/refresh errors | `error`/`error_description` + truncated body; debug-http form redaction |

---

## Shipped 091358ZAUG26 (fenced wave 3)

| # | Item | Notes |
| --- | --- | --- |
| 2 | KG `--all` + table | Cursor walk + stable table columns; `symbol_name` in search SYMBOL |
| 6 | Device-code OAuth | `auth login --device`; TTL + transient poll retries |
| 8 | Release asset CRUD | list/upload/download/delete; download follows `download_url` |
| 10 | Gist command group | CRUD + raw via `GetStream`; `GroupID: repo` |
| 18 | Friendlier `api --input` | Invalid/empty JSON names `--input` pre-HTTP |
| 19 | recovery-scan mismatch test | Wrong typed confirm → no POST |
| 20 | `make install` empty man guard | Clear error before `*.1` install |
| 22 | Document `oauth clients list --dcr` | Client-side filter; incompatible with `--watch` |

---

## Shipped 091210ZAUG26 (fenced wave)

| # | Item | Notes |
| --- | --- | --- |
| 5 | Scrub stale account-security docs | Web settings panel; CHANGELOG corrected |
| 9 | Project admin recovery-scan | `project admin recovery-scan` + typed confirm / `--yes` |
| 11 | MCP list/read retries | Hand-rolled allowlist; `tools/call` + `initialize` single-shot |
| 12 | Cobra help groups | Auth / Repo & Git / Collaboration / Ops / Meta |
| 13 | `api --input` | Raw JSON body; mutually exclusive with `-f` |
| 14a | `make install` | `PREFIX`/`DESTDIR`; binary + man1 — Homebrew still open (#14b) |
| 15 | OAuth `--dcr` filter | Client-side (daemon list has no `dcr` query); rejects `--watch --dcr` |

---

## Round 1 — blocked on server SSE (091204ZAUG26)

### 3. Audit event tail / `--watch`

**Feature.** `cli-audit` deferred `--follow` until SSE; `cli-watch` now owns SSE for many list verbs, but `audit list` has no watch flag.

| | |
| --- | --- |
| **Packages / files** | `cmd/audit.go`, `cmd/watch.go`, `internal/sseclient`, `docs/cli.md` / `HUMANS.md` (audit) |
| **Carry-from** | `specs/done/cli-audit/tasks.md` C4; `specs/done/cli-audit/spec.md` A6 |
| **Blocked** | Daemon `auditapi` has **no** `listwatch` / `Accept: text/event-stream` path (verified 091358ZAUG26 against Citadel-Inc/citadel). Do not wire CLI until server ships SSE. |
| **Acceptance** | `audit list --watch` (and ndjson mode) streams events until interrupt; missing server SSE fails with a clear error, not a hang |

### 4. Extend `--watch` to high-churn workflow lists

**Feature.** Watch is on repo/agent/token/oauth/namespace lists; not on `issue list`, `pr list`, or `notification list`.

| | |
| --- | --- |
| **Packages / files** | `cmd/issue.go`, `cmd/pr.go`, `cmd/notification.go`, `cmd/watch.go` / `watch_table.go` |
| **Blocked** | Daemon issues/PR/notification APIs have **no** listwatch SSE (same audit as #3). |
| **Acceptance** | Documented `--watch` on issue/PR/notification lists when server supports them; behaviour matches `repo list --watch` |

---

## Explicitly not tracked here

| Item | Why |
| ------ | ----- |
| `cli-webhook-test`, billing, avatar, privacy, account-export, mcp-stdio/stream | `specs/parked/` — rejected or superseded |
| Restoring `account *` security verbs | Deliberate removal `b1871a6` |
| Live-smoke / C1-only residuals on done specs | Env-gated operator work, not product features |
| shadcn / web component work | This repo is a Go Cobra CLI; browser UX lives in Citadel web |

---

## Round 2 — server-ahead CLI gaps (091204ZAUG26)

### 14b. Homebrew formula (optional)

**Feature.** `make install` shipped; Homebrew tap still deferred until a second non-operator adopter.

| | |
| --- | --- |
| **Packages / files** | optional `rethunk-tech/tap` formula (out of tree), `docs/cli.md` Distribution |
| **Traps** | Do not force tap early; keep docs demand-gated |
| **Acceptance** | Formula installs current release binary; docs link the tap when published |

---

## Round 3 — secondary carry-forwards (091204ZAUG26)

### 16. Replace Phase 0 placeholder LICENSE

**Feature.** Root `LICENSE` still says terms may change; releases publish under that text.

| | |
| --- | --- |
| **Packages / files** | `LICENSE`, `NOTICE`, README license badge/blurb if any |
| **Traps** | Legal decision — not an agent fiat; keep NOTICE attributions intact |
| **Acceptance** | Chosen license text committed; README/HUMANS link matches; no "Phase 0 placeholder" language remains |

---

## Round 4 — post-wave carry-forwards

### 21. Optional migrate MCP client to official go-sdk

**Feature.** Retries shipped in hand-rolled client; SDK `StreamableClientTransport` + `ReconnectOptions` still a larger swap.

| | |
| --- | --- |
| **Packages / files** | `internal/mcpclient`, `cmd/mcp.go`, `cmd/doctor.go` |
| **Refs** | Context7 `/modelcontextprotocol/go-sdk` |
| **Traps** | Keep protocol pin; never auto-retry `tools/call`; doctor must still initialize |
| **Acceptance** | Behaviour parity with current client + reconnection options documented |

---

## Round 7 — wave-5 audit carry-forwards (091637ZAUG26)

_(#30 shipped in wave 6.)_

---

## Round 8 — wave-6 audit carry-forwards (091652ZAUG26)

_(#34–#36 shipped in wave 7.)_

---

## Round 9 — wave-7 audit carry-forwards (091705ZAUG26)

_(#37–#40 shipped in wave 8.)_

---

## Round 10 — wave-8 audit carry-forwards (102238ZAUG26)

_(#41–#43 shipped in wave 9.)_

---

## Round 11 — wave-9 audit carry-forwards (102245ZAUG26)

_(#44–#46 shipped in wave 10; audit should-fixes closed in-wave.)_

---

## Round 12 — wave-10 audit carry-forwards (102337ZAUG26)

_(#47–#48 shipped in wave 11.)_

---

## Round 13 — wave-11 audit carry-forwards (102348ZAUG26)

_(#49–#53 shipped in wave 12; should-fixes closed in-wave.)_

---

## Round 14 — wave-12 audit carry-forwards (102354ZAUG26)

_(#54–#55 and #56a–c shipped in wave 13; watch should-fixes closed in-wave.)_

---

## Round 15 — wave-13 audit carry-forwards (110123ZAUG26)

_(#57–#60 and #61a shipped in wave 14; should-fixes closed in-wave.)_

---

## Round 16 — wave-14 audit carry-forwards (110131ZAUG26)

_(#61b–#64 shipped in wave 15; should-fixes closed in-wave.)_

---

## Round 17 — wave-15 audit carry-forwards (110204ZAUG26)

_(#65–#67 shipped in wave 16; release/label delete-mutate guards + should-fix closed in-wave.)_

---

## Round 18 — wave-16 audit carry-forwards (110218ZAUG26)

_(#68–#74 shipped in wave 17; should-fixes closed in-wave.)_

---

## Round 19 — wave-17 audit carry-forwards (110232ZAUG26)

_(#75–#76 and #77–#81 shipped in wave 18; should-fixes closed in-wave.)_

---

## Round 20 — wave-18 audit carry-forwards (110300ZAUG26)

_(#82–#88 shipped in wave 19; should-fixes closed in-wave.)_

---

## Round 21 — wave-19 audit carry-forwards (110316ZAUG26)

_(#89 shipped in wave 20; should-fixes closed in-wave.)_

---

## Round 22 — wave-20 audit carry-forwards (110324ZAUG26)

_(#93, #95–#97 shipped in wave 21; should-fixes closed in-wave.)_

---

## Round 23 — wave-21 audit carry-forwards (110340ZAUG26)

_(#98–#101 shipped in wave 22; should-fixes closed in-wave.)_

---

## Round 24 — wave-22 audit carry-forwards (110347ZAUG26)

_(#102–#107 shipped in wave 23; should-fixes closed in-wave.)_

---

## Round 25 — wave-23 audit carry-forwards (110430ZAUG26)

_(No open carry-forwards — should-fixes closed in-wave. Optional: relocate `TestTokenList_BadOutput_Hermetic` from `handler_test.go` into `more_handler_test.go` when that hot file is owned.)_
