# Tasks — cli-account-security

Status: DONE 060459ZMAY26 — Phase A delivered: account passkey list/rename/delete, device list/revoke, PATCH client support, httptest + opt-in live tests (CITADEL_TEST_ACCOUNT_SECURITY_LIVE), docs and CSV contracts. Phase B MFA recovery verbs intentionally deferred (P1 B1 remains open).

## P0

- [x] [HUMAN] NOMAD ratifies Q-table (Q1–Q3) + confirms Phase A scope.
- [x] A1. RECON appendix: per-endpoint JSON request/response/error codes (copy from server handlers).
- [x] A2. Implement Phase A passkey + device list/delete (and PATCH if in scope) + httptest suite.
- [x] A3. Wire `account` command subtree + `root.go`.

## P1

- [ ] B1. MFA recovery verbs **if** Q-table unlocks + safety gates implemented.
- [x] B2. `docs/cli.md` security section.

## P2

- [x] C1. Env-gated live tests per verb family.
- [x] C2. Spec close.
