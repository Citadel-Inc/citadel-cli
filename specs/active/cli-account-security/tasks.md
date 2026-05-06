# Tasks — cli-account-security

Status: IN_PROGRESS 060455ZMAY26 — Bastion (J-3) claims execution

## P0

- [x] [HUMAN] NOMAD ratifies Q-table (Q1–Q3) + confirms Phase A scope.
- [x] A1. RECON appendix: per-endpoint JSON request/response/error codes (copy from server handlers).
- [x] A2. Implement Phase A passkey + device list/delete (and PATCH if in scope) + httptest suite.
- [x] A3. Wire `account` command subtree + `root.go`.

## P1

- [ ] B1. MFA recovery verbs **if** Q-table unlocks + safety gates implemented.
- [x] B2. `docs/cli.md` security section.

## P2

- [ ] C1. Env-gated live tests per verb family.
- [ ] C2. Spec close.
