# TODO — follow-on and carry-forward

Planning backlog only. Parked/rejected specs (`specs/parked/`) and live-smoke-only residuals stay out unless reopened by product decision.

Do **not** restore `account passkey` / `account device` — removed deliberately in `b1871a6` (settings-panel, not a developer workflow). Phase B MFA recovery under that tree is obsolete with the removal.

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

### 44. Hermetic config for gist negative-offset test

**Polish.** `TestGistList_NegativeOffset` hits `newAPIClient` before the offset guard; a bad host config can fail the test before the assertion. Same latent shape as webhook deliveries negative-offset tests.

| | |
| --- | --- |
| **Packages / files** | `cmd/gist_test.go` (optionally mirror in webhook negative-offset tests) |
| **Acceptance** | `t.Setenv("XDG_CONFIG_HOME", t.TempDir())` (or shared helper) so the guard assertion is hermetic |

### 45. Gist list negative `--limit` guard

**Polish.** `gist list` only applies `limit > 0`; negatives are silently dropped. Wave 9 deliberately scoped offset-only.

| | |
| --- | --- |
| **Packages / files** | `cmd/gist.go`, `cmd/gist_test.go` |
| **Acceptance** | Negative `--limit` errors with `--limit cannot be negative` (audit/webhook wording) |

### 46. Webhook list builders prefer `strconv.Itoa`

**Polish.** Repo/namespace webhook list still use `fmt.Sprintf("%d", limit|offset)` at list sites; delivery completion fetch already uses `strconv.Itoa`.

| | |
| --- | --- |
| **Packages / files** | `cmd/webhook.go` (list handlers ~358/648/653); optionally `cmd/gist.go` list query builders |
| **Acceptance** | List query builders use `strconv.Itoa` consistently with sibling verbs |
