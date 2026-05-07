# Spec — cli-branch-tag

| | |
|---|---|
| Status | BLOCKED 070026ZMAY26 — Waiting on HUMAN live smoke (C1) before spec close and before starting cli-deploy-tokens. |
| Authored | 120000ZMAY26 |
| Owner | Bastion |

## Why

Citadel repositories expose branch and tag metadata via the API. Today the
CLI can list/view repos but has no commands to enumerate or manipulate refs.
Teams using `citadel-cli` in release scripts need at minimum `branch list`
and `tag list/create/delete` without falling back to raw `git` calls or the
web UI.

## In scope

- `repo branch list <repo>` — list branches with HEAD commit SHA + date
- `repo branch delete <repo> <name>` — delete a branch; `--dry-run` support
- `repo branch set-default <repo> <name>` — update default branch
- `repo tag list <repo>` — list tags with SHA + date
- `repo tag create <repo> <name> --ref <commit|branch>` — create a lightweight or annotated tag
- `repo tag delete <repo> <name>` — delete a tag; `--dry-run` support
- JSON / YAML / table `--output` for list commands
- Shell completion for repo paths, branch names, tag names

## Out of scope

- Branch protection rules — separate spec
- Force-push / ref advancement — deferred to `cli-git-wrappers`
- Signed tags (GPG / SSH) — v2

## Decision log

| Q | Proposal | Status |
|---|----------|--------|
| Q1 | Use nested `repo branch` / `repo tag` commands. | RATIFIED — repository-scoped verbs keep the surface aligned with the existing `repo` tree. |
| Q2 | Default `repo tag create` to a lightweight tag; use `--message` to create an annotated tag. | RATIFIED — matches `git tag` defaults so users are not surprised. |
| Q3 | Reuse the existing read-only refs route for list operations and add the missing repo-scoped branch/tag mutation routes in `citadel`. | RATIFIED — local server-source survey confirmed `GET /api/namespaces/{parent}/repos/{repo}/refs` exists today, while branch delete, tag create/delete, and default-branch mutation routes still need implementation. |

## Blocking

Waiting on HUMAN live smoke (C1) before spec close and before starting cli-deploy-tokens.
