# Spec — cli-branch-tag

| | |
|---|---|
| Status | DRAFT |
| Authored | 120000ZMAY26 |
| Owner | unassigned |

## Why

Citadel repositories expose branch and tag metadata via the API. Today the
CLI can list/view repos but has no commands to enumerate or manipulate refs.
Teams using `citadel-cli` in release scripts need at minimum `branch list`
and `tag list/create/delete` without falling back to raw `git` calls or the
web UI.

## In scope

- `branch list <repo>` — list branches with HEAD commit SHA + date
- `branch delete <repo> <name>` — delete a branch; `--dry-run` support
- `branch set-default <repo> <name>` — update default branch
- `tag list <repo>` — list tags with SHA + date
- `tag create <repo> <name> --ref <commit|branch>` — create a lightweight or annotated tag
- `tag delete <repo> <name>` — delete a tag; `--dry-run` support
- JSON / YAML / table `--output` for list commands
- Shell completion for repo paths, branch names, tag names

## Out of scope

- Branch protection rules — separate spec
- Force-push / ref advancement — deferred to `cli-git-wrappers`
- Signed tags (GPG / SSH) — v2

## Decision log

| Q | Proposal | Status |
|---|----------|--------|
| Q1 | Top-level `branch`/`tag` vs. nested `repo branch`/`repo tag`? | OPEN — top-level preferred for discoverability; nesting mirrors GitHub CLI |
| Q2 | Annotated vs. lightweight tags at `tag create`? | OPEN — recommend `--message` flag gates annotated; absence = lightweight |
| Q3 | API endpoints — needs server-side survey | OPEN |
