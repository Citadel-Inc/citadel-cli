# Spec — cli-notifications

| | |
|---|---|
| Status | IN_PROGRESS 072204ZMAY26 — Copilot claims execution |
| Authored | 072309ZMAY26 |
| Owner | Copilot |
| Carry-forward from | Dossier-backed logged-in surface follow-up: the daemon already ships `notifapi` for inbox and notification preferences, but `citadel-cli` has no notification commands. |

## Why

The dossier's architecture includes notifications as a first-class logged-in surface, and the daemon already exposes inbox list/read/unread-count plus notification-preference routes.  
CLI users currently have no terminal inbox even though many audit and collaboration events already resolve into notification rows server-side.

## In scope

- `citadel-cli notification list`
- `citadel-cli notification read <id>`
- `citadel-cli notification read-all`
- `citadel-cli notification unread-count`
- `citadel-cli notification prefs get`
- `citadel-cli notification prefs set`
- Pagination/output support for inbox list
- Tests and docs for the notification surface

### API mapping

| Verb | Method + Path |
|------|---------------|
| `list` | `GET /api/me/notifications` |
| `read` | `POST /api/me/notifications/{id}/read` |
| `read-all` | `POST /api/me/notifications/read-all` |
| `unread-count` | `GET /api/me/notifications/unread-count` |
| `prefs get` | `GET /api/me/notification-prefs` |
| `prefs set` | `PATCH /api/me/notification-prefs` |

## Out of scope

- Desktop notification delivery
- Email notification sending or Mailgun configuration
- Notification creation/emission (daemon-owned)

## Decision log

| Q | Proposal | Status |
|---|----------|--------|
| Q1 | Use a top-level `notification` noun rather than nesting under `account`. | **Ratified 072309ZMAY26** — inbox workflows read more naturally as a first-class surface. |
| Q2 | `list` supports `--limit`, `--cursor`, `--all`, `--unread`, and standard list outputs. | **Ratified 072309ZMAY26** — matches existing paginated list ergonomics. |
| Q3 | Preference updates use a single `prefs set` mutation with explicit flags. | **Ratified 072309ZMAY26** — mirrors the daemon PATCH shape and avoids tiny one-field verbs. |

## Acceptance criteria

- Inbox list/read/unread-count work against the daemon routes above
- `notification list` supports cursor pagination and `--all`
- Notification preference get/set cover the daemon-backed in-app cadence/override fields the route exposes
- Docs explain how inbox triage works from the terminal
- `make verify` passes
