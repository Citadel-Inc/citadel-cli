# Spec — cli-webhook-test

| | |
|---|---|
| Status | PARKED 091233ZMAY26 — REJECTED — out of scope per NCA order 091232ZMAY26. Webhook test-ping is an operational surface; no current CLI mandate to wrap it. |
| Authored | 091227ZMAY26 |
| Owner | Bastion (J-3) |
| Carry-forward from | `cli-webhooks`: shipped CRUD; test-ping deferred pending server endpoint (`citadel#8`) |

## Why

`cli-webhooks` shipped list/create/get/delete verbs but explicitly deferred the `webhook test` command: no server-side test-ping endpoint existed at that time. The server has since added the endpoint (closed as part of `go-issues-webhooks`). `docs/cli.md` calls this out: "Citadel now exposes a server-side webhook test/ping endpoint; `citadel-cli` does not yet wrap it as a `webhook test` command." This spec closes that gap.

## In scope

- `citadel-cli repo webhook test -R <slug> <webhook-id>` — trigger a server-side test ping for a repo namespace webhook
- `citadel-cli namespace webhook test <slug> <webhook-id>` — same for an org namespace webhook
- Human output: show test delivery status (success/fail), HTTP response code, latency if available
- JSON / YAML output parity via `--output`
- Shell completion for webhook IDs (reuse existing completion from `cli-webhooks`)
- Error paths: webhook not found (404), endpoint unreachable (timeout), auth failure (403)

## Out of scope

- Webhook delivery log / retry history — separate `webhook deliveries` spec
- Replay of a specific past delivery — future spec
- Webhook rotation (secret re-roll) — separate spec

## Decision log

| Q | Proposal | Status |
|---|----------|--------|
| Q1 | Command placement: `repo webhook test` / `namespace webhook test` matching existing CRUD shape | Ratified 091227ZMAY26 |
| Q2 | Server endpoint route — needs survey of `go-issues-webhooks` implementation | Open |
| Q3 | Output on success: summary table vs raw delivery response? | Open |
