# Spec — go-citadel-cli-repo

| | |
|---|---|
| Status | DONE 032036ZMAY26 — Shipped repo/namespace/agent CRUD CLI verbs against live APIs. repo create/list/get/delete; namespace list/get/members/transfer (with initiate/list-pending/accept/decline/revoke subcommands); agent list/get/delete/rotate-token. All verbs carry --help + --output json (A1). cmd_test.go integration suite covers command-tree structure, flag presence, and destructive-verb --yes gates (A2). Destructive verbs gate on typed-slug confirm (A3). Q-table ratified (A4). repo rename descoped (no server endpoint). namespace transfer org-only for now; personal namespace transfer deferred to server-side follow-on. |
| Authored | 030548ZMAY26 |
| Owner | Bastion (J-3) |
| Carry-forward from | `go-citadel-cli` D5 task ("Draft follow-on `go-citadel-cli-repo` spec for repo / namespace / agent CRUD verbs") + spec §Out-of-scope ("Repo / namespace operations defer to `go-citadel-cli-repo` follow-on once gitwire and the namespace API surface are real."). |

## Why

`go-citadel-cli` shipped find-or-create-on-issue agent flow + auth/profile basics; explicitly deferred repo/namespace/agent CRUD verbs pending real gitwire + namespace API. Both substrates now exist (`go-gitwire-http`, `go-gitwire-ssh` DONE; namespace API live). This spec ships the deferred verbs.

## In scope

- `citadel repo create|list|get|delete|rename` — CRUD against the existing repo HTTP API.
- `citadel namespace list|get|members|transfer` — read + admin operations.
- `citadel agent list|get|delete|rotate-token` — operations beyond the find-or-create primitive.
- Output: human (default) + `--output json` for scripting (parity with existing CLI verbs).

## Out of scope

- **Local clone / git plumbing** — `git clone` directly via gitwire URL is the user's job; CLI wraps API surface, not git.
- **Bulk import** (`citadel repo import-from <url>`) — separate follow-on.
- **Org/billing operations** — deferred until billing lands.

## Decision log

| Q | Proposal | Status |
|---|----------|--------|
| Q1 | `citadel repo delete` confirmation: `--yes` flag vs. typed slug confirm? | **Ratified 032130ZMAY26** — typed-slug confirm (interactive); `--yes` skips. Matches web danger-zone UX. |
| Q2 | `citadel namespace transfer` interactive flow vs. fully-flagged? | **Ratified 032130ZMAY26** — fully-flagged: `--namespace <slug> --to <new-owner>`; typed-slug confirm still guards destructive step. |
| Q3 | `citadel agent rotate-token` prints new token to stdout once + exits — no reveal-later? | **Ratified 032130ZMAY26** — stdout once + exits; no re-reveal path. Matches dashboard one-time reveal. |

## Acceptance

- A1. All verbs land with `--help` + `--output json`.
- A2. Verb set passes `citadel-cli/cmd/cmd_test.go` integration suite against a test droplet.
- A3. Destructive verbs (`delete`, `transfer`) require typed-slug confirmation absent `--yes`.
- A4. Q-table ratified.
