# Spec — cli-milestones

| | |
|---|---|
| Status | DRAFT 071745ZMAY26 |
| Authored | 071745ZMAY26 |
| Owner | Bastion (J-3) |
| Carry-forward from | Phase 0 review against PoL + `citadel` tracker (2026-05-07): the milestones API (`/api/namespaces/{slug}/milestones`) shipped server-side (go-issues-milestones) but has no CLI coverage. Issue view already surfaces `milestone_title` in table output; operators cannot create or manage milestones from the terminal. |

## Why

Issues CLI (`cli-issues`) landed list/view/create/comment/close/reopen/label verbs.  
The milestone substrate is **already live** server-side (CRUD + progress tracking) but is inaccessible from the CLI surface.  
Operators demoing the Phase 0 issue workflow need at least `list` + `create` to group issues into release buckets without opening the web app.

## In scope

- `citadel-cli issue milestone list -R <ns/path> [--state open|closed|all] [--output ...]`
- `citadel-cli issue milestone view -R <ns/path> <id>` — milestone detail + progress bar
- `citadel-cli issue milestone create -R <ns/path> --title "..." [--description "..."] [--due-on YYYY-MM-DD]`
- `citadel-cli issue milestone edit -R <ns/path> <id> [--title ...] [--description ...] [--due-on ...] [--state open|closed]`
- `citadel-cli issue milestone delete -R <ns/path> <id>` — with typed-slug confirm

All verbs nest under `citadel-cli issue milestone` (second-level noun under the existing `issue` command tree).

### API mapping

| Verb | Method + Path |
|------|--------------|
| `list` | `GET /api/namespaces/{slug}/milestones` |
| `view` | `GET /api/namespaces/{slug}/milestones/{id}` |
| `create` | `POST /api/namespaces/{slug}/milestones` |
| `edit` | `PUT /api/namespaces/{slug}/milestones/{id}` |
| `delete` | `DELETE /api/namespaces/{slug}/milestones/{id}` |

### Cross-cutting
- `-R` repo/namespace selector (existing `repocontext` rules)
- Output modes: `table` (default TTY), `json`, `yaml`
- `delete` gates on `--yes` / typed-title confirm (matching `repo delete` pattern)
- Shell completion: milestone IDs via cached list for `view`/`edit`/`delete`
- `milestone_id` wiring for `issue create` (new `--milestone <id>` flag on existing `issue create`)
- List supports `--all` cursor walk (matching pagination contract)

## Out of scope

- Milestone commenting / discussion threads
- Bulk-assign issues to milestone (separate API surface if it exists)
- `--web` browser-open for milestones (no direct milestone URL in current Citadel)

## Decision log

| Q | Proposal | Status |
|---|----------|--------|
| Q1 | Nest as `issue milestone <verb>` to keep the command tree parallel to `issue label` pattern. | **Open** |
| Q2 | `id` argument accepts either UUID or title prefix (fuzzy-match with disambiguation prompt). | **Open** |
| Q3 | `due_on` is an optional date field; accept `YYYY-MM-DD` only (no relative dates). | **Open** |

## Acceptance criteria

- `issue milestone list` returns at least the milestone title and state in table output
- `issue milestone create` POSTs and echoes the new milestone ID
- `issue milestone edit` PUTs and confirms success
- `issue milestone delete` requires confirm and deletes
- `issue create --milestone <id>` wires `milestone_id` into the POST body
- httptest coverage for list/create/get/put/delete and at least one 404 path
- `docs/cli.md` milestone section added
- `make verify` passes
