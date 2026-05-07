# Tasks — cli-git-wrappers

Status: BLOCKED 071706ZMAY26 — Live Citadel repo smoke is blocked on backend inconsistency: `repo create` returns success, but repeated `repo get` calls for the same slug still return not_found and the git HTTPS endpoint remains unusable for push/clone.

## P0

- [x] [HUMAN] NOMAD ratifies Q-table (Q1–Q4).
- [x] A1. Implement auth injection mechanism (short-lived `GIT_ASKPASS` helper).
- [x] A2. Implement `citadel repo clone <repo-path> [<local-dir>]`.
- [x] A3. Implement `citadel repo push` (detect current repo remote or accept --remote; prompt/create missing remote repo).
- [x] A4. Implement `citadel repo pull` (detect current repo remote or accept --remote).

## P1

- [x] B1. Tests: mock exec calls to verify correct git invocation and env injection.
- [x] B2. Error handling: friendly message when `git` binary not found on PATH.
- [x] B3. Shell completion: repo path completion for `clone`.

## P2

- [ ] C1. [HUMAN] Live smoke: `citadel clone` and `citadel push` against a real repo.
- [ ] C2. Spec close.
