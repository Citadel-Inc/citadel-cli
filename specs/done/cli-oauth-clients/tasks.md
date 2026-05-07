# Tasks — cli-oauth-clients

Status: DONE 071629ZMAY26 — OAuth client CLI support remains shipped: the opt-in live integration test and runbook landed, while any additional operator citadel-cli smoke stays as an out-of-band follow-up instead of a HUMAN_BLOCKERS entry.

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
