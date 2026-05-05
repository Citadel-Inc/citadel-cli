# Spec — cli-account-security

| | |
|---|---|
| Status | DRAFT 050506ZMAY26 |
| Authored | 050506ZMAY26 |
| Owner | Bastion (J-3) |
| Carry-forward from | Power users need terminal access to **passkeys**, **registered devices**, and **MFA recovery** flows without the SPA. Citadel exposes JWT-gated routes under `/api/account/passkeys*`, `/api/auth/devices*`, `/api/auth/mfa/*` (see `cmd/citadel/main.go`). |

## Why

Security-sensitive account operations are often scripted or performed in locked-down environments where opening a browser is undesirable. Thin CLI wrappers improve parity with the web app and reduce support friction.

## In scope (phased)

**Phase A — read/list + low-risk mutations**

| Verb area | Example commands | HTTP (indicative) |
|-----------|-------------------|-------------------|
| Passkeys | `account passkey list`, `account passkey delete <id>` | `GET/DELETE /api/account/passkeys…` |
| Devices | `account device list`, `account device revoke <id>` | `GET/DELETE /api/auth/devices…` |

**Phase B — MFA recovery material (stdout-sensitive)**

| Verb area | Example | Notes |
|-----------|---------|-------|
| Recovery codes | `account mfa recovery-codes` | May **print secrets once** — mirror web UX warnings; TTY-only or `--force` gate. |
| Redeem / verify | defer or minimal flags | Survey CSRF/step-up requirements in RECON. |

**Phase C — passkey enrol crypto flows**

| Area | Notes |
|------|-------|
| `begin-enrol` / `finish-enrol` | Likely requires WebAuthn client — **may be impossible** in pure CLI; Q-table may ratify **web-only enrol** + CLI **list/delete only**. |

## Out of scope

- **Replacing Supabase Auth dashboard entirely.**
- **Password change** — separate endpoints if any; defer unless trivial GET/POST survey says otherwise.

## Decision log

| Q | Proposal | Status |
|---|----------|--------|
| Q1 | Phase A only at v1 if WebAuthn blocks enrol in CLI? | **Open** — ship list/delete for passkeys + devices first. |
| Q2 | Recovery codes: never print to non-TTY without `--force`? | **Open** — yes. |

## Acceptance

- A1. Phase A verbs implemented + tests OR Q-table explicitly scopes Phase B/C follow-ons.
- A2. Secrets never logged in verbose HTTP debug (respect existing redaction).
- A3. `make verify` passes.
- A4. Q-table ratified.
- A5. Docs warn users about recovery-code sensitivity.
