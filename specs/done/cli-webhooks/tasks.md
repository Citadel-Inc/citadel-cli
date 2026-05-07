# Tasks — cli-webhooks

Status: DONE 071726ZMAY26 — Shipped nested `repo webhook` and `namespace webhook` list/create/get/delete commands against Citadel's namespace-scoped webhook API, including server-generated secret handling, webhook ID completion, handler coverage, docs, and a backend follow-up issue for missing test-ping support (`citadel#8`).

## P0

- [x] [HUMAN] NOMAD ratifies Q-table (Q1–Q4) for nested repo/namespace commands.
- [x] [HUMAN] Server-side API survey: confirm webhook CRUD/deliveries routes and whether a test-ping route exists.
- [x] A1. Scaffold `cmd/webhook.go` under `repo webhook` and `namespace webhook`
- [x] A2. Implement webhook list verbs with pagination and shared `--output` support.
- [x] A3. Implement webhook create verbs with server-generated secret handling and
- [x] A4. Implement webhook get by listing and filtering when no dedicated GET route exists.
- [x] A5. Implement webhook delete with `--dry-run`.
- [x] A6. Track missing server-side `webhook test` support in Citadel and do not fake it client-side.

## P1

- [x] B1. Shell completion for repo/namespace webhook IDs.
- [x] B2. Handler tests: list, create, get, delete happy paths + error branches.

## P2

- [ ] C1. [HUMAN] Live smoke: create and delete a webhook; add test-ping once the server exposes it.
- [ ] C2. Spec close.
