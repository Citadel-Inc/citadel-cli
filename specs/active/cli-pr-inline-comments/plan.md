# Plan — cli-pr-inline-comments

## Implementation order

1. `cmd/pr_collab.go` — extend `prComment` struct with inline fields; add flags to `prCommentAddCmd`; update `runPRCommentAdd`
2. `cmd/pr_collab.go` — extend `runPRCommentList` with `--inline` / `--general` filter and thread-group human render
3. `cmd/pr_handler_test.go` — handler tests for new paths
4. `docs/cli.md` update

## Key changes in pr_collab.go

### prComment struct additions

```go
type prComment struct {
    // existing fields unchanged
    DiffCommitSHA *string `json:"diff_commit_sha,omitempty"`
    DiffFile      *string `json:"diff_file,omitempty"`
    DiffLine      *int    `json:"diff_line,omitempty"`
    DiffSide      *string `json:"diff_side,omitempty"`
}
```

### prCommentAddCmd flag additions

```go
prCommentAddCmd.Flags().String("diff-file", "", "File path for inline comment (requires --diff-line)")
prCommentAddCmd.Flags().Int("diff-line", 0, "Hunk line number for inline comment (requires --diff-file)")
prCommentAddCmd.Flags().String("diff-side", "right", "Diff side: left or right (default right)")
prCommentAddCmd.Flags().String("diff-sha", "", "Commit SHA to anchor inline comment")
prCommentAddCmd.Flags().String("thread-id", "", "Thread UUID to reply to an existing thread")
```

### Validation in runPRCommentAdd

```go
diffFile, _ := cmd.Flags().GetString("diff-file")
diffLine, _ := cmd.Flags().GetInt("diff-line")
if (diffFile == "") != (diffLine == 0) {
    return fmt.Errorf("--diff-file and --diff-line must be supplied together")
}
diffSide, _ := cmd.Flags().GetString("diff-side")
if diffSide != "left" && diffSide != "right" {
    return fmt.Errorf("--diff-side must be \"left\" or \"right\"")
}
```

### Thread grouping in runPRCommentList

When rendering human output:
- Group comments by `thread_id` (nil = standalone)
- Print standalone general comments first, then inline threads grouped under `▸ Thread <id> — <file>:<line>`

## Q-table rationale

- Q1 (extend vs new verb): Additive flags on existing command; no breakage; matches `gh` style.
- Q3 (client-side grouping): Server returns flat list — no thread endpoint. Group by thread_id in render.
