# Spec — cli-account-avatar

| | |
|---|---|
| Status | PARKED 072200ZMAY26 — Settings-panel concern, not a dev-loop workflow. No GitHub CLI analogue. Avatar management belongs in a browser/UI; a terminal user would never reach for this during a coding session. Superseded by the CLI-as-workflow-tool principle. |
| Authored | 072300ZMAY26 |
| Owner | Copilot |
| Carry-forward from | PoL dossier follow-up after `cli-auth-providers`: the daemon already ships `fe-profile-avatar-import` (`POST /api/account/avatar/import`) and `fe-profile-avatar-sync` (`PUT /api/account/avatar/sync`), but `citadel-cli` still exposes no account-avatar workflow beyond passkeys and devices. |

## Why

Citadel operators can already manage account security from the terminal, but avatar import and avatar-sync remain web-only even though the daemon routes are live.  
Adding a small `account avatar` surface closes that CLI gap for provider-avatar workflows and gives Phase 0/1 demos a terminal path for import plus opt-in sync.

## In scope

- `citadel-cli account avatar import --source github|google|gravatar`
- `citadel-cli account avatar sync --source github|google|gravatar --enable`
- `citadel-cli account avatar sync --source github|google|gravatar --disable`
- Tests and docs for the new account-avatar surface

### API mapping

| Verb | Method + Path |
|------|---------------|
| `import` | `POST /api/account/avatar/import` |
| `sync` | `PUT /api/account/avatar/sync` |

### Cross-cutting

- Mutation verbs keep human-default output plus `--json`
- `source` is constrained to the daemon's accepted values: `github`, `google`, `gravatar`
- `sync` forwards the daemon's `provider_not_linked`, `invalid_source`, and availability errors directly

## Out of scope

- Avatar upload from local files (already handled in the web app)
- Listing current avatar metadata beyond what the daemon mutation responses return
- Org-avatar import or org-avatar sync

## Decision log

| Q | Proposal | Status |
|---|----------|--------|
| Q1 | Nest under `account avatar <verb>` alongside existing `account passkey` / `account device`. | **Ratified 072300ZMAY26** — keeps account-management concerns together. |
| Q2 | Use a single `sync` verb with mutually-exclusive `--enable` / `--disable` flags instead of separate `enable` / `disable` subcommands. | **Ratified 072300ZMAY26** — keeps the daemon's single PUT shape visible in the CLI. |
| Q3 | Restrict `--source` to the daemon's known values only. | **Ratified 072300ZMAY26** — avoids speculative provider IDs and matches the server contract. |

## Acceptance criteria

- `account avatar import` POSTs the selected source and renders the returned avatar path/URLs
- `account avatar sync` PUTs `{source, enabled}` with clear validation for `--enable` / `--disable`
- Errors from the daemon surface clearly enough for provider-link and availability troubleshooting
- `docs/cli.md` and `HUMANS.md` document the avatar-account workflow
- `make verify` passes
