# TODO — follow-on and carry-forward

Planning backlog only. Parked/rejected specs (`specs/parked/`) and live-smoke-only residuals stay out unless reopened by product decision.

Do **not** restore `account passkey` / `account device` — removed deliberately in `b1871a6` (settings-panel, not a developer workflow). Phase B MFA recovery under that tree is obsolete with the removal.

---

## Round 1 — session continuity and list UX (091204ZAUG26)

### 1. Wire OAuth refresh-token exchange

**Feature.** `refresh_token` is persisted on login but never used to mint a new access token; sessions past JWT expiry force a full re-login (or env JWT).

| | |
| --- | --- |
| **Packages / files** | `internal/clicfg/clicfg.go` (`Load` comment ~53–58), `cmd/auth.go` (token exchange), `cmd/client.go` (`errSessionExpired`), `docs/cli.md` (auth / 401 recovery) |
| **Upstream** | Citadel `/api/oauth/token` refresh grant (same client as PKCE login) |
| **Traps** | Agent-token path already does one-shot 401 rotation — do not double-refresh or clobber agent binding; never log refresh tokens under `--debug-http`; race between concurrent commands rewriting `config.toml` |
| **Acceptance** | After access JWT expiry with a valid stored refresh token, a REST command succeeds without interactive login; failed refresh yields `auth_required` and clears stale secrets; docs describe the behaviour |

### 2. KG list polish (`cli-kg-extended` P1 B1/B2)

**Feature.** Cursor flags exist on search/symbols/files/fulltext but `--all` is rejected; human `--output table` is unsupported (json/yaml only).

| | |
| --- | --- |
| **Packages / files** | `cmd/kg_extended.go`, `cmd/pagination.go`, `cmd/output.go`, `docs/cli.md` (Knowledge graph) |
| **Carry-from** | `specs/done/cli-kg-extended/tasks.md` P1 B1/B2 |
| **Traps** | Only enable `--all` where the daemon emits `next_cursor`; table columns must stay stable for scripting like other CSV/table contracts |
| **Acceptance** | `--all` walks pages for endpoints that advertise cursors; `--output table` renders symbols/search rows; json/yaml unchanged |

### 3. Audit event tail / `--watch`

**Feature.** `cli-audit` deferred `--follow` until SSE; `cli-watch` now owns SSE for many list verbs, but `audit list` has no watch flag.

| | |
| --- | --- |
| **Packages / files** | `cmd/audit.go`, `cmd/watch.go`, `internal/sseclient`, `docs/cli.md` / `HUMANS.md` (audit) |
| **Carry-from** | `specs/done/cli-audit/tasks.md` C4; `specs/done/cli-audit/spec.md` A6 |
| **Traps** | Confirm daemon SSE on audit list path before wiring; filter flags (`--kind`, `--namespace`, `--since`) must apply to the stream; do not invent a second transport beside existing watch helpers |
| **Acceptance** | `audit list --watch` (and ndjson mode) streams events until interrupt; missing server SSE fails with a clear error, not a hang |

### 4. Extend `--watch` to high-churn workflow lists

**Feature.** Watch is on repo/agent/token/oauth/namespace lists; not on `issue list`, `pr list`, or `notification list`.

| | |
| --- | --- |
| **Packages / files** | `cmd/issue.go`, `cmd/pr.go`, `cmd/notification.go`, `cmd/watch.go` / `watch_table.go` |
| **Traps** | Needs matching server SSE; new `watchListKind` + table redraw rows; reject `--output json` with the same hint as other watch verbs |
| **Acceptance** | Documented `--watch` on issue/PR/notification lists when server supports them; behaviour matches `repo list --watch` |

### 5. Scrub stale account-security docs

**Feature.** Docs/CHANGELOG still describe removed `account passkey` / `account device` verbs.

| | |
| --- | --- |
| **Packages / files** | `docs/cli.md` (§ Account security), `CHANGELOG.md` (v0.1.0 Added bullet), optionally `specs/done/cli-account-security` resolution note |
| **Traps** | Do not reintroduce the command tree; keep SSH keys / auth providers docs intact |
| **Acceptance** | No user-facing doc claims those verbs exist; points operators to the web settings panel |

### 6. Device-code OAuth login (RFC 8628)

**Feature.** Headless / SSH-only hosts without loopback browser still need an interactive login path beyond pasting a JWT.

| | |
| --- | --- |
| **Packages / files** | `cmd/auth.go`, `internal/clicfg`, `docs/cli.md` (First-run / headless) |
| **Carry-from** | `specs/done/cli-oauth-login/spec.md` (device-code OOS) |
| **Traps** | Server must expose device authorization endpoints; UX shows user-code + verification URL; poll with backoff; do not weaken PKCE browser path |
| **Acceptance** | `auth login --device` (or equivalent) completes on a machine without a local browser; stores the same agent-token shape as PKCE login |

### 7. Repo raw blob download

**Feature.** `repo browse blob` prints text / skips binaries; authenticated `/raw?ref=&path=` streaming download was deferred.

| | |
| --- | --- |
| **Packages / files** | `cmd/repo_browse.go`, `internal/apiclient`, `docs/cli.md` (browse) |
| **Carry-from** | `specs/done/cli-repo-browse/spec.md` Out of scope |
| **Traps** | Stream to stdout/file without buffering whole objects; respect auth; binary-safe; progress optional but must not corrupt pipes |
| **Acceptance** | `repo browse raw` (or flag) writes bytes for large/binary paths; `--output` modes documented |

---

## Explicitly not tracked here

| Item | Why |
| ------ | ----- |
| `cli-webhook-test`, billing, avatar, privacy, account-export, mcp-stdio/stream | `specs/parked/` — rejected or superseded |
| Restoring `account *` security verbs | Deliberate removal `b1871a6` |
| Live-smoke / C1-only residuals on done specs | Env-gated operator work, not product features |
| shadcn / web component work | This repo is a Go Cobra CLI; browser UX lives in Citadel web |
