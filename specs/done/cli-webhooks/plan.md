# Plan — cli-webhooks

Blocked on Q-table ratification (Q1–Q4) and server-side API survey.
No implementation work should begin until P0 human tasks are checked.

## Proposed file layout

```
cmd/webhook.go          — cobra commands (list, create, get, delete, test)
cmd/webhook_test.go     — handler tests
```
