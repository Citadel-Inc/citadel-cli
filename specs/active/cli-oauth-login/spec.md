# Spec — cli-oauth-login

| | |
|---|---|
| Status | DRAFT 075800ZMAY26 |
| Authored | 075800ZMAY26 |
| Owner | Bastion (J-3) |
| Carry-forward from | `citadel-cli@4996580` (2026-05-05) flagged `loginCmd` as EXPERIMENTAL pending a productised auth flow. Server-side counterpart: `citadel/specs/done/go-cli-oauth-provider/` (DONE 060608ZMAY26). |

## Why

`citadel-cli auth login` currently sends `client_id=""` to Supabase Auth and is broken-by-design. Today's working bootstrap path is `auth set-token` (paste a JWT minted out-of-band) — fine for operators / CI, hostile for human first-run.

The companion daemon-side spec (`go-cli-oauth-provider`) makes Citadel its own OAuth 2.1 provider that brokers Supabase under the hood. This spec covers the CLI half: rewrite `runLogin` against Citadel's new `/api/oauth/authorize` + `/api/oauth/token`, then immediately mint a long-lived agent token so the durable credential is server-issued and revocable from the web UI.

End-to-end success: `citadel-cli auth login`, browser flow, return to terminal. CLI is now authenticated for subsequent verbs. Re-running on the same host invalidates the previous host-token (atomic rotate). User can revoke from web UI.

## In scope

- **`runLogin` rewrite**: point at `https://api.src.land/api/oauth/authorize` with PKCE. `client_id` is the CLI-side constant `citadel-cli`. Loopback redirect `http://127.0.0.1:N/callback` with N chosen at runtime. Token exchange against `https://api.src.land/api/oauth/token`.
- **Post-auth agent-token bootstrap**: after the JWT lands, call `/api/agents` to find-or-create an agent named `citadel-cli@<hostname>` (host-scoped so revocation from the web UI is per-machine). Then call `/api/agents/{id}/rotate-token` (atomic; `citadel@9af1596c`) to mint a long-lived agent token. Persist the agent token. Discard the JWT.
- **Refresh handling**: on a 401 response from any verb, attempt a single rotate-token round trip; if that also 401s, surface a friendly "session expired — run `auth login` again" error.
- **Status semantics**: `auth status` should show the bound agent ID + agent name (instead of, or in addition to, the user UUID). The user UUID is no longer the durable identity from the CLI's point of view.
- **Drop EXPERIMENTAL warning** from `loginCmd --help` once the path is wired end-to-end against a real server.
- **`auth set-token` retained** as the headless / CI bypass. Documented as the intended path for non-interactive bootstraps.
- **Migration of existing configs**: a config file with only `access_token` (a JWT) and no agent token continues to work for one expiry cycle. On the next CLI launch after upgrade, the CLI eagerly performs find-or-create + `rotate-token` and rewrites config to the agent-token shape (no need to wait for a 401).

## Out of scope

- **Refresh-token grant on the CLI side** at v1. CLI relies on agent-token rotation instead. The server's `grant_type=refresh_token` exists but the CLI does not use it (durable credential is the agent token, not the JWT).
- **Multiple-account / profile support.** Single account per `~/.config/citadel/config.toml` at v1. (`citadel-cli auth use <profile>` is a v2 idea.)
- **Custom `client_id` per CLI deploy.** Hardcoded `citadel-cli` constant; aligned with the companion server spec Q6.
- **Device-code grant** (RFC 8628). Loopback browser only at v1; deferred to a follow-on spec if SSH-only environments report the loopback flow as unworkable.

## Decision log

| Q | Proposal | Status |
|---|----------|--------|
| Q1 | Store the Supabase JWT or the post-mint agent token? | **Ratified** — agent token only; JWT discarded after mint. (User decision 2026-05-05.) |
| Q2 | Agent-name format: `citadel-cli@<hostname>` vs. `citadel-cli-<random>`? | **Ratified 061430ZMAY26** — `citadel-cli@<hostname>` for per-machine revocation legibility (NOMAD). |
| Q3 | What does `auth status` show — bound agent or bound user UUID? | **Ratified 061430ZMAY26** — Agent name + ID + expiry first; user UUID retained below for cross-reference (NOMAD). |
| Q4 | Migration: re-prompt on first 401, or eagerly mint on next launch after upgrade? | **Ratified 061430ZMAY26** — Eager: on next CLI launch after upgrade, JWT-only configs mint the agent token before ordinary API traffic (NOMAD). |
| Q5 | `runLogin` listener port range — fully ephemeral vs. fixed range (e.g., 53682–53700)? | **Ratified 061430ZMAY26** — Fully ephemeral kernel-assigned port; OAuth 2.1 allows it (NOMAD). |

## Acceptance

- A1. `runLogin` POSTs to Citadel's authorize/token endpoints, not Supabase directly.
- A2. After token exchange the CLI calls `/api/agents` (find-or-create) + `/api/agents/{id}/rotate-token` and stores the resulting agent token in `~/.config/citadel/config.toml`.
- A3. `~/.config/citadel/config.toml` no longer contains a Supabase JWT after a successful login (only the agent token + bound agent metadata).
- A4. On a 401 response, the CLI attempts one rotate-token; if that 401s, surfaces a "session expired" error pointing at `auth login`.
- A5. `auth status` shows agent name + ID + expiry alongside the user UUID.
- A6. Live end-to-end test: `citadel-cli auth login` against a real Citadel test instance opens the browser, completes the flow, and the next verb (`agent list`) succeeds without re-prompt.
- A7. `auth set-token` continues to work for headless bootstraps; new doc string clarifies the choice.
- A8. EXPERIMENTAL warning removed from `loginCmd --help`.
- A9. Q-table ratified.
