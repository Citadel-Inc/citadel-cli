# Tasks — cli-webhook-test

Status: PARKED 091233ZMAY26 — REJECTED — out of scope per NCA order 091232ZMAY26. Webhook test-ping is an operational surface; no current CLI mandate to wrap it.

## P0

- [ ] [HUMAN] NOMAD ratifies Q-table (Q1–Q3): command placement, server route, output shape.
- [ ] A1. Survey `go-issues-webhooks` implementation in `citadel` to confirm test-ping route path, request shape, and response fields.
- [ ] A2. Add `repo webhook test <webhook-id>` command under `cmd/webhook.go`; reuse existing webhook-ID completion.
- [ ] A3. Add `namespace webhook test <webhook-id>` command; same implementation via namespace-slug dispatch.
- [ ] A4. Human-readable output: show status (success/fail), HTTP response code, and latency if returned.
- [ ] A5. `--output json|yaml` parity.
- [ ] A6. Error handling: webhook not found (404), endpoint unreachable, auth failure (403).

## P1

- [ ] B1. Handler tests: success path, 404 not-found, 403 unauthorized.
- [ ] B2. Update `docs/cli.md` webhook section to document `test` subcommand; remove the "does not yet wrap" note.

## P2

- [ ] C1. [HUMAN] Live smoke: trigger test ping against a real webhook endpoint; confirm delivery in webhook provider dashboard.
- [ ] C2. Spec close.
