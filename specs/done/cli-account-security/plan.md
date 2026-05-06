# Plan — cli-account-security

## ORIENT

- **Routing:** `cmd/citadel/main.go` comments near **passkey**, **device**, **mfa** mounts (lines ~344–374 region — verify on merge).
- **Handlers:** likely under `internal/api/accountapi` + `internal/api/authapi` — P0 A1 indexes concrete Go types for JSON bodies.

## RECON

1. **Passkey list/delete/patch** — read handler funcs for exact JSON + error codes.
2. **Devices list/delete** — confirm DELETE requires recent MFA — CLI must print actionable message referencing web step-up when server rejects.
3. **MFA recovery GET/POST** — determine whether GET returns one-time printable codes — if yes, gate stdout carefully.

## Implementation sketch

- **`cmd/account.go`** with subcommand groups **`passkey`**, **`device`**, **`mfa`** (hidden until shipped).
- Prefer **thin** GET/DELETE wrappers before touching MFA recovery.

## Risks

- **Step-up MFA** on device delete may block **fully non-interactive CI** — document exemption path (skip test) vs operator expectation.

## Appendix: HTTP JSON contracts (cli ↔ daemon)

All paths below are relative to the configured API host (same style as existing verbs: `/account/ssh-keys`, `/oauth/clients`, …).

| Verb | Method | Path | Response / body |
|------|--------|------|-------------------|
| Passkey list | GET | `/account/passkeys` | JSON object **`passkeys`**: array of `{ id, name, created_at }`, or a bare JSON array of those objects. |
| Passkey delete | DELETE | `/account/passkeys/{id}` | Prefer **204 No Content**; **404** when missing. |
| Passkey rename | PATCH | `/account/passkeys/{id}` | Request JSON **`{ "name": "<display name>" }`**. Prefer **204** without body (CLI does not require a response envelope). |
| Device list | GET | `/auth/devices` | JSON object **`devices`**: array of `{ id, name, user_agent?, last_seen_at?, created_at? }`, or a bare array. |
| Device revoke | DELETE | `/auth/devices/{id}` | Prefer **204**. **412 Precondition Failed** when recent MFA / step-up is required (surfaced as **`cli-error-format`** `mfa_required` via global error mapping). |

**Errors:** **401** auth, **403** forbidden, **404** missing resource; device DELETE may return **412** for MFA step-up per daemon routing (`recentMFA`).
