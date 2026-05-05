# Tasks — cli-issue-pr

Status: DRAFT 081550ZMAY26

Blocked by: daemon-side issue + PR HTTP APIs (see plan.md for survey notes; may need a citadel-side companion spec depending on Q5).

## P0

- [ ] [HUMAN] NOMAD ratifies Q-table (Q1-Q6).
- [ ] A1. Survey daemon-side issue + PR HTTP endpoints. Document gaps in plan.md. If PR API absent, file companion spec for daemon-side delivery before A4.
- [ ] A2. CLI: `cmd/issue.go` scaffolding — IssueCmd parent + list/view/create/comment/close/reopen/label subcommand structs with --help text only.
- [ ] A3. CLI: implement issue list / view / create against daemon API.

## P1

- [ ] B1. CLI: implement issue close / reopen / comment / label.
- [ ] B2. CLI: `cmd/pr.go` parent + list/view/create/merge/close (or defer per Q5).
- [ ] B3. Body-editor support: `$EDITOR` invocation when `--body` omitted under a TTY; stdin read when piped.
- [ ] B4. `--web` flag on view verbs (issue + pr): opens browser via existing openBrowser helper.
- [ ] B5. Pagination integration: list verbs honor flags from cli-pagination spec.
- [ ] B6. Output-format integration: every verb honors --output json|ndjson|csv|yaml from cli-output-formats spec.
- [ ] B7. Tests: handler-level happy-path + error path per verb against httptest fixture.

## P2

- [ ] C1. Live integration test (`CITADEL_TEST_ISSUES_LIVE=1`) create+comment+close round trip.
- [ ] C2. README + HUMANS.md: new "Issues + PRs" section.
- [ ] C3. [HUMAN] Operator smoke: file a real issue against a test repo from terminal.
- [ ] C4. Spec close.
