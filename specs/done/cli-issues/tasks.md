# Tasks — cli-issues

Status: DONE 070107ZMAY26 — Operator smoke completed via citadel-cli against live namespace rethunk: created issue #1, added a comment, closed it, and verified close-refs. REST routing fix landed in citadel-cli so the live API resolves through api.src.land while OAuth/MCP remain on mcp.src.land.

## P0

- [x] [HUMAN] Replace the stale `cli-issue-pr` framing with an issues-only spec.
- [x] A1. Survey daemon issue endpoints (`/api/issues/*`, `/api/labels/*`, `/close-refs`) and pin the exact request/response shapes in `plan.md`.
- [x] A2. Scaffold `cmd/issue.go`: IssueCmd parent + `list`, `view`, `create`, `comment`, `close`, `reopen`, `label`, `close-refs`.
- [x] A3. Implement `issue list` and `issue view`.

## P1

- [x] B1. Implement `issue create` + `issue comment`.
- [x] B2. Implement `issue close` + `issue reopen`.
- [x] B3. Implement `issue label` and `issue close-refs`.
- [x] B4. TTY `$EDITOR` / stdin body handling for create + comment.
- [x] B5. Integrate `-R`, pagination, and output-mode contracts.
- [x] B6. Add `--web` on `issue view`.
- [x] B7. Add handler / httptest coverage across read + write verbs.

## P2

- [x] C1. Env-gated live integration test (`CITADEL_TEST_ISSUES_LIVE=1`) covering create + comment + close + close-refs.
- [x] C2. README / HUMANS / docs/cli.md updates for the new issue surface.
- [x] C3. [HUMAN] Operator smoke: file and manage a real namespace issue from the terminal.
- [x] C4. Spec close.
