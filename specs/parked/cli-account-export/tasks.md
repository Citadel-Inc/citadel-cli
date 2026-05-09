# Tasks — cli-account-export

Status: PARKED 091233ZMAY26 — REJECTED — out of scope per NCA order 091232ZMAY26. GDPR data export is a web account-settings surface; CLI export path not in current phase mandate.

## P0

- [ ] [HUMAN] NOMAD ratifies Q-table (Q1–Q3): command root confirmed `account export`, polling interval, progress indicator.
- [ ] A1. Scaffold `account export` subcommand in `cmd/account.go` (or new `cmd/account_export.go`).
- [ ] A2. Implement `account export` — POST export-request; print `request_id`.
- [ ] A3. Implement `--wait` flag on `account export` — poll `GET /api/account/export-status/{id}` until `ready` or `failed`; print signed URL on ready.
- [ ] A4. Implement `account export status <request-id>` — GET export-status; show state + URL when ready.
- [ ] A5. Implement `account export history` — GET export-history; show list of requests with status and download link.
- [ ] A6. `--output json|yaml|table` parity on `status` and `history`.
- [ ] A7. Error handling: 409 export-in-progress (show existing request_id), 410 expired bundle, auth failure.

## P1

- [ ] B1. Handler tests: request creation, --wait polling, status, history, 409/410 error paths.
- [ ] B2. Update `docs/cli.md` account section with `account export` subcommand documentation.

## P2

- [ ] C1. [HUMAN] Live smoke: trigger export, wait for ready, confirm download URL resolves to a valid ZIP.
- [ ] C2. Spec close.
