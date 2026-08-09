# TODO — follow-on and carry-forward

Planning backlog only. Parked/rejected specs (`specs/parked/`) and live-smoke-only residuals stay out unless reopened by product decision.

Do **not** restore `account passkey` / `account device` — removed deliberately in `b1871a6` (settings-panel, not a developer workflow). Phase B MFA recovery under that tree is obsolete with the removal.

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

## Round 1 — session continuity and list UX (091204ZAUG26)

### 1. Wire OAuth refresh-token exchange

**Feature.** `refresh_token` is persisted on login but never used to mint a new access token; sessions past JWT expiry force a full re-login (or env JWT).

| | |
| --- | --- |
| **Packages / files** | `internal/clicfg/clicfg.go` (`Load` comment ~53–58), `cmd/auth.go` (token exchange), `cmd/client.go` (`errSessionExpired`), `docs/cli.md` (auth / 401 recovery) |
| **Upstream** | Citadel `/api/oauth/token` refresh grant (same client as PKCE login) |
| **Traps** | Agent-token path already does one-shot 401 rotation — do not double-refresh or clobber agent binding; never log refresh tokens under `--debug-http`; race between concurrent commands rewriting `config.toml` |
| **Acceptance** | After access JWT expiry with a valid stored refresh token, a REST command succeeds without interactive login; failed refresh yields `auth_required` and clears stale secrets; docs describe the behaviour |

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

### 7. Repo raw blob download

**Feature.** `repo browse blob` prints text / skips binaries; authenticated `/raw?ref=&path=` streaming download was deferred.

| | |
| --- | --- |
| **Packages / files** | `cmd/repo_browse.go`, `internal/apiclient` (`GetStream` now available), `docs/cli.md` (browse) |
| **Carry-from** | `specs/done/cli-repo-browse/spec.md` Out of scope |
| **Traps** | Reuse `GetStream` (absolute/same-origin); binary-safe; progress optional but must not corrupt pipes |
| **Acceptance** | `repo browse raw` (or flag) writes bytes for large/binary paths; `--output` modes documented |

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

### 17. Doctor: API host coercion + git remote sanity

**Feature.** Doctor checks server `/healthz`, token presence, MCP init, config mode — not REST host coercion (`api.` vs MCP host) or CWD git remote inference.

| | |
| --- | --- |
| **Packages / files** | `cmd/doctor.go`, `cmd/repocontext.go`, `cmd/client.go` (host routing) |
| **Traps** | Keep checks read-only; WARN not FAIL for missing git remote; do not require network beyond existing probes |
| **Acceptance** | Doctor reports resolved REST base vs MCP base; optional WARN when CWD origin is non-Citadel while `CITADEL_REPO` unset |

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

## Round 5 — wave-3 audit optional follow-ons (091358ZAUG26)

### 23. Gist raw TTY / buffering polish

**Feature.** `gist raw` buffers the full body and has no TTY binary refusal (upstream forces `text/plain`, but large files still buffer).

| | |
| --- | --- |
| **Packages / files** | `cmd/gist.go` |
| **Acceptance** | Stream to stdout/file; optional TTY guard parity with release asset download |

### 24. KG handler-level `--all` tests

**Feature.** Pagination/table unit-tested; no multi-page httptest handler test for `kg search`/`symbols`/`files`/`fulltext`.

| | |
| --- | --- |
| **Packages / files** | `cmd/kg_extended_test.go` (or handler test) |
| **Acceptance** | Multi-page `--all` asserts multiple GETs and merged output |

### 25. Soften PKCE token-exchange error bodies

**Feature.** `exchangePKCECode` embeds full HTTP body in errors; prefer truncated/redacted OAuth error fields.

| | |
| --- | --- |
| **Packages / files** | `cmd/auth.go` |
| **Acceptance** | Failed exchange errors stay actionable without dumping full payloads |
