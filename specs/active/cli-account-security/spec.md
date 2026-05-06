# Spec — cli-account-security

| | |
|---|---|
| Status | IN_PROGRESS 060455ZMAY26 — Bastion (J-3) claims execution |
| Authored | 050506ZMAY26 |
| Owner | Bastion (J-3) |
| Carry-forward from | Terminal parity for account security settings (passkeys, sessions/devices, MFA recovery) exposed by Citadel auth routes but absent from `citadel-cli`. |

## Daemon HTTP contract (baseline from `cmd/citadel/main.go`)

All routes require JWT unless noted; many mutate flows wrap **`recentMFA`** (step-up) — expect **`412`** / structured **`mfa_required`**-class errors (exact mapping — survey `httputil` + auth handlers during P0).

### Passkeys (`gatePasskey` — same gate bundle)

| Method | Path |
|--------|------|
| POST | `/api/account/passkey/begin-enrol` |
| POST | `/api/account/passkey/finish-enrol` |
| POST | `/api/account/passkey/begin-auth` |
| POST | `/api/account/passkey/finish-auth` |
| GET | `/api/account/passkeys` |
| DELETE | `/api/account/passkeys/{id}` |
| PATCH | `/api/account/passkeys/{id}` |

**WebAuthn reality:** enrol/finish flows exchange crypto payloads meant for **browser WebAuthn APIs** — likely **not implementable** in pure CLI (Q1 ratifies CLI scope).

### Devices (`devicePrivate` middleware)

| Method | Path | MFA step-up |
|--------|------|---------------|
| POST | `/api/auth/devices/register` | no |
| GET | `/api/auth/devices` | no |
| DELETE | `/api/auth/devices/{id}` | **yes** (`recentMFA`) |

### MFA recovery (`mfaRoutes`; some with `recentMFA`)

| Method | Path | Notes |
|--------|------|-------|
| POST | `/api/auth/mfa/recovery-codes` | step-up |
| GET | `/api/auth/mfa/recovery-codes` | list / regenerate policy — **survey** |
| POST | `/api/auth/mfa/redeem-recovery` | redeem |
| POST | `/api/auth/mfa/recent-verify` | step-up ping |

## In scope (phased — default CLI-first subset)

**Phase A (target v1)**

| Feature | CLI shape | Server paths |
|---------|-----------|--------------|
| Passkeys | `account passkey list`, `account passkey delete <id>`, `account passkey rename …` if PATCH maps to label — **RECON** | GET/DELETE/PATCH |
| Devices | `account device list`, `account device revoke <id>` | GET + DELETE |

**Phase B (secrets — high caution)**

| Feature | Risk | Gate |
|---------|------|------|
| Recovery codes display/regen | prints **high-entropy secrets** | TTY-only + **`--force`** for scripts; never log |

**Phase C (defer / unlikely)**

- WebAuthn **begin/finish** enrol — browser-only unless future platform helper exists.

## Out of scope

- **Replacing web settings hub entirely**
- **Password change**, **email change** — different endpoints (`/api/auth/email-change` etc.) — separate spec if needed.

## Decision log

| Q | Proposal | Status |
|---|----------|--------|
| Q1 | Ship Phase A only; defer WebAuthn enrol CLIs? | **Ratified 051430ZMAY26** — Phase A only for v1 (passkey list/delete/rename via PATCH; device list/revoke); defer WebAuthn begin/finish enrol CLIs until a browser-less enrol path exists. |
| Q2 | MFA recovery output rules | **Ratified 051430ZMAY26** — MFA recovery output (Phase B): stderr banner + interactive confirm; honour **`NO_COLOR`**. |
| Q3 | Device delete step-up: interactive WebAuthn impossible — document “run from logged-in browser session first” vs proxy cookie | **Ratified 051430ZMAY26** — Document that callers complete MFA step-up in a logged-in browser session before CLI device revoke; WebAuthn cannot be driven interactively in the terminal. |

## Acceptance

- A1. Phase A verbs implemented **OR** Q1 narrows scope further with explicit list.
- A2. No secrets in **`--debug-http`** logs (verify redaction).
- A3. Structured MFA/step-up errors surface via **`cli-error-format`** conventions.
- A4. Q-table ratified (incl. Q3 disposition).
- A5. Security section in `docs/cli.md`.
