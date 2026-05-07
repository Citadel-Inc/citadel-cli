# Spec — cli-repo-commits

| | |
|---|---|
| Status | IN_PROGRESS |
| Authored | 130512ZMAY26 |
| Owner | Copilot |
| Carry-forward from | Dossier appendix-K K.8 commit.* group: daemon already exposes commit list, detail, and per-file diff routes; citadel-cli has no commit query surface. |

## Why

Citadel repositories expose a commit graph through the daemon's browse API.  
Developers and scripts frequently need to inspect recent commits, retrieve a single commit's metadata and file stats, and read the unified diff for a changed file — all without leaving the terminal or cloning the repo locally.  
GitHub CLI (`gh api`) provides this today only through raw API calls; a first-class `repo commit` subtree gives a workflow-aligned interface consistent with the existing `repo branch` and `repo tag` surfaces.

## In scope

- `citadel-cli repo commit list [<namespace>/<repo>]` — paginated commit log for a ref
- `citadel-cli repo commit get [<namespace>/<repo>] <sha>` — single commit detail (message, parents, per-file stats, signature presence)
- `--ref` flag on `list` to target a branch or tag (defaults to repo default branch)
- `--path` flag on `list` to filter commits that touch a specific file
- Standard pagination flags (`--limit`, `--cursor`, `--all`) on `list`
- Standard output flags (`--output json|yaml|csv|ndjson`) on `list`; `--output json|yaml` on `get`
- `--path` flag on `get` to print the per-file unified diff for a single file (calls `/commits/{sha}/diff`)
- Tests and docs for all new commands

### API mapping

| Verb | Method + Path |
|------|---------------|
| `list` | `GET /api/namespaces/{ns}/repos/{repo}/commits?ref=&path=&limit=&after=` |
| `get` | `GET /api/namespaces/{ns}/repos/{repo}/commits/{sha}` |
| `get --path <file>` | `GET /api/namespaces/{ns}/repos/{repo}/commits/{sha}/diff?path=` |

## Out of scope

- `commit.search` (cross-repo message search) — no daemon endpoint yet; deferred
- `commit.diff` from_sha..to_sha range diff — no daemon endpoint yet; deferred
- Commit creation / push (git-level operation delegated to `repo git push`)
- `branch.create` / `branch.merge` — no daemon write endpoint exists yet; deferred

## Decision log

| Q | Proposal | Status |
|---|----------|--------|
| Q1 | Nest under `repo commit` rather than a top-level `commit` noun. | **Ratified 130512ZMAY26** — commits are repo-scoped; mirrors `repo branch` and `repo tag` patterns already in the CLI. |
| Q2 | `list` returns `sha`, `author`, `timestamp`, and the first line of `message` (subject) in human table; full message in JSON. | **Ratified 130512ZMAY26** — aligns with `git log --oneline` ergonomics and avoids truncation ambiguity in tabular view. |
| Q3 | `get --path <file>` prints raw unified diff to stdout (plain text, not JSON). When `--output json` is also set, wrap in `{ "unified": "..." }`. | **Ratified 130512ZMAY26** — raw diff is more useful for shell piping; JSON mode keeps the API consistent. |
| Q4 | `after` cursor from the daemon is passed through opaque (base64 string); the CLI does not decode it. | **Ratified 130512ZMAY26** — matches existing cursor handling in webhook and notification pagination. |
