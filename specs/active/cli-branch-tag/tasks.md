# Tasks — cli-branch-tag

Status: DRAFT

## P0

- [ ] [HUMAN] NOMAD ratifies Q-table (Q1–Q3).
- [ ] [HUMAN] Server-side API survey: confirm branch/tag CRUD routes.
- [ ] A1. Scaffold `cmd/branch.go` with `branch` top-level command and
  subcommands: list, delete, set-default.
- [ ] A2. Scaffold `cmd/tag.go` with `tag` top-level command and
  subcommands: list, create, delete.
- [ ] A3. Implement `branch list <repo>` (table/json/yaml, pagination).
- [ ] A4. Implement `branch delete <repo> <name>` with `--dry-run`.
- [ ] A5. Implement `branch set-default <repo> <name>` with `--dry-run`.
- [ ] A6. Implement `tag list <repo>` (table/json/yaml, pagination).
- [ ] A7. Implement `tag create <repo> <name> --ref <sha|branch>`.
- [ ] A8. Implement `tag delete <repo> <name>` with `--dry-run`.

## P1

- [ ] B1. Shell completion for repo paths, branch names, tag names.
- [ ] B2. Handler tests for all happy paths and key error branches (404, 409 conflicts).

## P2

- [ ] C1. [HUMAN] Live smoke: list branches, create and delete a tag.
- [ ] C2. Spec close.
