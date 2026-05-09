# Spec — cli-issue-labels

| | |
|---|---|
| Status | IN_PROGRESS 091315ZMAY26 — Bastion (J-3) claims execution |
| Priority | Medium — completes label management surface; assign/remove already ship |
| Authored | 091311ZMAY26 |
| Owner | Bastion (J-3) |
| Carry-forward from | `issuesapi` ships full label CRUD at `GET/POST/PATCH/DELETE /api/namespaces/{slug}/labels`; zero CLI CRUD coverage today. `issue label <number>` (assign/remove), `issue list --label`, and `issue create --label` are already implemented. |

## Why

Namespace-level label management (list, create, edit, delete) is missing from the CLI. Operators cannot create custom labels, rename defaults, or remove stale labels without going through the API directly. All four CRUD routes are live in `issuesapi`. This spec closes the remaining label surface as a top-level `citadel-cli label` command group — matching the `gh label` pattern.

## In scope

- `citadel-cli label list -R org/repo` — list all labels in namespace; `--output`
- `citadel-cli label create -R org/repo --name <name> --color <hex> [--description <text>]` — create label
- `citadel-cli label edit -R org/repo <slug>` — update name, color, or description; `--name`, `--color`, `--description`
- `citadel-cli label delete -R org/repo <slug>` — delete label; `--yes` bypass; surface `labelDeleteGuard` error as actionable message

### API mapping

| Verb | Method + Path | Auth |
|------|---------------|------|
| `list` | `GET /api/namespaces/{slug}/labels` | `issues:read` |
| `create` | `POST /api/namespaces/{slug}/labels` | `issues:write` |
| `edit` | `PATCH /api/namespaces/{slug}/labels/{label_slug}` | `issues:write` |
| `delete` | `DELETE /api/namespaces/{slug}/labels/{label_slug}` | `issues:write` |

### Response shapes (abbreviated)

`Label` → `{id, namespace_id, slug, name, color, description?, created_at, updated_at}`
`list` → `{labels: [...]}`

### Error codes to surface explicitly

| Code | Status | Friendly message |
|------|--------|-----------------|
| `label_not_found` | 404 | Label `<slug>` not found |
| `label_already_exists` | 409 | Label `<slug>` already exists |
| `label_delete_guard` | 422 | Cannot delete last default label for semantic role |

## Out of scope

- `issue label <number> --add/--remove` — already implemented in `cmd/issue.go`
- `issue list --label` filter — already implemented
- `issue create --label` — already implemented
- Label seeding / default-label management (operator surface, not end-user)
- Bulk label operations

## Decision log

| # | Question | Proposed default | NOMAD |
|---|----------|------------------|-------|
| Q1 | Top-level `label` command vs subcommand of `issue`? | Top-level — matches `gh label` pattern; cleaner UX | Pending |
| Q2 | `-R`/`--repo` flag convention? | Yes — consistent with `issue`, `pr`, `repo` commands | Pending |
| Q3 | `edit` verb or `update`? | `edit` — shorter, consistent with `gh label edit` | Pending |
| Q4 | `--yes` flag for delete? | Yes — destructive operation, guard by default | Pending |
| Q5 | `--color` format: bare hex vs `#`-prefixed? | Accept both; normalise to `#rrggbb` before POST | Pending |

## Acceptance criteria

- A1. `label list -R org/repo` renders table with slug, name, color, description; supports `--output`.
- A2. `label create -R org/repo --name foo --color #a2eeef --description "bug track"` creates label; prints slug on success.
- A3. `label edit -R org/repo <slug>` patches only supplied fields; errors if none supplied.
- A4. `label delete -R org/repo <slug>` deletes label; `--yes` skips confirmation; `labelDeleteGuard` 422 surfaced as actionable error.
- A5. Shell completion for label slugs on `edit` and `delete`.
- A6. Handler tests for all verbs including error paths (404, 409, 422).
- A7. `docs/cli.md` updated.
- A8. `make verify` passes.
