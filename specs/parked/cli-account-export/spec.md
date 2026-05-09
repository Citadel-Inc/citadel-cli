# Spec — cli-account-export

| | |
|---|---|
| Status | PARKED 091233ZMAY26 — REJECTED — out of scope per NCA order 091232ZMAY26. GDPR data export is a web account-settings surface; CLI export path not in current phase mandate. |
| Authored | 091227ZMAY26 |
| Owner | Bastion (J-3) |
| Carry-forward from | `fe-account-export`: GDPR export backend + web panel DONE; CLI surface absent |

## Why

Phase 1 freemium launch (PoL XV.3.1) mandates GDPR data portability. The server ships `POST /api/account/export-request`, `GET /api/account/export-status/{id}`, and `GET /api/account/export-history` (via `fe-account-export`). Users can trigger and download their data bundle via the web Account → Privacy panel. No CLI path exists. Developers need a scriptable, CI-friendly way to request and retrieve their data bundle without a browser.

## In scope

- `citadel-cli account export` — request a new export bundle; prints the `request_id`; with `--wait`, polls until ready and prints the signed download URL
- `citadel-cli account export status <request-id>` — check the status (queued/building/ready/failed/expired) of an existing request; prints the download URL when ready
- `citadel-cli account export history` — list past export requests (last 5 per server); shows status, requested-at, and download link for ready bundles
- `--output json|yaml|table` on `status` and `history`
- Error paths: export already in progress (409), expired bundle (410), auth failure

## Out of scope

- Automatic download / streaming of the ZIP archive — print the signed URL; downloading is the user's responsibility
- Operator-initiated export (`go-export-operator`) — operator-admin surface, permanently out of CLI scope
- PGP-encrypted bundles — `fe-account-export-pgp` follow-on

## Acceptance

A1. `account export` creates a queued request and prints `request_id`.
A2. `account export --wait` polls until `ready` and prints the signed download URL.
A3. `account export status <id>` reflects current server state; includes download URL when ready.
A4. `account export history` lists up to 5 past requests with status and download URL.
A5. Duplicate submit while in-progress surfaces the 409 error message clearly.

## Decision log

| Q | Proposal | Status |
|---|----------|--------|
| Q1 | Subcommand root: `account export` vs `account data-export` vs `export`? | Ratified 091227ZMAY26 — `account export` matches `gh account` pattern and `account` command group already used by `cli-account-security`. |
| Q2 | `--wait` polling interval and timeout? | Open |
| Q3 | Progress indicator during `--wait` polling (spinner vs periodic "still building…")? | Open |
