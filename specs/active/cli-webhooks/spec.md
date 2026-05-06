# Spec — cli-webhooks

| | |
|---|---|
| Status | DRAFT |
| Authored | 120000ZMAY26 |
| Owner | unassigned |

## Why

Citadel webhooks allow external systems to subscribe to repo/namespace
events (push, PR open, deployment, etc.). There is no CLI surface today;
operators configure webhooks through the web UI only. This spec adds
`citadel-cli webhook` for scriptable management of webhook registrations.

## In scope

- `webhook list [--repo <path>] [--namespace <ns>]` — list registered webhooks
- `webhook create --repo | --namespace <target> --url <url> [--secret <s>] [--events <e,...>]` — register a new webhook
- `webhook get <id>` — show a single webhook's configuration
- `webhook delete <id>` — remove a webhook; support `--dry-run`
- `webhook test <id>` — send a test ping; report HTTP response status
- JSON / YAML / table `--output` parity
- Shell completion for webhook IDs

## Out of scope

- Webhook rotation (secret re-roll) — extend separately
- Listing delivery logs / retry history — separate `webhook deliveries` command (future spec)
- Bulk delete by filter

## Decision log

| Q | Proposal | Status |
|---|----------|--------|
| Q1 | `webhook` at top level vs. nested under `repo`/`namespace`? | OPEN |
| Q2 | `--secret` from flag vs. env var vs. stdin prompt? | OPEN — recommend env var `CITADEL_WEBHOOK_SECRET` + flag + stdin fallback |
| Q3 | `webhook test` output: JSON response or human summary? | OPEN |
| Q4 | API endpoints — needs server-side survey | OPEN |
