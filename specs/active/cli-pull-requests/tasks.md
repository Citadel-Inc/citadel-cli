# Tasks — cli-pull-requests

Status: IN_PROGRESS 080823ZMAY26 — Bastion (J-3) claims execution
Priority: High

## P0

- [x] A1. `pr list -R org/repo` — paginated, `--state`, `--output json|yaml|table|csv|ndjson`.
- [x] A2. `pr view -R org/repo <number>` — PR detail + reviewers, `--output`.
- [x] A3. `pr create -R org/repo` — `--title`, `--body` (TTY/$EDITOR fallback), `--source`, `--target`.
- [x] A4. `pr close -R org/repo <number>` — `--yes` bypass.
- [x] A5. `pr merge -R org/repo <number>` — friendly errors for merge_conflict, approval_required, already_merged, invalid_state.
- [x] A6. `pr diff -R org/repo <number>` — raw unified diff; `--file <path>` for single-file variant.
- [x] A7. `pr check -R org/repo <number>` — mergeability result, human + `--output json`.
- [x] A8. `pr comment list/add -R org/repo <number>`.
- [x] A9. `pr reviewer list/add -R org/repo <number>`.
- [x] A10. `pr review -R org/repo <number> --approve/--request-changes/--comment <body>`.
- [x] A11. Handler tests for all verbs including error paths.

## P1

- [x] B1. `docs/cli.md` updated with pull request subcommand reference.
- [x] B2. `make verify` passes (vet, race tests, golangci-lint).

## P2

- [ ] C1. Live smoke test behind `CITADEL_TEST_PR_LIVE=1`.
- [ ] C2. Spec close.
