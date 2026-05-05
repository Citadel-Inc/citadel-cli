# Spec — cli-org-invitations

| | |
|---|---|
| Status | DRAFT 050506ZMAY26 |
| Authored | 050506ZMAY26 |
| Owner | Bastion (J-3) |
| Carry-forward from | Phase 0 closed-alpha org ops: `orgsmembersapi` implements invitations + pending inbox + accept-by-token; `citadel-cli` has **members** + **transfer** but no invitation lifecycle. |

## Daemon HTTP contract (`orgsmembersapi`)

| Route | Handler | RBAC / notes |
|-------|---------|----------------|
| `GET /api/invitations/pending` | `HandlePendingInvitations` | Caller-scoped pending invites for signed-in user’s email(s). |
| `GET /api/orgs/{slug}/invitations` | `HandleListInvitations` | Org **members:read** (see `gateOrg` pattern in handler). |
| `POST /api/orgs/{slug}/invitations` | `HandleCreateInvitation` | Org **members:write**. Body **`createInvitationRequest`**: `email`, `slug`, `permissions` ([]string). Email **or** discoverable **slug** (public user namespace) required — server resolves slug→email for public users only (`user_not_found` if no match). Permissions validated via `orgs.StringsToPermissions` + `ValidatePermissions`. Duplicate pending invite → **`409 already_pending`**. |
| `DELETE /api/orgs/{slug}/invitations/{id}` | `HandleRevokeInvitation` | members:write |
| `POST /api/invitations/{token}/accept` | `HandleAcceptInvitation` | JWT required; token in path identifies invite. |

**Email dispatch:** create path triggers **best-effort** email goroutine — CLI users still see success if DB insert succeeded.

## In scope

**Suggested tree:** `citadel-cli org invitation …` (Q-table may rename — see below).

| CLI | HTTP |
|-----|------|
| `org invitation pending` | `GET /api/invitations/pending` |
| `org invitation list <org-slug>` | `GET /api/orgs/{slug}/invitations` |
| `org invitation create <org-slug>` | `POST …` — flags: `--email`, `--slug` (target user slug), `--permission` repeatable or CSV (must match server atom strings). |
| `org invitation revoke <org-slug> <invite-id>` | `DELETE …` |
| `org invitation accept <token>` | `POST /api/invitations/{token}/accept` — **warn** token appears in shell history; offer **`--token-file`**. |

**Cross-cutting**

- **cli-output-formats** on list/pending.
- **Pagination** if server returns next pages — survey list handlers (append to plan if cursor exists).

## Out of scope

- **Resend invite email** — web / operator (unless REST exists — not in baseline RECON).
- **Editing invite permissions** — create-only + revoke.

## Decision log

| Q | Proposal | Status |
|---|----------|--------|
| Q1 | `org invitation` vs `namespace invitation`? | **Open** — `org invitation` (org-scoped language). |
| Q2 | `create` interactive email prompt on TTY? | **Open** — yes when flags omitted. |
| Q3 | Permissions flag: repeated `--permission` vs JSON file? | **Open** — repeated flag v1. |

## Acceptance

- A1. All five routes + tests including **`409 already_pending`**, **`user_not_found`**, **`invalid_permission`** paths stubbed.
- A2. Security note in `--help` for **accept** token handling.
- A3. Q-table ratified.
- A4. Optional `CITADEL_TEST_ORG_INVITATIONS_LIVE=1`.
