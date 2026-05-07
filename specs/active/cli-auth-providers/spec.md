# Spec — cli-auth-providers

| | |
|---|---|
| Status | IN_PROGRESS 071813ZMAY26 — Copilot claims execution |
| Authored | 072200ZMAY26 |
| Owner | Copilot |
| Carry-forward from | No active `citadel-cli` specs remained after `cli-milestones`, so the next grounded CLI follow-up was selected from the PoL dossier plus shipped daemon capability. `citadel` already serves provider discovery and identity link/unlink routes, but the CLI still only exposes `auth login`, `auth logout`, `auth status`, and `auth set-token`. |

## Why

Citadel's Phase 1 OAuth-provider depth is already present server-side, but terminal users still cannot inspect which providers are enabled or manage linked identities without dropping to raw API calls.  
Adding a small `auth provider` surface closes that CLI gap without inventing new backend behavior.

## In scope

- `citadel-cli auth provider list`
- `citadel-cli auth provider link <provider>`
- `citadel-cli auth provider unlink <provider>`
- Provider-ID shell completion sourced from the daemon provider registry
- Tests and docs for the new command surface

### API mapping

| Verb | Method + Path |
|------|---------------|
| `list` | `GET /api/auth/providers` |
| `link` | `POST /api/auth/link-provider` |
| `unlink` | `POST /api/auth/unlink-provider` |

### Cross-cutting

- `list` uses the standard list output modes (`table`, `json`, `yaml`, `ndjson`, `csv`)
- `link` opens the returned `redirect_url` in the browser by default and supports structured output for headless automation
- `unlink` requires explicit confirmation unless `--yes` is passed
- Provider IDs are the daemon's canonical IDs (`github`, `google`, etc.), reused for completion and request payloads

## Out of scope

- Completing the OAuth callback inside the CLI
- Adding a linked-identities listing route the daemon does not currently expose
- Reworking the existing `auth login` / agent-token path

## Decision log

| Q | Proposal | Status |
|---|----------|--------|
| Q1 | Nest provider management under `auth provider <verb>`. | **Ratified 072200ZMAY26** — keeps auth lifecycle commands together and matches the existing second-level verb style. |
| Q2 | `link` should open the daemon-provided browser URL by default, but expose machine-readable output when requested. | **Ratified 072200ZMAY26** — convenient for humans while preserving automation. |
| Q3 | `unlink` should require confirmation unless `--yes` is passed. | **Ratified 072200ZMAY26** — destructive account changes should be deliberate. |

## Server contract

- `GET /api/auth/providers` returns `{ "providers": [{ "id", "label" }] }`
- `POST /api/auth/link-provider` accepts `{ "provider" }` and returns `{ "provider", "redirect_url" }`
- `POST /api/auth/unlink-provider` accepts `{ "provider" }`
- `link` and `unlink` reject unknown providers with `422 unknown_provider`
- `unlink` may reject the last removable provider with `412 last_provider_blocked`

## Acceptance criteria

- `auth provider list` renders enabled providers in human and structured formats
- `auth provider link` validates provider input, uses the saved session, and can either open or print the redirect URL
- `auth provider unlink` validates provider input, respects `--yes`, and surfaces daemon errors directly
- Provider completion works for `link` and `unlink`
- `docs/cli.md` documents browser-default and headless usage
- `make verify` passes
