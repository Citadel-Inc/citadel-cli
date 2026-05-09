# Tasks — cli-pr-inline-comments

Status: DONE 091518ZMAY26 — Shipped inline PR comment flags (--diff-file, --diff-line, --diff-side, --diff-sha, --thread-id) on `pr comment add`; client-side validation for paired flags and side values; --inline/--general filters and thread-grouped human output on `pr comment list`; --diff-file tab completion via diff endpoint; handler tests covering all paths; docs/cli.md updated. make verify passes. C1 (live smoke) deferred — gates on CITADEL_TEST_PR_INLINE_LIVE=1.
Priority: Medium

## P0

- [x] A1. Add `--diff-file`, `--diff-line`, `--diff-side`, `--diff-sha`, `--thread-id` flags to `pr comment add`; validate paired (diff-file+diff-line together or neither).
- [x] A2. `--diff-side` defaults to `"right"` when `--diff-file` is set; validate `left`/`right`; error on invalid value.
- [x] A3. `pr comment list` — add `--inline` / `--general` filter flags; group inline comments under thread headers in human output.
- [x] A4. Handler tests for inline add, thread reply, validation errors (400), filtered list.

## P1

- [x] B1. `docs/cli.md` PR comment section updated with inline usage examples.
- [x] B2. `make verify` passes (vet, race tests, golangci-lint).

## P2

- [ ] C1. Live smoke test behind `CITADEL_TEST_PR_INLINE_LIVE=1`.
- [ ] C2. Spec close.
- [x] D1. `--diff-file` flag completion: fetch changed-file list from `pr diff` endpoint and register via `RegisterFlagCompletionFunc`.
