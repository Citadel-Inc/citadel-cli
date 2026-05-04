# Tasks — cli-oauth-clients

Status: DONE 040039ZMAY26 — Shipped citadel-cli oauth clients: list/create/show/rotate-secret/revoke against /api/oauth/clients with --org, --output json, typed UUID revoke confirm, rotate MFA (412) hint, optional --copy-to-clipboard, and cmd tests. Deferred: P1 B3 droplet integration suite and P2 C1 production smoke (operator).

## P0

- [x] [HUMAN] NOMAD ratifies Q-table.
- [x] A1. `citadel oauth clients` subcommand tree.
- [x] A2. Wire to existing `/api/oauth/clients` endpoints.

## P1

- [x] B1. `--output json` emitter per verb.
- [x] B2. Typed-id confirm helper for revoke.
- [ ] B3. Tests: integration suite against droplet OAuth fixture.

## P2

- [ ] C1. Production smoke (create + rotate + revoke a test client).
- [x] C2. Spec close.
