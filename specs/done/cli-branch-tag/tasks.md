# Tasks — cli-branch-tag

Status: DONE 070526ZMAY26

## P0

- [x] [HUMAN] NOMAD ratifies Q-table (Q1–Q3).
- [x] [HUMAN] Server-side API survey: confirm branch/tag CRUD routes.
- [x] A1. Scaffold nested `repo branch` commands in `cmd/repo_branch.go`
- [x] A2. Scaffold nested `repo tag` commands in `cmd/repo_tag.go`
- [x] A3. Implement `repo branch list <repo>` (table/json/yaml, pagination).
- [x] A4. Implement `repo branch delete <repo> <name>` with `--dry-run`.
- [x] A5. Implement `repo branch set-default <repo> <name>` with `--dry-run`.
- [x] A6. Implement `repo tag list <repo>` (table/json/yaml, pagination).
- [x] A7. Implement `repo tag create <repo> <name> --ref <sha|branch>`.
- [x] A8. Implement `repo tag delete <repo> <name>` with `--dry-run`.

## P1

- [x] B1. Shell completion for repo paths, branch names, tag names.
- [x] B2. Handler tests for all happy paths and key error branches (404, 409 conflicts).

## P2

- [x] C1. [HUMAN] Live smoke: list branches, create and delete a tag.
- [x] C2. Spec close.
