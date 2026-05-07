# Tasks — cli-branch-tag

Status: IN_PROGRESS 070012ZMAY26 — Bastion claims execution

## P0

- [x] [HUMAN] NOMAD ratifies Q-table (Q1–Q3).
- [x] [HUMAN] Server-side API survey: confirm branch/tag CRUD routes.
- [ ] A1. Scaffold nested `repo branch` commands in `cmd/repo_branch.go`
- [ ] A2. Scaffold nested `repo tag` commands in `cmd/repo_tag.go`
- [ ] A3. Implement `repo branch list <repo>` (table/json/yaml, pagination).
- [ ] A4. Implement `repo branch delete <repo> <name>` with `--dry-run`.
- [ ] A5. Implement `repo branch set-default <repo> <name>` with `--dry-run`.
- [ ] A6. Implement `repo tag list <repo>` (table/json/yaml, pagination).
- [ ] A7. Implement `repo tag create <repo> <name> --ref <sha|branch>`.
- [ ] A8. Implement `repo tag delete <repo> <name>` with `--dry-run`.

## P1

- [ ] B1. Shell completion for repo paths, branch names, tag names.
- [ ] B2. Handler tests for all happy paths and key error branches (404, 409 conflicts).

## P2

- [ ] C1. [HUMAN] Live smoke: list branches, create and delete a tag.
- [ ] C2. Spec close.
