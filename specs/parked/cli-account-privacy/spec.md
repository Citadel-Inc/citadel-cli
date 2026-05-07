# Spec — cli-account-privacy

| | |
|---|---|
| Status | PARKED 072200ZMAY26 — Settings-panel concern, not a dev-loop workflow. No GitHub CLI analogue. Privacy preference toggles belong in a browser/UI settings surface; they are not actions a developer would need mid-session. Superseded by the CLI-as-workflow-tool principle. |
| Authored | 072308ZMAY26 |
| Owner | Copilot |
| Carry-forward from | Dossier-backed account settings follow-up: the daemon already ships `GET/PATCH /api/account/privacy`, while `citadel-cli` still exposes no way to inspect or change privacy preferences from the terminal. |

## Why

The dossier's logged-in surface includes settings/profile workflows, and the daemon already persists per-user privacy knobs for telemetry, frecency, and avatar auto-import.  
Without a CLI surface, operators on headless hosts or SSH-only workflows cannot inspect or mutate those preferences without raw API calls.

## In scope

- `citadel-cli account privacy get`
- `citadel-cli account privacy set`
- Human-default output plus structured output for reads and writes
- Tests and docs for privacy preference inspection and mutation

### API mapping

| Verb | Method + Path |
|------|---------------|
| `get` | `GET /api/account/privacy` |
| `set` | `PATCH /api/account/privacy` |

### Fields

- `telemetry_opt_out`
- `frecency_opt_out`
- `avatar_auto_import_oauth`
- `avatar_auto_import_gravatar`

## Out of scope

- New privacy settings not already present in the daemon
- Data export / deletion workflows
- Reworking telemetry or frecency behavior beyond toggling the existing prefs

## Decision log

| Q | Proposal | Status |
|---|----------|--------|
| Q1 | Nest under `account privacy` alongside the existing account-management tree. | **Ratified 072308ZMAY26** — keeps user-scoped settings together. |
| Q2 | Use a single `set` verb with explicit boolean flags instead of one subcommand per field. | **Ratified 072308ZMAY26** — mirrors the daemon's PATCH shape and keeps the surface compact. |
| Q3 | `get` defaults to a short human table/summary; `set` defaults to a human confirmation and supports `--json`. | **Ratified 072308ZMAY26** — matches the CLI's existing read/mutation output split. |

## Acceptance criteria

- `account privacy get` renders the four daemon-backed preferences
- `account privacy set` PATCHes any chosen subset of those preferences
- Empty/no-field mutations are rejected client-side before issuing the PATCH
- Docs cover both inspection and mutation from a headless workflow
- `make verify` passes
