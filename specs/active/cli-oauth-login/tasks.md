# Tasks — cli-oauth-login

Status: IN_PROGRESS 061800ZMAY26 — Bastion (J-3) claims execution

**Companion daemon:** `go-cli-oauth-provider` is **DONE** (`citadel/specs/done/go-cli-oauth-provider/`, 060608ZMAY26); P0+P1 shipped (authorize/token, handoff, consent). CLI work is no longer blocked on that spec.

## P0

- [x] [HUMAN] NOMAD ratifies Q-table (Q2-Q5).
- [x] A1. Rewrite `runLogin` in `cmd/auth.go` to POST against Citadel `/api/oauth/authorize` + `/api/oauth/token` with PKCE. Hardcoded `client_id=citadel-cli`. Loopback `http://127.0.0.1:N/callback`.
- [x] A2. Post-auth agent-token bootstrap: find-or-create `citadel-cli@<hostname>` via `/api/agents`, then `/api/agents/{id}/rotate-token`. Persist the agent token; drop the JWT.

## P1

- [x] B1. Refresh handling: on 401 from any verb, attempt one rotate-token round trip. If still 401, surface friendly "session expired" error.
- [x] B2. `clicfg.Config` extension: add `AgentID`, `AgentName` fields alongside `AccessToken`. Migration: a config with only `access_token` (JWT, no `agent_id`) is upgraded eagerly on next CLI launch (find-or-create + rotate-token), not deferred to first 401.
- [ ] B3. `auth status` rewrite to show agent name + ID + expiry first; user UUID retained as cross-reference.
- [ ] B4. `auth set-token` doc string update: explicitly call out as the headless / CI / SSH-only bypass.
- [ ] B5. Tests: rewrite handler-test fixtures for the new auth flow; add a `TestRunLogin_FlowSmoke` that pumps a fake Citadel server (httptest) end-to-end.
- [ ] B6. Drop `EXPERIMENTAL` warning from `loginCmd --help` after live verify.

## P2

- [ ] C1. Live end-to-end test (`CITADEL_TEST_OAUTH_FULL=1`) against a real Citadel test instance + real browser via Playwright (or a manual operator runbook entry).
- [ ] C2. README + HUMANS.md updates: replace the "EXPERIMENTAL: prefer set-token" callout with the new canonical `auth login` instructions.
- [ ] C3. [HUMAN] Production smoke: `citadel-cli auth login` against `https://api.src.land`, confirm token persists across CLI restarts and a subsequent verb succeeds.
- [ ] C4. Spec close.
