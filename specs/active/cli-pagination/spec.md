# Spec — cli-pagination

| | |
|---|---|
| Status | DRAFT 080800ZMAY26 |
| Authored | 080800ZMAY26 |
| Owner | Bastion (J-3) |
| Carry-forward from | second-pass review of `citadel-cli` (2026-05-05): silent-truncation bug in every list verb once a namespace exceeds 50 rows. Server `LIMIT 50` is hardcoded; CLI exposes no `--limit` / `--cursor`. |

## Why

Every list verb (`repo list`, `agent list`, `oauth clients list`, `namespace transfer list-pending`, `token list`) silently truncates at 50 rows. The server caps with `LIMIT 50` server-side and never advertises that there's more. Today this is invisible because nobody has 50 of anything yet. The day someone does, `repo list` quietly hides repos and the operator has no signal that they're missing data.

Need cursor-based pagination on the server, plus `--limit` / `--cursor` / `--all` flags on every list verb. Server-side counterpart is small (one cursor codec, one extension to each query); CLI side is a tabwriter / emitList shape change to surface a "next-cursor" hint when not in `--all` mode.

## In scope

### Server (Citadel) — companion repo

Tracked here for symmetry; will be split into a daemon-side spec when this one lands.

- **Cursor codec**: opaque base64-url encoding of `(created_at, id)` so cursor remains stable when rows are inserted. `pagination.NewCursor(t time.Time, id uuid.UUID) string` + `Decode`.
- **Per-endpoint paging param**: `?limit=N&cursor=<opaque>` on every list endpoint (`/api/namespaces/{slug}/repos`, `/api/agents`, `/api/agent-tokens`, `/api/orgs`, `/api/transfers/pending`, `/api/oauth/clients`).
- **Response envelope**: existing `{ "repos": [...] }` etc. envelopes get a `next_cursor` sibling. Empty / null → no more rows. Top-level shape stays backwards-compatible — old clients ignoring `next_cursor` still work.
- **Caps**: `limit` defaults to 50, max 200 (server enforced; over-limit silently clamps).

### CLI

- **Per-verb flags** added to every list:
  - `--limit N` (default 50, max 200, fails over-limit with the server's clamped value)
  - `--cursor <opaque>` (opt-in continuation)
  - `--all` (auto-paginate until `next_cursor` empty; print every row in one stream)
- **Empty-cursor handling**: when `--cursor` is set but the server returns no rows + no next-cursor, exit 0 silently (idempotent terminal page).
- **Output guarantees**: `--all` streams rows incrementally to stdout (flushes after each page) so `--output ndjson | head -N` works for big lists. Default page-by-page mode prints one tabwriter-formatted block per page + a tail line `(use --cursor <next> for more, or --all to fetch everything)`.
- **`--output json` vs `--output ndjson`**: pure JSON is per-page only (a single array per server round trip); ndjson streams across pages naturally. Document the difference.

## Out of scope

- **Backwards-pagination / random-access**: forward-only; no `prev_cursor`. v2.
- **Filtering by row attributes** (`--prefix slug=foo*`, `--created-after`). Different feature.
- **Per-namespace count summary** (`namespace stats --count repos`). Different feature.
- **Web app pagination UI** — the web app does its own list rendering today; updating its server calls is out of scope here.

## Decision log

| Q | Proposal | Status |
|---|----------|--------|
| Q1 | Cursor encoding: `(created_at, id)` vs. row offset (`limit/offset`)? | **Open** — tuple cursor; offset is unsafe under concurrent inserts. |
| Q2 | Default `--limit`: 50 (matches server today) vs. 100 (better for human ergonomics)? | **Open** — 50 to keep first-page latency stable; `--all` covers heavy users. |
| Q3 | Max `--limit` cap: 200 vs. 500? | **Open** — 200 unless an operator surfaces a use case for 500. |
| Q4 | Server response: `{ "next_cursor": "..." }` envelope vs. `Link:` HTTP header (RFC 5988)? | **Open** — envelope. Header parsing on the CLI side is a needless dep. |
| Q5 | Should `--all` cap implicitly to avoid runaway fetches? | **Open** — no implicit cap; `--all` is opt-in, document the cost. |
| Q6 | Should `--all` parallelise pages, or strictly serial? | **Open** — strictly serial. Cursor causality requires it; parallel adds confusion for marginal latency win. |

## Acceptance

- A1. Server: cursor codec + every list endpoint accepts `?limit&cursor`, returns `{...rows, next_cursor}`. (Companion daemon spec for the citadel-side delivery.)
- A2. CLI: every list verb honors `--limit`, `--cursor`, `--all`. Defaults match Q2/Q3.
- A3. CLI: `--all` streams paged output incrementally (tested under `--output ndjson` with row count > limit).
- A4. CLI: tail line under default mode prints `(use --cursor <next> for more, or --all to fetch everything)` when `next_cursor` is non-empty.
- A5. Tests: handler-level tests with a 3-page fixture verifying both `--limit` truncation, `--cursor` continuation, and `--all` exhaustion.
- A6. Q-table ratified.
- A7. Live integration test (gated on `CITADEL_TEST_PAGINATION_LIVE=1`) walks 200+ rows on a real test instance.
