# Plan — cli-issue-labels

## Implementation order

1. `cmd/label.go` — `LabelCmd` root + `label list`, `label create`, `label edit`, `label delete`
2. `cmd/label_handler_test.go` — handler tests for all four verbs + error paths
3. Shell completion hook for label slugs
4. `docs/cli.md` update

## Key patterns

- `-R`/`--repo` flag: `c.Flags().StringP("repo", "R", ...)` per-command; resolved via `resolveNamespacePath()` (same helper used by `issue` and `pr` commands)
- `label list`: `GET /api/namespaces/{slug}/labels` → decode `{labels: []Label}`; render as table with columns: SLUG, NAME, COLOR, DESCRIPTION
- `label create`: `POST /api/namespaces/{slug}/labels` with `{name, color, description}`; print `slug` from response on success
- `label edit`: `PATCH /api/namespaces/{slug}/labels/{label_slug}` — only include fields that were explicitly set (use `cmd.Flags().Changed("name")` etc.); error if zero fields changed
- `label delete`: `DELETE /api/namespaces/{slug}/labels/{label_slug}` — `--yes` flag skips `confirm.Confirm()`; map 422 `label_delete_guard` to human message
- Color normalisation: strip `#` prefix if present for storage if needed; re-add for display. Check server expectation (bare or prefixed) via `issuesapi` handler before implementing.
- Shell completion: `label edit` and `label delete` use `cobra.FixedCompletions` or dynamic completion via `GET /api/namespaces/{slug}/labels` to list slug candidates

## File layout

```
cmd/
  label.go               # LabelCmd + list/create/edit/delete subcommands
  label_handler_test.go  # handler tests (mock HTTP server)
docs/
  cli.md                 # label section added
```

## Q-table rationale

- Q1 (top-level vs subcommand): Top-level avoids collision with existing `issue label <number>` and matches `gh` convention. Cleaner completion tree.
- Q3 (edit vs update): `edit` is shorter and matches `gh label edit`. `update` also acceptable but less idiomatic.
- Q5 (color format): Accept `#rrggbb` and `rrggbb`; normalise before sending. Server stores bare hex — verify in `issuesapi/handler.go` before deciding normalisation direction.
