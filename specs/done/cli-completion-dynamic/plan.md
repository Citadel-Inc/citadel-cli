# Plan — cli-completion-dynamic

## Package layout

```
internal/completion/
  cache.go     // XDG cache read/write, TTL check
  lookup.go    // Lookup(ctx, server, token, resource) → []string
  cache_test.go
```

`Lookup` is the only public symbol. It opens an `apiclient.Client` (with `Verbose: false`, `DebugHTTP: false` — completion never narrates) and dispatches to the right list endpoint per resource.

## Cobra wire-up

```go
// cmd/repo.go (representative)
RepoGetCmd.ValidArgsFunction = func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
    if len(args) > 0 {
        return nil, cobra.ShellCompDirectiveNoFileComp
    }
    ns := namespaceFlag(cmd)
    cands, err := completion.Lookup(cmd.Context(), serverFlag(cmd), "", completion.RepoIn(ns))
    if err != nil {
        return nil, cobra.ShellCompDirectiveError
    }
    return cands, cobra.ShellCompDirectiveNoFileComp
}
```

Token resolution mirrors `newAPIClient`: `clicfg.Load()` → access token → empty-token = silent error directive.

## Cache file shape

`$XDG_CACHE_HOME/citadel-cli/completion/<server-host>/<resource>.json`:

```jsonc
{
  "fetched_at": "2026-05-05T08:26:00Z",
  "server": "https://api.src.land",
  "resource": "repos:myorg",
  "values": ["alpha", "beta", "gamma"]
}
```

Key path includes the server host so multi-server users (post cli-multi-context, if it lands) don't poison each other's cache.

## Mutation invalidation

Every write verb's `PostRunE` adds:

```go
PostRunE: func(cmd *cobra.Command, args []string) error {
    completion.Invalidate(serverFlag(cmd), completion.RepoIn(namespaceFlag(cmd)))
    return nil
}
```

`Invalidate` is a one-line `os.Remove` on the cache file path; ENOENT is fine.

## Test fixtures

`cmd/handler_test.go` already has a `httptest.Server` factory. Reuse it: stub `/api/namespaces/myorg/repos` → 3 repos, drive `__complete repo get ""`, assert candidates.

---
050826ZMAY26
