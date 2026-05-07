# Plan — cli-branch-tag

Blocked on Q-table ratification (Q1–Q3) and server-side API survey.
No implementation work should begin until P0 human tasks are checked.

## Proposed file layout

```
cmd/branch.go           — cobra commands (list, delete, set-default)
cmd/tag.go              — cobra commands (list, create, delete)
cmd/branch_test.go      — handler tests
cmd/tag_test.go         — handler tests
```
