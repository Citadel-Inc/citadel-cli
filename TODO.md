# TODO — follow-on and carry-forward

Planning backlog only. Parked/rejected specs (`specs/parked/`) and live-smoke-only residuals stay out unless reopened by product decision.

Do **not** restore `account passkey` / `account device` — removed deliberately in `b1871a6` (settings-panel, not a developer workflow). Phase B MFA recovery under that tree is obsolete with the removal.

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

### 6. Device-code OAuth login (RFC 8628)

**Feature.** Headless / SSH-only hosts without loopback browser still need an interactive login path beyond pasting a JWT.

| | |
| --- | --- |
| **Packages / files** | `cmd/auth.go`, `internal/clicfg`, `docs/cli.md` (First-run / headless) |
| **Upstream** | Citadel `POST /api/oauth/device` (RFC 8628) + `urn:ietf:params:oauth:grant-type:device_code` on `/api/oauth/token` — already live in `citadel/internal/api/authapi` |
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

---

## Round 2 — server-ahead CLI gaps (091204ZAUG26)

Cross-checked against `Citadel-Inc/citadel` HTTP surface. Prefer developer/operator workflows; skip settings-panel and parked items.

### 8. Release asset upload / download / delete

**Feature.** Daemon already serves release asset CRUD under `/api/namespaces/{slug}/releases/{tag}/assets`; CLI `release` only covers release metadata.

| | |
| --- | --- |
| **Packages / files** | `cmd/release.go`, `internal/apiclient`, `docs/cli.md` (releases) |
| **Upstream** | `citadel/internal/api/releasesapi` (`handleCreateAsset` / `handleListAssets` / `handleGetAsset` / `handleDeleteAsset`); Spaces-backed store when configured |
| **Traps** | Multipart/stream upload size caps (`CITADEL_RELEASE_ASSET_MAX_BYTES`); store may be nil → clear CLI error; download must be binary-safe to file/stdout; completion for asset IDs |
| **Acceptance** | `release asset list | upload | download | delete` (or nested verbs) round-trip against a tagged release; human mode never dumps binary to TTY without redirect |

### 10. Gist command group (`gh gist` parity)

**Feature.** Full `gistapi` is live on the daemon; CLI has zero gist verbs.

| | |
| --- | --- |
| **Packages / files** | new `cmd/gist.go` (+ registration in `cmd/root.go`), completion helpers, `docs/cli.md` |
| **Upstream** | `citadel/internal/api/gistapi` (`/api/gists*`, promote, raw, Atom feeds) |
| **Traps** | Visibility (public/secret); raw rate limits; promote-to-repo needs `ReposDir` semantics; secret scanning on write; keep scope to create/list/view/edit/delete/clone-raw before comments/feeds |
| **Acceptance** | Core CRUD + raw/download works with standard `--output`; docs show a `gh gist`-shaped cookbook |

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

## Round 4 — post-wave carry-forwards (091210ZAUG26)

Optional / polish from wave audits. Not blocking.

### 18. Friendlier `api --input` JSON validation

**Feature.** Empty/invalid `--input` fails at apiclient marshal (`unexpected end of JSON input`); prefer an upfront `--input: invalid JSON` error.

| | |
| --- | --- |
| **Packages / files** | `cmd/api.go`, `cmd/api_handler_test.go` |
| **Acceptance** | Invalid/empty body errors name `--input` before any HTTP round-trip |

### 19. `project admin recovery-scan` confirm-mismatch test

**Feature.** Happy/`--yes`/403 covered; typed-confirm rejection path untested.

| | |
| --- | --- |
| **Packages / files** | `cmd/project_admin_test.go` |
| **Acceptance** | Wrong typed value exits non-zero without POSTing |

### 20. Harden `make install` when man generation yields no `*.1`

**Feature.** `install -m 644 …/*.1` fails under `set -u` if the glob is empty.

| | |
| --- | --- |
| **Packages / files** | `Makefile` |
| **Acceptance** | Clear error if man generation produces zero pages; no shell glob explode |

### 21. Optional migrate MCP client to official go-sdk

**Feature.** Retries shipped in hand-rolled client; SDK `StreamableClientTransport` + `ReconnectOptions` still a larger swap.

| | |
| --- | --- |
| **Packages / files** | `internal/mcpclient`, `cmd/mcp.go`, `cmd/doctor.go` |
| **Refs** | Context7 `/modelcontextprotocol/go-sdk` |
| **Traps** | Keep protocol pin; never auto-retry `tools/call`; doctor must still initialize |
| **Acceptance** | Behaviour parity with current client + reconnection options documented |

### 22. Document `oauth clients list --dcr` in cli.md

**Feature.** Flag shipped; docs may still omit the client-side filter note.

| | |
| --- | --- |
| **Packages / files** | `docs/cli.md` (OAuth clients) |
| **Acceptance** | One-line note: `--dcr` filters locally; incompatible with `--watch` |
