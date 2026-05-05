# Plan — cli-account-security

## ORIENT

- Routes registered in `citadel/cmd/citadel/main.go` — search for `passkey`, `device`, `mfa`, `gatePasskey`, `recentMFA`.
- Many mutations require **step-up / recent MFA** (`recentMFA` middleware) — CLI must surface `403` / structured errors clearly.

## RECON

- For each handler: request method, path, JSON body, and whether WebAuthn/browser is required.
- Decide v1 cut: **list/delete passkeys** and **list/delete devices** are usually feasible; enrol flows may remain web-only.

## Implementation sketch

- Parent: `citadel-cli account` with subgroups `passkey`, `device`, `mfa` (hidden until commands exist).
- Reuse `newAPIClient`; no new token storage.

## Risks

- **WebAuthn in terminal** — likely unsupported for enrol; avoid half-baked flows that leak partial state.
