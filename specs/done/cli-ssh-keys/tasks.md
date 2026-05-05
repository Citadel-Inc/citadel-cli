# Tasks — cli-ssh-keys

Status: DONE 052318ZMAY26 — Shipped top-level `citadel-cli ssh-key` (list/add/delete) against `/account/ssh-keys`, with public-key-only uploads, private-key rejection, output formats on list, `--output json` on add, httptest matrix including 400/404 paths, `docs/cli.md`, and opt-in `TestLiveSshKeys_list_optIn`. Left P1 B1 (optional delete-ID completion) deliberately open.

## P0

- [x] [HUMAN] NOMAD ratifies Q-table (Q1–Q2).
- [x] A1. Implement list/add/delete + httptest coverage for known error codes from handler.
- [x] A2. Register under `root.go`.

## P1

- [ ] B1. Completion for delete IDs (optional).

## P2

- [x] C1. Live test `CITADEL_TEST_SSH_KEYS_LIVE=1`.
- [x] C2. Spec close.
