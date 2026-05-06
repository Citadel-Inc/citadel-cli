# Tasks — cli-git-wrappers

Status: DRAFT

## P0

- [ ] [HUMAN] NOMAD ratifies Q-table (Q1–Q4).
- [ ] A1. Implement auth injection mechanism (GIT_ASKPASS or credential helper).
- [ ] A2. Implement `citadel clone <repo-path> [<local-dir>]`.
- [ ] A3. Implement `citadel push` (detect current repo remote or accept --remote).
- [ ] A4. Implement `citadel pull` (detect current repo remote or accept --remote).

## P1

- [ ] B1. Tests: mock exec calls to verify correct git invocation and env injection.
- [ ] B2. Error handling: friendly message when `git` binary not found on PATH.
- [ ] B3. Shell completion: repo path completion for `clone`.

## P2

- [ ] C1. [HUMAN] Live smoke: `citadel clone` and `citadel push` against a real repo.
- [ ] C2. Spec close.
