# Tasks — cli-ssh-keys

Status: DONE 060441ZMAY26 — SSH key surface complete: list/add/delete against /account/ssh-keys, private-key rejection, output modes, httptest coverage, docs/cli.md, live opt-in list test, and shell tab completion for delete UUIDs via cached GET /account/ssh-keys (KeySSHKeys) with PostRun invalidation after add/delete.

## P0

- [x] [HUMAN] NOMAD ratifies Q-table (Q1–Q2).
- [x] A1. Implement list/add/delete + httptest coverage for known error codes from handler.
- [x] A2. Register under `root.go`.

## P1

- [x] B1. Completion for delete IDs (optional).

## P2

- [x] C1. Live test `CITADEL_TEST_SSH_KEYS_LIVE=1`.
- [x] C2. Spec close.
