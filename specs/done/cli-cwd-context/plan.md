# Plan — cli-cwd-context

## Resolution order

```
1. --no-cwd-repo flag set?
     yes → require -R explicitly; no inference at all
     no  → continue
2. -R / --repo flag value?
     non-empty → return parsed ns/slug
     empty     → continue
3. CITADEL_REPO env var?
     non-empty → return parsed ns/slug
     empty     → continue
4. exec("git", "remote", "get-url", "origin") in CWD?
     fail (not a git repo / no origin) → friendly error
     ok → continue
5. Parse the URL:
     ssh://, git+ssh://, git@host:ns/slug.git → host=host, path=ns/slug
     https://host/ns/slug(.git)              → host=host, path=ns/slug
     other shapes                             → friendly error
6. host in citadelHosts() ?
     no  → friendly error ("origin remote points at <host>; pass -R explicitly or set CITADEL_GIT_HOSTS")
     yes → return ns,slug; print TTY hint to stderr
```

`citadelHosts()` returns the union of:

- Compile-time defaults: `api.src.land`, `src.land`, `git.src.land`, `mcp.src.land`.
- `CITADEL_GIT_HOSTS` env (comma-separated; deduped + lowercased).

## URL parsing

Single regex covers both shapes:

```go
var (
    repoHTTPS = regexp.MustCompile(`^https?://(?P<host>[^/]+)/(?P<ns>[^/]+)/(?P<slug>[^/.]+)(?:\.git)?/?$`)
    repoSSH   = regexp.MustCompile(`^(?:ssh://)?(?:[^@]+@)?(?P<host>[^:/]+)[:/](?P<ns>[^/]+)/(?P<slug>[^/.]+?)(?:\.git)?$`)
)
```

`(?P<...>)` named groups make the test fixtures easier to read than positional matching.

## Verb integration

Add a flag helper to cmd/output.go:

```go
func addRepoFlag(cmds ...*cobra.Command) {
    for _, c := range cmds {
        c.Flags().StringP("repo", "R", "", "Repository as <namespace>/<slug> (overrides CWD inference)")
        c.Flags().Bool("no-cwd-repo", false, "Disable CWD git-remote inference")
    }
}
```

Then per-verb:

```go
ns, slug, err := resolveRepoFlag(cmd)
if err != nil { return err }
```

`repo get` keeps its positional arg as a shortcut: if `args` non-empty, use that; else fall through to resolveRepoFlag.

## Estimated delta

| Component | LOC (rough) |
|-----------|-------------|
| `cmd/repocontext.go` (resolver + URL parsers) | 120 |
| `addRepoFlag` helper + per-verb wiring (×3 v1 verbs) | 30 |
| Tests (parsers + resolver + integration) | 150 |
| Docs (README + HUMANS) | 30 |
| **Total** | **~330** |

## Risks

- **`git` not on PATH**: rare on dev machines; absent CI runners might lack it. Detect and fall through to "git not available; pass -R" error.
- **Submodules / worktrees**: `git remote get-url origin` is correct in both because `git` itself resolves the right config file. Confirm with a worktree fixture in tests.
- **Self-hosted Citadel deploys**: hardcoded host list won't match. `CITADEL_GIT_HOSTS` env override covers that; document it.
- **TTY-warning noise**: heavy CI users will hate "Inferred -R ..." on stderr. Add `--quiet` or honor `CI=true` in a follow-on (out of scope here, see Q5).
