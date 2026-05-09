# Tasks — cli-pr-inline-comments

Status: IN_PROGRESS 091320ZMAY26 — Bastion (J-3) claims execution
Priority: Medium

## P0

- [ ] A1. Add `--diff-file`, `--diff-line`, `--diff-side`, `--diff-sha`, `--thread-id` flags to `pr comment add`; validate paired (diff-file+diff-line together or neither).
- [ ] A2. `--diff-side` defaults to `"right"` when `--diff-file` is set; validate `left`/`right`; error on invalid value.
- [ ] A3. `pr comment list` — add `--inline` / `--general` filter flags; group inline comments under thread headers in human output.
- [ ] A4. Handler tests for inline add, thread reply, validation errors (400), filtered list.

## P1

- [ ] B1. `docs/cli.md` PR comment section updated with inline usage examples.
- [ ] B2. `make verify` passes (vet, race tests, golangci-lint).

## P2

- [ ] C1. Live smoke test behind `CITADEL_TEST_PR_INLINE_LIVE=1`.
- [ ] C2. Spec close.
