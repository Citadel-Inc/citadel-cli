# Plan — cli-webhook-test

Q-table pending NOMAD ratification. Defaults assumed:

- Q1: `repo webhook test` and `namespace webhook test` — matches existing CRUD shape in `cmd/webhook.go`
- Q2: server route — survey `go-issues-webhooks` at task A1; expected pattern is `POST /api/namespaces/{slug}/webhooks/{id}/test`
- Q3: human output — show status line + HTTP response code + latency; `--output json` returns raw server response

## Proposed file layout

```
cmd/webhook.go    — add `test` subcommand to existing repoWebhookCmd and nsWebhookCmd trees
cmd/webhook_test.go — extend existing handler tests
docs/cli.md       — update webhook section; remove "does not yet wrap" note
```

No new files needed; `test` is a subcommand of the existing webhook command groups.
