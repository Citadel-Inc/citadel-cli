# Tasks — cli-issues

Status: DRAFT 062323ZMAY26

## P0

- [x] [HUMAN] Replace the stale `cli-issue-pr` framing with an issues-only spec.
- [ ] A1. Survey daemon issue endpoints (`/api/issues/*`, `/api/labels/*`, `/close-refs`) and pin the exact request/response shapes in `plan.md`.
- [ ] A2. Scaffold `cmd/issue.go`: IssueCmd parent + `list`, `view`, `create`, `comment`, `close`, `reopen`, `label`, `close-refs`.
- [ ] A3. Implement `issue list` and `issue view`.

## P1

- [ ] B1. Implement `issue create` + `issue comment`.
- [ ] B2. Implement `issue close` + `issue reopen`.
- [ ] B3. Implement `issue label` and `issue close-refs`.
- [ ] B4. TTY `$EDITOR` / stdin body handling for create + comment.
- [ ] B5. Integrate `-R`, pagination, and output-mode contracts.
- [ ] B6. Add `--web` on `issue view`.
- [ ] B7. Add handler / httptest coverage across read + write verbs.

## P2

- [ ] C1. Env-gated live integration test (`CITADEL_TEST_ISSUES_LIVE=1`) covering create + comment + close + close-refs.
- [ ] C2. README / HUMANS / docs/cli.md updates for the new issue surface.
- [ ] C3. [HUMAN] Operator smoke: file and manage a real namespace issue from the terminal.
- [ ] C4. Spec close.
