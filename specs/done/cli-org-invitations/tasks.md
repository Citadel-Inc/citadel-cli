# Tasks — cli-org-invitations

Status: DONE 052317ZMAY26 — Delivered `citadel-cli org invitation` (pending, list, create, revoke, accept) with output formats, TTY email prompt, token-file accept, httptest matrix for 409/404/400 paths, docs/cli.md, plan RECON appendix, and opt-in live pending test behind CITADEL_TEST_ORG_INVITATIONS_LIVE=1.

## P0

- [x] [HUMAN] NOMAD ratifies Q-table (Q1–Q3).
- [x] A1. RECON appendix: invitation row JSON + permission atom examples from server docs/tests.
- [x] A2. Implement pending/list/create/revoke/accept + httptest suite.
- [x] A3. Wire command tree + `root.go` per Q1 resolution.

## P1

- [x] B1. `docs/cli.md` — org invitations section + token hygiene.

## P2

- [x] C1. Live test `CITADEL_TEST_ORG_INVITATIONS_LIVE=1`.
- [x] C2. Spec close.
