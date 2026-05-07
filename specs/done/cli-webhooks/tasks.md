# Tasks — cli-webhooks

Status: DRAFT

## P0

- [ ] [HUMAN] NOMAD ratifies Q-table (Q1–Q4).
- [ ] [HUMAN] Server-side API survey: confirm webhook CRUD + test-ping routes.
- [ ] A1. Scaffold `cmd/webhook.go` with `webhook` top-level cobra command and
  subcommands: list, create, get, delete, test.
- [ ] A2. Implement `webhook list` (pagination, --output).
- [ ] A3. Implement `webhook create` (secret via env/flag/stdin; print summary).
- [ ] A4. Implement `webhook get <id>` (single webhook detail).
- [ ] A5. Implement `webhook delete <id>` with `--dry-run`.
- [ ] A6. Implement `webhook test <id>` (ping endpoint; report status).

## P1

- [ ] B1. Shell completion for webhook IDs.
- [ ] B2. Handler tests: list, create, get, delete, test happy paths + error branches.

## P2

- [ ] C1. [HUMAN] Live smoke: create, test, and delete a webhook.
- [ ] C2. Spec close.
