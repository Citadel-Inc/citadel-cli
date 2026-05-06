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
- [x] B3. `auth status` rewrite to show agent name + ID + expiry first; user UUID retained as cross-reference.
- [x] B4. `auth set-token` doc string update: explicitly call out as the headless / CI / SSH-only bypass.
- [x] B5. Tests: rewrite handler-test fixtures for the new auth flow; add a `TestRunLogin_FlowSmoke` that pumps a fake Citadel server (httptest) end-to-end.
- [x] B6. Drop `EXPERIMENTAL` warning from `loginCmd --help` after live verify.

## P2

- [x] C1. Automation-capable live end-to-end test (`CITADEL_TEST_OAUTH_FULL=1`) against a real Citadel instance + real browser via Playwright. Supports either a signed-in Playwright storage-state file or a Citadel refresh-token bootstrap path that mints a fresh JWT, bridges the OAuth cookie, and auto-approves consent.
- [x] C2. README + HUMANS.md updates: replace the "EXPERIMENTAL: prefer set-token" callout with the new canonical `auth login` instructions.
- [ ] C3. Production smoke: `citadel-cli auth login` against the live production host (`https://mcp.src.land` as of 2026-05-06), confirm token persists across CLI restarts and a subsequent verb succeeds. Current blocker in this session: only a stale legacy Supabase token pair was available locally, and the copied Chromium profile did not contain an active src.land session to reuse.
- [ ] C4. Spec close.
