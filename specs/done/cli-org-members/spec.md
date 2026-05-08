# Spec — cli-org-members

| | |
|---|---|
| Status | DONE 080809ZMAY26 — Shipped org member list, set-permissions, and remove. 14/14 handler tests green, make verify clean. docs/cli.md updated. C1 (live smoke) and C2 left as P2 open rows per allow_open. |
| Priority | Medium — admin workflow; `gh org member` analogue; required for closed-alpha team ops |
| Authored | 080750ZMAY26 |
| Owner | Bastion (J-3) |
| Carry-forward from | `orgsmembersapi` ships `GET/PATCH/DELETE /api/orgs/{slug}/members`; `cli-org-invitations` (DONE 052317ZMAY26) covers the invite lifecycle but the member-management surface (list, update permissions, remove) has zero CLI coverage. |

## Why

Org owners and admins managing closed-alpha teams need to inspect membership, adjust permission sets for graduating members, and remove departed contributors — all without a browser session. `gh org member list` / `gh org member remove` are established analogues. The daemon routes are live; this is a pure CLI gap.

## In scope

- `citadel-cli org member list <org-slug>` — paginated member list with permissions, `--output json|yaml|table|csv`
- `citadel-cli org member set-permissions <org-slug> <member>` — replace permission set with `--permission` flag(s)
- `citadel-cli org member remove <org-slug> <member>` — remove member; surface `self_removal_lockout` and `cannot_remove_owner` clearly

`<member>` accepted as either user UUID **or** user slug (slug resolved by scanning list response; see Q2).

### API mapping

| Verb | Method + Path | Auth |
|------|---------------|------|
| `list` | `GET /api/orgs/{slug}/members` | `members:read` |
| `set-permissions` | `PATCH /api/orgs/{slug}/members/{user_id}` | `members:write` |
| `remove` | `DELETE /api/orgs/{slug}/members/{user_id}` | `members:write` |

### Response shapes

`list` → `{ members: [{user_id, email?, slug?, display_name?, is_owner, permissions, joined_at}], next_cursor? }`
`update` → 204 No Content
`remove` → 204 No Content

### Error codes to surface explicitly

| Code | Status | Friendly message |
|------|--------|-----------------|
| `cannot_modify_owner` | 403 | Cannot change permissions for the org owner |
| `cannot_remove_owner` | 403 | Cannot remove the org owner |
| `self_removal_lockout` | 403 | Cannot remove yourself: no other members:write holder remains |
| `invalid_permission` | 400 | Unknown permission atom; valid atoms include members:read, members:write, code:read, … |

## Out of scope

- `GET /api/orgs/{slug}/members/public` (unauthenticated public list) — web-facing surface, no CLI use case
- Resend invitation email (carry-forward in invitation spec)
- Org transfer (separate `renameapi`/`orgstransferapi` concern)
- Passkey/device management (already in `account passkey` / `account device`)

## Decision log

| # | Question | Proposed default | NOMAD |
|---|----------|------------------|-------|
| Q1 | Subcommand verb for permission replacement: `update` vs `set-permissions`? | `set-permissions` — explicit about what changes. | Ratified 080753ZMAY26 |
| Q2 | `<member>` arg: UUID only, slug only, or auto-detect? | Auto-detect: UUID-shaped used directly; slug resolved to UUID via list call. | Ratified 080753ZMAY26 |
| Q3 | Confirmation prompt on TTY for `remove`? | Yes, prompt on TTY; suppressible via `--force`. | Ratified 080753ZMAY26 |
| Q4 | Permission flag shape: `--permission` repeatable + comma-sep CSV? | Yes — matches `org invitation create`. Empty set clears all grants. | Ratified 080753ZMAY26 |
| Q5 | Pagination on `list` via `--limit` / `--cursor` / `--all`? | Yes — matches existing paginated list ergonomics. | Ratified 080753ZMAY26 |

## Acceptance criteria

- A1. `org member list` renders member table with slug, owner flag, permissions, joined date; supports `--output`.
- A2. `org member set-permissions` replaces permission set and confirms success; validates atoms client-side.
- A3. `org member remove` removes member; surfaces `self_removal_lockout` and `cannot_remove_owner` as actionable errors.
- A4. `<member>` slug-resolution works: pass slug → resolved to UUID before API call.
- A5. Tests for all three verbs including error paths (403 lockout, 403 owner, 400 invalid_permission).
- A6. `docs/cli.md` updated.
- A7. `make verify` passes.
