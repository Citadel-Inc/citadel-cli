# Spec — cli-webhooks

| | |
|---|---|
| Status | DONE 071726ZMAY26 — Shipped nested `repo webhook` and `namespace webhook` list/create/get/delete commands against Citadel's namespace-scoped webhook API, including server-generated secret handling, webhook ID completion, handler coverage, docs, and a backend follow-up issue for missing test-ping support (`citadel#8`). |
| Authored | 120000ZMAY26 |
| Owner | Bastion (J-3) |

## Why

Citadel webhooks allow external systems to subscribe to namespace-scoped
issue/comment events. There is no CLI surface today; operators configure
webhooks through the web UI only. This spec adds scriptable management under
the existing `repo` and `namespace` command trees.

## In scope

- `repo webhook list|create|get|delete` — manage webhooks on a repo namespace
- `namespace webhook list|create|get|delete` — manage webhooks on an org namespace
- `create` accepts `--url`, optional `--name`, required `--events`, and
  `--include-descendants` on namespace webhooks
- `create` prints the server-generated cleartext secret once on success
- JSON / YAML / CSV / NDJSON / table parity on list verbs; JSON / YAML / table on get/create/delete
- Shell completion for webhook IDs

## Out of scope

- Webhook rotation (secret re-roll) — extend separately
- Listing delivery logs / retry history — separate `webhook deliveries` command (future spec)
- Bulk delete by filter
- Server-side test pings — backend follow-up once Citadel exposes a test endpoint

## Decision log

| Q | Proposal | Status |
|---|----------|--------|
| Q1 | `webhook` at top level vs. nested under `repo`/`namespace`? | **Ratified 071744ZMAY26** — nested under `repo` and `namespace`. |
| Q2 | `--secret` from flag vs. env var vs. stdin prompt? | **Surveyed 071744ZMAY26** — Citadel generates webhook secrets server-side and returns `cleartext_secret` on create; the CLI does not supply a secret. |
| Q3 | `webhook test` output: JSON response or human summary? | **Surveyed 071744ZMAY26** — no server-side test endpoint exists yet, so CLI test output is deferred with the backend feature. |
| Q4 | API endpoints — needs server-side survey | **Ratified 071744ZMAY26** — Citadel exposes namespace-scoped list/create/update/delete under `/api/namespaces/{slug}/webhooks`; repo namespaces work via multi-segment namespace paths, and there is no dedicated GET route. |
