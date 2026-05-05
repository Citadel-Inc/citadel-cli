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
