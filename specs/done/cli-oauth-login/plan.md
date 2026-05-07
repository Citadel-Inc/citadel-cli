# Plan — cli-oauth-login

## Sequence

```
1. citadel-cli auth login
2. listen on 127.0.0.1:N (kernel-assigned)
3. open browser to:
     https://mcp.src.land/api/oauth/authorize
       ?client_id=citadel-cli
       &redirect_uri=http://127.0.0.1:N/callback
       &response_type=code
       &code_challenge=<S256(verifier)>
       &code_challenge_method=S256
       &state=<random>
4. wait on local listener for callback
5. on hit, parse code + state; verify state
6. POST https://mcp.src.land/api/oauth/token
     grant_type=authorization_code
     code=...
     code_verifier=<verifier>
     client_id=citadel-cli
     redirect_uri=http://127.0.0.1:N/callback
   → { access_token: <Supabase JWT>, refresh_token: <opaque> }
7. POST /api/agents { name: "citadel-cli@<hostname>" }
   → { id: <agent-uuid>, name: "..." }
8. POST /api/agents/<agent-uuid>/rotate-token
   → { cleartext_token: <opaque>, ... }
9. write to ~/.config/citadel/config.toml:
     access_token = <opaque agent token>   # NOT the JWT
     agent_id    = <agent-uuid>
     agent_name  = "citadel-cli@<hostname>"
     expires_at  = <90 days>
```

## Code shape

`cmd/auth.go` after rewrite:

- `runLogin` returns to its existing skeleton (loopback listener, generatePKCE, openBrowser, code-channel select). Three things change:
  - `buildAuthorizeURL` takes a Citadel base URL, not a Supabase URL. New constant `oauthClientID = "citadel-cli"`.
  - `exchangePKCECode` POSTs to Citadel's `/api/oauth/token` with the right form fields (no `apikey` header needed; Citadel-mediated).
  - After `exchangePKCECode` returns, call a new `bootstrapAgentToken(ctx, c, jwt)` helper that does the find-or-create + rotate dance. That helper writes the agent token directly into `clicfg.Config` (overwriting any leftover JWT).
- `pkceTokenResponse` adds `expires_in` so the post-auth window can be propagated to `cfg.ExpiresAt`. The agent token's lifetime supersedes once `bootstrapAgentToken` lands.
- `runStatus` reads the new `AgentID` / `AgentName` fields and prints them first.
- `claimsFromJWT` / `userUUIDFromClaims` / `expiryFromClaims` retained for the post-token-exchange identity extraction (we still want to display "logged in as <user>" once even though we drop the JWT).

`internal/clicfg/clicfg.go`:

- Add `AgentID uuid.UUID`, `AgentName string` fields alongside the existing `AccessToken` etc.
- Migration is silent: an old config without those fields decodes fine; the next successful `auth login` writes them. A config with `AccessToken` and no `AgentID` is treated as "JWT-mode legacy" — on the next CLI launch after upgrade, eagerly run find-or-create + `rotate-token` and rewrite config (per ratified Q4), without waiting for 401.

`internal/apiclient/client.go`:

- One-shot rotate hook. Add a `RetryOn401 func(ctx) (string, error)` field; the `do` loop calls it on a 401 and replays the request once with the new token. The hook lives in `cmd/client.go` (where it can call `/api/agents/{id}/rotate-token`); apiclient stays domain-free.

## Risks

- **Listener port collision** in restricted environments. OAuth 2.1 explicitly allows ephemeral; if a corp firewall blocks loopback (rare), `auth set-token` is the documented escape hatch.
- **Hostname leak via agent name.** Q2: `citadel-cli@<hostname>` makes per-machine revocation legible but exposes machine names in the agent list. Mitigation: optional `--agent-name <custom>` flag on `auth login` for users who care.
- **Browser session-bridge UX**: if the user is not logged into the web app, the handoff redirect chain bounces them through `/login` first. Should still work, but the UX is "login twice" the first time. Document this in README.
- **Pre-existing JWT mode**: configs from set-token won't have `AgentID`. `auth login` overwrites cleanly; eager launch-time migration covers legacy JWT-only configs.

## Estimated delta

| Component | LOC (rough) |
|-----------|-------------|
| `runLogin` rewrite (Citadel-mediated PKCE) | 60 |
| `bootstrapAgentToken` helper | 40 |
| `clicfg` field additions + migration handling | 30 |
| `apiclient` 401-retry hook | 50 |
| `runStatus` rewrite | 20 |
| Tests (handler + apiclient retry) | 100 |
| **Total** | **~300** |

Lower than the placeholder ~200 estimate in the daemon spec (Markdown drift; that estimate predates the apiclient retry-hook scope).
