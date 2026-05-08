# Plan — cli-pull-requests

## Implementation order

1. `cmd/pr.go` — PrCmd root + `pr list`, `pr view`, `pr create`, `pr close`, `pr merge`
2. `cmd/pr_comment.go` — `pr comment list`, `pr comment add`
3. `cmd/pr_reviewer.go` — `pr reviewer list`, `pr reviewer add`
4. `cmd/pr_diff.go` — `pr diff` (raw text) + `pr check` (mergeability)
5. `cmd/pr_review.go` — `pr review --approve/--request-changes/--comment`
6. `cmd/pr_handler_test.go` — all handler tests
7. `docs/cli.md` update

## Key patterns

- `-R`/`--repo` flag: `c.Flags().StringP("repo", "R", ...)` per-command; resolved via `resolvePRNamespacePath()` helper (analogous to `resolveIssueNamespacePath`)
- Namespace path encoding: `url.PathEscape(nsPath)` — daemon uses `{slug}` which accepts `org%2Frepo` encoded path
- PR number: `strconv.FormatInt(num, 10)` in URL path; parsed from args with `strconv.ParseInt`
- Friendly errors: `prFriendlyError()` wrapping `*apiclient.HTTPError` → `already_merged`, `merge_conflict`, `approval_required`, `invalid_state`
- Diff output: `c.Get(ctx, path, nil)` won't work for raw text — need custom streaming or accept JSON + emit `Files[].Unified` field. Check if diff endpoint returns JSON with unified field or raw text.

## Unknowns to verify

- Does `GET .../diff` return `Content-Type: application/json` with `PRDiffResult` (file stats, no unified text) or raw text? → check handler.
- File diff endpoint returns `CommitFileUnifiedDiffResult{Unified string}` — so raw text is in the JSON `.unified` field.
- Full diff: `PRDiffResult{Files []PRFileStat}` — file stats only, no unified text per file. To stream raw diff, need to concatenate per-file unified diffs via separate requests, OR just print the structured file list and note the `--file` flag.

**Resolution**: `pr diff` with no `--file` → prints structured file-stat table (additions/deletions per file) + note to use `--file` for per-file unified text. `pr diff --file <path>` → calls the file-diff endpoint and prints `.unified` raw text.
