## Approach

Add a `repo commit` subtree (`cmd/repo_commit.go`) with `list` and `get` subcommands.
Reuse `addPaginationFlags`, `readPagination`, `addOutputFlag`, `validateListOutput`, `validateGetOutput`,
`emitJSON`, `emitYAML`, `emitNDJSONLines`, `emitCSVRows`, and `emitOne` helpers already present in `cmd/`.
Wire `repoCommitCmd` into `cmd/repo.go` alongside the existing `BranchCmd` and `TagCmd`.

## Implementation notes

- `commitItem` domain type: `SHA`, `Author`, `AuthorEmail`, `Committer`, `CommitterEmail`, `Timestamp`, `Message` (full), `Subject()` helper (first line of message).
- Human table for list: SHA (8 chars), author, date (relative or ISO), subject line.
- `get` human output: key-value block similar to `repo get`; list changed files with +/- counts below.
- `get --path <file>`: call `/commits/{sha}/diff?path=<file>` and write response to stdout as-is (plain diff text); with `--output json` wrap in `{"unified":...}`.
- Shell completion: reuse `completeRepoPath` for the positional arg.
- Cursor opaque passthrough — do not base64-decode, just relay to `after=` query param.

## Risks

- Partial SHAs accepted by daemon (>=4 chars) but tab completion on SHAs is infeasible without a local cache; skip completion for sha arg.
- `--all` flag on list may be slow on large repos; add a reasonable default limit (30) consistent with other list commands.
