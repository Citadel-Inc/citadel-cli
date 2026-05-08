# Spec — cli-pull-requests

| | |
|---|---|
| Status | DRAFT 080821ZMAY26 |
| Priority | High — core developer workflow; `gh pr` analogue; no CLI PR surface exists |
| Authored | 080821ZMAY26 |
| Owner | Bastion (J-3) |
| Carry-forward from | `prsapi` ships full PR CRUD at `GET/POST /api/namespaces/{slug}/pulls` and related sub-routes; zero CLI coverage today. |

## Why

Pull requests are the primary code-review workflow primitive. Developers need `pr list`, `pr view`, `pr create`, `pr merge`, and review submission without leaving the terminal. All `prsapi` routes are live; this is a pure CLI gap covering the full developer-facing PR surface.

## In scope

- `citadel-cli pr list -R org/repo` — paginated list, `--state`, `--output`
- `citadel-cli pr view -R org/repo <number>` — view PR + reviewers, `--output`
- `citadel-cli pr create -R org/repo` — create PR; `--title`, `--body`, `--source`, `--target`
- `citadel-cli pr close -R org/repo <number>` — close (not merge); `--yes`
- `citadel-cli pr merge -R org/repo <number>` — merge; surface `merge_conflict`, `approval_required`, `already_merged`
- `citadel-cli pr diff -R org/repo <number>` — raw unified diff to stdout; `--file <path>` for single-file diff
- `citadel-cli pr check -R org/repo <number>` — mergeability check; human + `--output json`
- `citadel-cli pr comment list -R org/repo <number>` — list PR comments
- `citadel-cli pr comment add -R org/repo <number> --body <text>` — add general comment
- `citadel-cli pr reviewer list -R org/repo <number>` — list reviewers + status
- `citadel-cli pr reviewer add -R org/repo <number> <user>` — add reviewer by UUID or slug
- `citadel-cli pr review -R org/repo <number>` — submit review; `--approve` / `--request-changes` / `--comment <body>`

`-R`/`--repo` accepts `org/repo` or nested `org/project/repo` namespace path — same convention as `issue` and `repo` commands.

### API mapping

| Verb | Method + Path | Auth |
|------|---------------|------|
| `list` | `GET /api/namespaces/{slug}/pulls` | `pull_requests:read` |
| `view` | `GET /api/namespaces/{slug}/pulls/{number}` | `pull_requests:read` |
| `create` | `POST /api/namespaces/{slug}/pulls` | `pull_requests:write` |
| `close` | `DELETE /api/namespaces/{slug}/pulls/{number}` | `pull_requests:write` |
| `merge` | `POST /api/namespaces/{slug}/pulls/{number}/merge` | `pull_requests:write` |
| `diff` | `GET /api/namespaces/{slug}/pulls/{number}/diff` | `pull_requests:read` |
| `diff --file` | `GET /api/namespaces/{slug}/pulls/{number}/diff/file?path=` | `pull_requests:read` |
| `check` | `GET /api/namespaces/{slug}/pulls/{number}/mergeability` | `pull_requests:read` |
| `comment list` | `GET /api/namespaces/{slug}/pulls/{number}/comments` | `pull_requests:read` |
| `comment add` | `POST /api/namespaces/{slug}/pulls/{number}/comments` | `pull_requests:write` |
| `reviewer list` | `GET /api/namespaces/{slug}/pulls/{number}/reviewers` | `pull_requests:read` |
| `reviewer add` | `POST /api/namespaces/{slug}/pulls/{number}/reviewers` | `pull_requests:write` |
| `review` | `PUT /api/namespaces/{slug}/pulls/{number}/reviews/me` | `pull_requests:write` |

### Response shapes (abbreviated)

`PR` → `{id, namespace_id, number, title, body_markdown, state, source_ref, target_ref, head_sha, base_sha, merge_sha?, author_id, merged_by?, created_at, updated_at, merged_at?, closed_at?}`
`Reviewer` → `{user_id, status, updated_at}`
`Comment` → `{id, pr_id, author_id, body_markdown, thread_id?, diff_commit_sha?, diff_file?, diff_line?, diff_side?, created_at, updated_at}`
`PRDiffResult` → `{files: [{path, old_path?, status, additions, deletions}], base_ref, head_ref, base_sha, head_sha}`
`MergeabilityResult` → `{mergeable, reason?}` — reason one of `fast_forward | clean | conflict | no_merge_base | resolve_error`
`list` → `{pull_requests: [...], next_cursor}`

### Error codes to surface explicitly

| Code | Status | Friendly message |
|------|--------|-----------------|
| `already_merged` | 409 | PR is already merged |
| `merge_conflict` | 409 | Merge conflict — resolve conflicts before merging |
| `approval_required` | 422 | Required reviewer approval missing |
| `invalid_state` | 409 | PR is not in a state that allows this action |
| `missing_required_fields` | 400 | title, source-ref, and target-ref are required |
| `invalid_refs` | 400 | One or both refs could not be resolved |

## Out of scope

- Diff line-level comments (thread system in `Comment` — deferred to inline review UX)
- `GET /api/namespaces/{slug}/pulls` public/unauthenticated variant
- PR labels / milestones (not wired in prsapi)
- Draft PR status (not in current schema)

## Decision log

| # | Question | Proposed default | NOMAD |
|---|----------|------------------|-------|
| Q1 | Scope: core 5 only vs full surface? | Full surface — all 13 verbs | Ratified 080821ZMAY26 |
| Q2 | `pr diff` output: raw unified text vs structured file-stat table? | Raw unified text stdout, pipeable | Ratified 080821ZMAY26 |
| Q3 | Mergeability verb: `check` vs `mergeability` vs `status`? | `pr check` — short, action-oriented | Ratified 080821ZMAY26 |
| Q4 | Review submit: single verb with flags vs separate verbs? | `pr review --approve / --request-changes / --comment` | Ratified 080821ZMAY26 |
| Q5 | `-R`/`--repo` flag for namespace path? | Yes — consistent with `issue` and `repo` commands | Ratified 080821ZMAY26 |
| Q6 | `--state` default for `pr list`? | `open` — matches `issue list` and gh convention | Ratified 080821ZMAY26 |
| Q7 | TTY `$EDITOR` fallback for `pr create --body`? | Yes — consistent with `issue create` | Ratified 080821ZMAY26 |

## Acceptance criteria

- A1. `pr list` renders table with number, title, state, source→target, author; supports `--state`, `--output`, pagination.
- A2. `pr view` renders full PR detail with reviewers; supports `--output`.
- A3. `pr create` posts with title/body/source/target; prompts `$EDITOR` on TTY when body omitted.
- A4. `pr close` closes PR; `--yes` skips confirmation.
- A5. `pr merge` merges PR; surfaces `merge_conflict`, `approval_required`, `already_merged`, `invalid_state` as actionable errors.
- A6. `pr diff` streams raw unified text; `--file <path>` narrows to one file.
- A7. `pr check` reports mergeable/not-mergeable with reason.
- A8. `pr comment list` and `pr comment add` work.
- A9. `pr reviewer list` and `pr reviewer add` work (slug or UUID accepted).
- A10. `pr review --approve/--request-changes/--comment` submits review status.
- A11. Handler tests for all verbs including error paths (409, 422, 400).
- A12. `docs/cli.md` updated.
- A13. `make verify` passes.
