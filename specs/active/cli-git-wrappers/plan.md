# Plan — cli-git-wrappers

Blocked on Q-table ratification (Q1–Q4).
No implementation work should begin until P0 human tasks are checked.

## Proposed file layout

```
cmd/git_clone.go        — citadel clone
cmd/git_push.go         — citadel push
cmd/git_pull.go         — citadel pull
cmd/git_auth.go         — shared auth injection (GIT_ASKPASS / credential helper)
cmd/git_wrappers_test.go — tests with mocked exec
```

## Key constraint

The wrappers must not shell out in a way that leaks credentials to process
listings. Use `GIT_ASKPASS` pointing to a short-lived temp script, or pipe
the token via stdin through `git credential-store`. Validate the approach
before P0 claims.
