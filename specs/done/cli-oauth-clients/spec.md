# Spec — cli-oauth-clients

| | |
|---|---|
| Status | DONE 071629ZMAY26 — OAuth client CLI support remains shipped: the opt-in live integration test and runbook landed, while any additional operator citadel-cli smoke stays as an out-of-band follow-up instead of a HUMAN_BLOCKERS entry. |
| Authored | 030619ZMAY26 |
| Owner | Bastion (J-3) |
| Carry-forward from | `fe-oauth-developers` Q7 (RATIFIED OOS) + retro line 117: "`cli-oauth-clients` — CLI parity for the Account → OAuth clients UI." + spec §55. |

## Why

`fe-oauth-developers` shipped the Account / Org → OAuth clients UI (create / list / rotate / revoke); `go-oauth-registry` shipped the API. CLI parity is missing — operators + power users can't script OAuth client management. NOMAD ratified Q7 deferring CLI to this follow-on.

## In scope

- `citadel oauth clients list [--org <slug>]`
- `citadel oauth clients create --name <n> --redirect-uri <uri> [...]`
- `citadel oauth clients show <client_id>`
- `citadel oauth clients rotate-secret <client_id>` — prints new secret once, exits.
- `citadel oauth clients revoke <client_id>` — typed-id confirm.
- `--output json` parity with other CLI verbs.

## Out of scope

- **DCR-tagged client management** — `go-oauth-dcr` follow-on adds the `--dcr` filter once that lands.
- **Bulk CSV import** — single-client operations only.
- **Token introspection** — `citadel oauth tokens` is a separate verb tree if ever needed.

## Decision log

| Q | Proposal | Status |
|---|----------|--------|
| Q1 | `rotate-secret`: print to stdout once + exit, with `--copy-to-clipboard` flag? | **Ratified 030359ZMAY26** — stdout once + exit; optional `--copy-to-clipboard` best-effort (`wl-copy` / `xclip` / `pbcopy`). |
| Q2 | `revoke` confirmation: typed client_id vs. `--yes`? | **Ratified 030359ZMAY26** — typed **resource UUID** (`id` from API) by default; `--yes` skips confirmation. |
| Q3 | List output columns: id, name, scopes, last-used? | **Ratified 030359ZMAY26** — table: `client_id`, name, scopes (joined), last-used; last-used column is `—` until API exposes usage; `--output json` returns full records. |

## Acceptance

- A1. All five verbs functional behind `oauth:manage` gate.
- A2. `--output json` emits machine-readable per verb.
- A3. Destructive verbs require typed-id confirm absent `--yes`.
- A4. Q-table ratified.
