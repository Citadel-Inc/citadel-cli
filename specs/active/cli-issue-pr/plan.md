# Plan — cli-issue-pr

## Daemon-side survey (P0 task A1)

Before any CLI implementation lands, confirm:

1. **Issues HTTP API**: where it lives (likely `internal/api/issues/`), what endpoints exist, what shapes they return. If `GET /api/repos/{ns}/{slug}/issues` etc. are absent, file a companion spec `go-issues-cli-api` for the daemon side.
2. **PR API**: very likely absent. Q5 lets us ship issue-only at v1 to avoid blocking on a 6-month server build-out.
3. **Comment API**: same survey.
4. **Audit emission**: every state change should emit an audit event (already true per `go-issues-webhooks` carry-forward).

The CLI design below is decoupled — verb shape is what operators see; backend can deliver in any reasonable shape and the CLI thin-adapts.

## CLI shape

```go
// cmd/issue.go
var IssueCmd = &cobra.Command{
    Use:   "issue",
    Short: "Manage repository issues",
}

var issueListCmd = &cobra.Command{
    Use:   "list",
    Short: "List issues in a repository",
    RunE:  runIssueList,
}
// ...
```

Per-verb body sketch (issue view):

```go
func runIssueView(cmd *cobra.Command, args []string) error {
    repo, err := resolveRepoFlag(cmd) // -R <ns>/<slug> or cwd inference
    if err != nil { return err }
    n, err := strconv.Atoi(args[0])
    if err != nil { return fmt.Errorf("issue number: %w", err) }

    if webFlag(cmd) {
        url := fmt.Sprintf("https://src.land/%s/%s/issues/%d", repo.ns, repo.slug, n)
        return openBrowser(url)
    }

    c, err := newAPIClient(cmd)
    if err != nil { return err }
    var issue issueRow
    path := fmt.Sprintf("/repos/%s/%s/issues/%d", url.PathEscape(repo.ns), url.PathEscape(repo.slug), n)
    if err := c.Get(cmd.Context(), path, &issue); err != nil {
        if apiclient.IsStatus(err, http.StatusNotFound) {
            return fmt.Errorf("issue #%d not found in %s/%s", n, repo.ns, repo.slug)
        }
        return err
    }
    return emitOne(outputFlag(cmd), issue, renderIssueHuman)
}
```

Body-editor pattern reuses the `os.UserConfigDir()` / `tempfile` / `exec.Command(editor)` approach:

```go
func bodyFromFlagOrEditor(cmd *cobra.Command) (string, error) {
    if b, _ := cmd.Flags().GetString("body"); b != "" { return b, nil }
    if !isatty.IsTerminal(os.Stdin.Fd()) {
        b, err := io.ReadAll(os.Stdin)
        return strings.TrimSpace(string(b)), err
    }
    return promptEditor("# Issue body\n# Lines starting with # are ignored.\n")
}
```

`isatty.IsTerminal` is a new dep but tiny; alternative is `os.Stdin.Stat().Mode()&os.ModeCharDevice` which is stdlib-only — prefer the stdlib version.

## Estimated delta

| Component | LOC (rough) |
|-----------|-------------|
| Daemon survey + (optionally) companion spec | survey only |
| `cmd/issue.go` parent + 7 subcommands | 400 |
| `cmd/pr.go` parent + 5 subcommands (or deferred) | 250 |
| Body-editor + stdin helpers | 60 |
| `--web` browser-open helper reuse | 20 |
| Repo-flag resolver (compatible with cli-cwd-context spec) | 50 |
| Tests | 200 |
| Docs | 50 |
| **Total (issue-only v1 if Q5 ratifies yes)** | **~780** |
| **Total (issue + PR)** | **~1030** |

## Risks

- **Daemon PR API absence** is the load-bearing risk. Q5 ratification "yes" — issue-only v1, PR follow-on — keeps this spec tractable.
- **Body editor + Windows**: cobra has cross-platform editor invocation in `cobra/editor` somewhere; fall back to NOTEPAD on Windows. Verify with a CI matrix run before the spec closes.
- **Issue numbering scheme**: GitHub uses per-repo monotonic; Citadel may use namespace-scoped or globally unique. Confirm in survey before designing the URL shape.
