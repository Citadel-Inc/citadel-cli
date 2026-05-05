# Spec — cli-ssh-keys

| | |
|---|---|
| Status | DRAFT 050506ZMAY26 |
| Authored | 050506ZMAY26 |
| Owner | Bastion (J-3) |
| Carry-forward from | Phase 0 L0 git wire: operators must register SSH public keys for `git@git.src.land` without opening the SPA. Server: `internal/api/accountapi` — `GET/POST/DELETE /api/account/ssh-keys`. |

## Why

SSH authentication to Citadel git endpoints requires registered keys on the user account. Today only HTTP APIs exist; CLI users configure keys manually via HTTP clients.

## In scope

**Parent command:** `citadel-cli ssh-key` (singular noun matches GitHub `ssh-key` UX; alternatives ratified in Q-table).

| Verb | HTTP |
|------|------|
| `ssh-key list` | `GET /api/account/ssh-keys` |
| `ssh-key add --key-file …` or `--key …` | `POST /api/account/ssh-keys` |
| `ssh-key delete <id>` | `DELETE /api/account/ssh-keys/{id}` |

**Cross-cutting**

- Auth: existing JWT from `auth login`.
- **cli-output-formats** on list.
- Add from file reads PEM/OpenSSH public key material; validate non-empty before POST.

## Out of scope

- **Generating keypairs** — users run `ssh-keygen`; we only upload public material.
- **Per-repo deploy keys** — not in account SSH API surveyed.

## Decision log

| Q | Proposal | Status |
|---|----------|--------|
| Q1 | Command name `ssh-key` vs `account ssh-key`? | **Open** — top-level `ssh-key` for brevity. |
| Q2 | Add: stdin vs `--key-file` default? | **Open** — `--key-file` required unless stdin piped (mirror patterns elsewhere). |

## Acceptance

- A1. Three verbs implemented + httptest coverage.
- A2. `make verify` passes.
- A3. Docs in `docs/cli.md`.
- A4. Q-table ratified.
- A5. Optional live test `CITADEL_TEST_SSH_KEYS_LIVE=1`.
