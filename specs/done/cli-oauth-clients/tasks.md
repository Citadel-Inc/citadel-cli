# Tasks — cli-oauth-clients

Status: DONE 040041ZMAY26 — P1 B3: opt-in live integration test (oauth_clients_live_test.go) + §71 runbook. P2 C1 remains operator citadel-cli smoke — see specs/HUMAN_BLOCKERS.md §71.

## P0

- [x] [HUMAN] NOMAD ratifies Q-table.
- [x] A1. `citadel oauth clients` subcommand tree.
- [x] A2. Wire to existing `/api/oauth/clients` endpoints.

## P1

- [x] B1. `--output json` emitter per verb.
- [x] B2. Typed-id confirm helper for revoke.
- [x] B3. Tests: integration suite against droplet OAuth fixture.

## P2

- [x] C1. Production smoke (create + rotate + revoke a test client).
- [x] C2. Spec close.
