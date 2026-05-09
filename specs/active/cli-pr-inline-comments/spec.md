# Spec — cli-pr-inline-comments

| | |
|---|---|
| Status | IN_PROGRESS 091320ZMAY26 — Bastion (J-3) claims execution |
| Priority | Medium — completes PR review surface; general comments ship, inline/thread comments do not |
| Authored | 091319ZMAY26 |
| Owner | Bastion (J-3) |
| Carry-forward from | `cli-pull-requests` ships general `pr comment add/list`; server already supports inline fields (diff_file, diff_line, diff_side, diff_commit_sha, thread_id) in `prsapi` — CLI does not expose them. |

## Why

`pr comment add` explicitly posts only "general (non-diff) comments." The server's `POST /api/namespaces/{slug}/pulls/{number}/comments` already accepts `diff_file`, `diff_line`, `diff_side`, `diff_commit_sha`, and `thread_id` — all backed by the `pull_request_comments` DB schema. Reviewers who want to leave inline comments on specific lines or reply to a thread are forced to use the web UI. This spec adds the missing flags to `pr comment add` and thread-aware display to `pr comment list`.

## In scope

- `pr comment add` — add `--diff-file`, `--diff-line`, `--diff-side`, `--diff-sha`, `--thread-id` flags for inline comments
  - `--diff-file` / `--diff-line` must be supplied together (error if only one)
  - `--diff-side` defaults to `"right"` when `--diff-file` is set; accepts `left` or `right`
  - `--diff-sha` optional; anchors comment to a specific commit SHA
  - `--thread-id` optional UUID; omit to start a new thread, supply to reply
- `pr comment list` — add `--inline` / `--general` filter flags; group inline comments by thread_id in human output
- Shell completion for `--diff-file` (file-path completion from `pr diff` file list)

### API mapping

| Verb | Method + Path | Notes |
|------|---------------|-------|
| `comment add` (inline) | `POST /api/namespaces/{slug}/pulls/{number}/comments` | Same endpoint; add inline fields to body |
| `comment list` | `GET /api/namespaces/{slug}/pulls/{number}/comments` | Same endpoint; filter/group client-side |

### Request body additions

```json
{
  "body_markdown": "...",
  "thread_id": "optional-uuid",
  "diff_commit_sha": "optional-sha",
  "diff_file": "path/to/file.go",
  "diff_line": 42,
  "diff_side": "right"
}
```

### Error codes to surface explicitly

| Code | Status | Friendly message |
|------|--------|-----------------|
| `invalid_inline_anchor` | 400 | `--diff-file` and `--diff-line` must be supplied together |
| `thread_not_found` | 404 | Thread `<id>` not found on this PR |
| `invalid_diff_side` | 400 | `--diff-side` must be `left` or `right` |

## Out of scope

- Suggest-changes patches (no server support yet)
- Batch inline comment submission (review drafts)
- Comment reactions / emoji
- Comment edit / delete (not in current prsapi surface)

## Decision log

| # | Question | Proposed default | NOMAD |
|---|----------|------------------|-------|
| Q1 | Extend `pr comment add` vs new verb `pr comment inline`? | Extend — same endpoint, additive flags; `gh` uses `gh pr comment --body` for both | Pending |
| Q2 | `--diff-side` default when `--diff-file` set? | `right` — change side is the overwhelmingly common case | Pending |
| Q3 | Thread grouping in list: client-side or new endpoint? | Client-side — server returns flat list; group by thread_id in table render | Pending |
| Q4 | `--thread-id` auto-complete? | No — UUIDs impractical; suggest `pr comment list` to find thread IDs | Pending |

## Acceptance criteria

- A1. `pr comment add <n> -R org/repo --body "..." --diff-file foo.go --diff-line 42` posts inline comment; prints comment ID.
- A2. `pr comment add` with only one of `--diff-file` / `--diff-line` → actionable error.
- A3. `pr comment add ... --diff-side left` accepted; invalid value → actionable error.
- A4. `pr comment add ... --thread-id <uuid>` posts reply in existing thread.
- A5. `pr comment list` default output groups inline comments under thread headers in human mode.
- A6. `pr comment list --general` shows only top-level (null diff_file) comments; `--inline` shows only inline/thread comments.
- A7. Handler tests for inline add, thread reply, validation errors (400), and filtered list.
- A8. `docs/cli.md` PR comment section updated.
- A9. `make verify` passes.
