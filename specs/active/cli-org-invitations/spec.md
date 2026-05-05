# Spec — cli-org-invitations

| | |
|---|---|
| Status | DRAFT 050506ZMAY26 |
| Authored | 050506ZMAY26 |
| Owner | Bastion (J-3) |
| Carry-forward from | Phase 0 closed-alpha + org ops: `orgsmembersapi` exposes invitations (`list`, `create`, `revoke`, `pending` inbox, `accept` by token) but `citadel-cli` only has `namespace members` + transfer — no invitation flow. |

## Why

Org owners invite collaborators via email/token links. Operators need to **list pending invites for the caller**, **manage invites on an org**, and **accept** an invite from the terminal (CI or headless) when a token is provided.

## In scope

**Suggested command tree:** `citadel-cli org invitation …` (or `namespace invitation` — Q-table) to avoid overloading `namespace` which already has many subcommands.

| Verb | HTTP |
|------|------|
| `org invitation pending` | `GET /api/invitations/pending` |
| `org invitation list <org-slug>` | `GET /api/orgs/{slug}/invitations` |
| `org invitation create <org-slug> …` | `POST /api/orgs/{slug}/invitations` |
| `org invitation revoke <org-slug> <id>` | `DELETE /api/orgs/{slug}/invitations/{id}` |
| `org invitation accept <token>` | `POST /api/invitations/{token}/accept` |

**Payload fields** for create: match server body (email, permission atoms, etc. — exact JSON from RECON in plan).

**Cross-cutting**

- **cli-output-formats** on list/pending.
- **cli-pagination** if list endpoints paginate.
- RBAC: write paths need `members:write` on org; document 403 copy.

## Out of scope

- **Email template editing** — web/operator.
- **Public org directory** — unrelated.

## Decision log

| Q | Proposal | Status |
|---|----------|--------|
| Q1 | `org invitation` vs `namespace invitation`? | **Open** — `org invitation` (shorter; still pass org slug). |
| Q2 | `create` interactive prompts for email when omitted? | **Open** — yes on TTY; else require flags. |

## Acceptance

- A1. All five HTTP paths covered with tests.
- A2. `make verify` passes.
- A3. Docs + Q-table.
- A4. Optional `CITADEL_TEST_ORG_INVITATIONS_LIVE=1`.
