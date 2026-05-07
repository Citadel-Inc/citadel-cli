# Spec — cli-issues

| | |
|---|---|
| Status | IN_PROGRESS 070030ZMAY26 — Bastion (J-3) claims execution |
| Authored | 062323ZMAY26 |
| Owner | Bastion (J-3) |
| Carry-forward from | Phase 0 review against PoL + `citadel` tracker (2026-05-06): the remaining first-class CLI product gap is issues. Server-side issue substrate, close-refs, labels, assignees, templates, audit, and MCP tools are already live; `citadel-cli` still lacks dedicated issue verbs. |

## Why

The Proof of Life dossier names **issues at any namespace level** as a Phase 0
differentiator. The daemon now ships the underlying issue platform — CRUD,
comments, labels, assignees, close-refs, audit, and MCP helpers — but the CLI
still stops at infrastructure verbs (`repo`, `namespace`, `project`, `oauth`).
That leaves the operator workflow incomplete:

- terminal users cannot file or manage a namespace issue without dropping to
  the web app or raw MCP calls;
- the PoL issue story is only partially exposed from the CLI surface; and
- the current draft spec (`cli-issue-pr`) over-scopes into PRs even though the
  PoL dossier explicitly keeps PR/MR out of Phase 0.

This follow-on tracks an **issues-only** CLI surface that matches shipped
daemon capabilities and the actual Phase 0 gap.

## In scope

### Primary verbs

- `citadel-cli issue list -R <ns/path> [--state open|closed|all] [--label <slug> ...] [--assignee <slug> ...] [--limit N] [--cursor X] [--all]`
- `citadel-cli issue view -R <ns/path> <number>`
- `citadel-cli issue create -R <ns/path> --title "..." --body "..." [--label <slug> ...]`
- `citadel-cli issue comment -R <ns/path> <number> --body "..."`
- `citadel-cli issue close -R <ns/path> <number>`
- `citadel-cli issue reopen -R <ns/path> <number>`
- `citadel-cli issue label -R <ns/path> <number> --add <slug> --remove <slug>`
- `citadel-cli issue close-refs -R <ns/path> <number>` — read-only close-ref status for PoL demo / ops debugging

### Cross-cutting

- Canonical repo/namespace selector is **`-R <ns/path>`**, matching `gh -R`
  muscle memory and existing `citadel-cli` repo-context rules.
- List verbs honor the existing pagination contract (`--limit`, `--cursor`,
  `--all`) and output modes (`json|yaml|ndjson|csv|table`) where shape permits.
- View/detail verbs honor `--output json|yaml|table`.
- TTY create/comment flows use `$EDITOR` when `--body` is omitted; piped stdin
  remains the non-interactive path.
- `--web` on `view` opens the issue in the browser via the existing browser
  helper.

## Out of scope

- **Pull requests / merge requests.** Explicitly outside PoL Phase 0. Track in a
  future dedicated `cli-pull-requests` spec only after the daemon has a real PR
  substrate.
- **Review verbs** (`approve`, `request-changes`, etc.).
- **Issue templates chooser / forms.** The daemon supports templates, but the
  initial CLI gap is daily-driver CRUD; template UX can follow once issue v1 is
  shipped.
- **Milestones and assignee mutation verbs.** The daemon has those surfaces, but
  they are not required to close the Phase 0 CLI parity gap. Read-side
  filtering on `--assignee` is in scope; write-side assignee management is a
  follow-on.
- **Notifications / watch verbs.**

## Decision log

| Q | Proposal | Status |
|---|----------|--------|
| Q1 | Replace `cli-issue-pr` with issues-only scope and defer PRs entirely? | **Ratified 062323ZMAY26** — yes; matches PoL boundaries and actual daemon readiness. |
| Q2 | Selector shape: `-R <ns/path>` vs separate `--namespace` / `--repo` flags? | **Ratified 062323ZMAY26** — `-R <ns/path>`. |
| Q3 | Create/comment body input: `$EDITOR` on TTY, stdin for pipes? | **Ratified 062323ZMAY26** — yes; mirrors established CLI ergonomics. |
| Q4 | `issue close` reason enum? | **Ratified 062323ZMAY26** — no separate close reason at v1; server lifecycle is binary open/closed and richer semantics stay in labels. |
| Q5 | Should PoL close-ref visibility be first-class in the CLI? | **Ratified 062323ZMAY26** — yes; add `issue close-refs` read-only. |

## Acceptance

- A1. `citadel-cli issue` parent plus `list`, `view`, `create`, `comment`,
  `close`, `reopen`, `label`, and `close-refs` subcommands ship with complete
  help text and standard auth/error handling.
- A2. `issue list` honors pagination + output-mode contracts and supports
  `--state`, repeated `--label`, and `--assignee` filters.
- A3. `issue view` returns full issue payload including comments and labels; JSON
  / YAML output is machine-readable.
- A4. `issue create`, `comment`, `close`, and `reopen` work non-interactively
  and under a TTY with `$EDITOR` fallback where applicable.
- A5. `issue label` applies add/remove mutations against the daemon label API.
- A6. `issue close-refs` exposes the daemon `/close-refs` read surface for a
  single issue number.
- A7. `--web` opens the issue URL in the default browser.
- A8. Tests cover happy paths plus representative 400/401/403/404/409 cases
  against httptest fixtures.
- A9. Env-gated live integration test (`CITADEL_TEST_ISSUES_LIVE=1`) performs a
  create + comment + close + close-refs round-trip against a real Citadel
  instance.
