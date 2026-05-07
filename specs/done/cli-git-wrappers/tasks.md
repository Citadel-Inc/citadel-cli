# Tasks — cli-git-wrappers

Status: DONE 071753ZMAY26 — Completed the deferred live smoke against src.land after the repo get stability fix landed in citadel. Verified `citadel-cli repo push --create`, `repo clone`, and `repo pull` end-to-end on disposable repository `rethunk-ai/copilot-smoke-105235`, then deleted the remote repo successfully.

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

- [x] C1. [HUMAN] Live smoke: `citadel clone` and `citadel push` against a real repo.
- [x] C2. Spec close.
