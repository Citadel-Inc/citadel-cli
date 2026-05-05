# Spec — cli-issue-pr

| | |
|---|---|
| Status | DRAFT 081550ZMAY26 |
| Authored | 081550ZMAY26 |
| Owner | Bastion (J-3) |
| Carry-forward from | third-pass review of `citadel-cli` (2026-05-05): biggest remaining product gap is first-class issue + PR verbs. Citadel server has issues + presumably PRs (per `go-issues` shipped + `go-issues-webhooks` DRAFT) but no CLI surface. |

## Why

`gh issue list/create/close/comment` and `gh pr create/list/merge/review` are the daily-driver verbs that operators expect from a code-host CLI. Today's `citadel-cli` covers infra (`repo`, `namespace`, `agent`, `oauth`) but stops short of the work surface. Operators wanting to file an issue from a terminal must drop into the web app.

Issues already have a backing schema + audit trail (the `issues`, `milestones`, `issue_labels`, `issue_close_refs` tables surfaced in repo-delete's cascade list). PR-side state is less concrete: the server appears to host bare repos via gitwire, but a structured "pull request" entity may or may not exist yet. This spec assumes:

- Issues: HTTP API exists or scope is small (find-or-fix at the daemon side).
- PRs: HTTP API may need to be built; daemon-side spec splits out if so.

CLI side is decoupled — verb shape is the same regardless of how the daemon delivers the data.

## In scope

### Issue verbs

- `citadel-cli issue list -R <ns>/<slug> [--state open|closed|all] [--label <l>] [--assignee <u>] [--limit N] [--cursor X] [--all]`
- `citadel-cli issue view -R <ns>/<slug> <number>` — full issue body + comment thread
- `citadel-cli issue create -R <ns>/<slug> --title "..." --body "..." [--label <l> ...] [--assignee <u> ...]`
- `citadel-cli issue comment -R <ns>/<slug> <number> --body "..."`
- `citadel-cli issue close -R <ns>/<slug> <number> [--reason completed|not-planned] [--comment "..."]`
- `citadel-cli issue reopen -R <ns>/<slug> <number>`
- `citadel-cli issue label -R <ns>/<slug> <number> --add <l> --remove <l>`

### PR verbs (gated on daemon-side support)

- `citadel-cli pr list -R <ns>/<slug> [--state open|closed|merged|all]`
- `citadel-cli pr view -R <ns>/<slug> <number>`
- `citadel-cli pr create -R <ns>/<slug> --head <branch> --base <branch> --title "..." --body "..."`
- `citadel-cli pr merge -R <ns>/<slug> <number> [--method merge|rebase|squash]`
- `citadel-cli pr close -R <ns>/<slug> <number>`

### Cross-cutting

- `-R <ns>/<slug>` flag canonical (matches `gh -R`); when CWD-context detection lands (companion spec `cli-cwd-context`) it auto-fills.
- All list verbs honor pagination contract from `cli-pagination` (`--limit`, `--cursor`, `--all`).
- All output supports the formats from `cli-output-formats` (`--output json|ndjson|csv|yaml`).
- Body/title prompts: when `--title` or `--body` is omitted under a TTY, open `$EDITOR` (mimics `gh issue create`); under a pipe, read from stdin.

## Out of scope

- **Review verbs** (`gh pr review --approve` etc.). Defer until daemon-side review/approval state is concrete.
- **Draft PRs**. v2 once the daemon shape is settled.
- **Issue templates / form-driven create**. Web app concern at v1.
- **Cross-repo linking** (`#42` resolution across multiple repos). Web app handles it; CLI just renders the raw text.
- **Notification / watch verbs** (`citadel-cli watch <repo>`). Different feature.

## Decision log

| Q | Proposal | Status |
|---|----------|--------|
| Q1 | Match `gh -R <ns>/<slug>` shape vs. `--namespace` + `--repo` separately? | **Open** — `-R <ns>/<slug>`; matches operator muscle memory. |
| Q2 | Body editor: respect `$EDITOR` (vi default) vs. force `nano`? | **Open** — `$EDITOR` (fallback `vi`); matches every other tool. |
| Q3 | `pr merge` default method: `merge` (preserves history) vs. `squash` (matches GH default for many repos)? | **Open** — `merge`; conservative default. |
| Q4 | Issue close `--reason`: free-form vs. enum (`completed`/`not-planned`)? | **Open** — enum, matches GH semantics. |
| Q5 | If daemon PR API doesn't exist yet, ship issue-only at v1 + split PR into a follow-on spec? | **Open** — recommend yes; reduces blast radius. |
| Q6 | `--web` flag (open issue/PR in browser) on view verbs? | **Open** — yes; trivial dep on `xdg-open`/`open`/`rundll32` (already used by `auth login`). |

## Acceptance

- A1. All issue verbs implemented; `make verify` passes including handler tests.
- A2. PR verbs implemented OR explicitly deferred per Q5 with companion spec carry-forward note.
- A3. `-R <ns>/<slug>` works on every verb.
- A4. List verbs honor pagination flags; output verbs honor `--output` formats.
- A5. `issue create --title` + `--body` work fully non-interactively (CI-friendly).
- A6. TTY-mode `issue create` without `--body` opens `$EDITOR`; piped stdin reads as body.
- A7. `--web` flag opens the resource in the default browser.
- A8. Q-table ratified.
- A9. Live integration test (gated on `CITADEL_TEST_ISSUES_LIVE=1`) creates + comments + closes an issue against a real test instance.
