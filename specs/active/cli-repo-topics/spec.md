# Spec — cli-repo-topics

| | |
|---|---|
| Status | IN_PROGRESS |
| Authored | 270640ZMAY26 |
| Owner | Copilot |
| Carry-forward from | Daemon exposes `GET/PUT .../topics` and `GET /api/topics/popular` endpoints; CLI has no topic management surface. Discovered during K.5 gap analysis. |

## Why

Repository topics are a lightweight classification mechanism that enables filtering, discovery,
and automation by language/stack/domain.  Without a CLI surface, developers must use the web UI
or raw API calls to tag repos and query what topics are popular across the platform.

## In scope

- `citadel-cli repo topic list [<namespace>/<repo>]` — print the topics attached to a repo
- `citadel-cli repo topic set [<namespace>/<repo>] <topic>...` — replace the full topic set
  (daemon PUT semantics; idempotent full-replace; no add/remove granularity needed at CLI level)
- `citadel-cli repo topic popular [--limit N]` — list platform-wide popular topics with usage count
- Standard output flags (`--output json`) on list and popular
- Tests and docs for all new commands

### API mapping

| Verb | Method + Path |
|------|---------------|
| `topic list` | `GET /api/namespaces/{ns}/repos/{repo}/topics` |
| `topic set` | `PUT /api/namespaces/{ns}/repos/{repo}/topics` body: `{"topics":["a","b"]}` |
| `topic popular` | `GET /api/topics/popular?limit=N` |

### Response shapes (daemon)

List: `{"topics":["go","cli","devtools"]}`

Set: `{"topics":["go","cli","devtools"]}` (returns the new set)

Popular: `[{"topic":"go","count":42},{"topic":"cli","count":31},...]`

### Topic validation (client-side mirror of server rules)

- Lowercase alphanumeric + hyphen; leading char must be alphanumeric; 1–50 chars
- Max 20 topics per repo; enforced server-side; client prints server error verbatim

## Out of scope

- `topic add` / `topic remove` granularity (use `topic set` with full list; matches daemon semantics)
- Cross-namespace topic search (server side not implemented yet)

## Decision log

| Q | Proposal | Status |
|---|----------|--------|
| Q1 | Nest under `repo topic` to match the existing `repo branch`, `repo tag`, `repo commit` pattern. | **Ratified 270640ZMAY26** — topics are repo-scoped; consistent with existing noun/verb structure. |
| Q2 | `set` uses positional args (`<topic>...`) rather than a flag. | **Ratified 270640ZMAY26** — topic names are simple tokens; positional is more ergonomic than repeating `--topic` flags. |
| Q3 | `popular` is a global query (not repo-scoped); nesting under `repo topic` is pragmatic rather than semantic. | **Ratified 270640ZMAY26** — discovery command belongs near the other topic commands even though it crosses repo scope; alternative `topic popular` top-level noun would be confusing. |
