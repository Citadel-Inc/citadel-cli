# Tasks — cli-completion-dynamic

Status: IN_PROGRESS 050921ZMAY26 — Bastion (J-3) claims execution

## P0

- [ ] [HUMAN] NOMAD ratifies Q-table (Q1–Q5).
- [ ] A1. Add `internal/completion` package: cache reader/writer (XDG path, 60 s TTL, JSON array codec) + `Lookup(ctx, server, token, resource)` returning canonical slug list.
- [ ] A2. `cmd/repo.go`: register `ValidArgsFunction` on `repo get` / `repo delete` returning repo slugs scoped to the resolved namespace.
- [ ] A3. `cmd/namespace.go`: register `ValidArgsFunction` on namespace verbs taking a slug positional.
- [ ] A4. `cmd/agent.go`: register `ValidArgsFunction` on `agent get/delete/rotate-token`.
- [ ] A5. `cmd/oauth_clients.go`: register `ValidArgsFunction` on `oauth clients get/delete`.
- [ ] A6. `cmd/token.go`: register `ValidArgsFunction` on `token revoke`.

## P1

- [ ] B1. Cache invalidation: every mutating verb (`*Cmd.PostRun`) deletes the matching cache file. Best-effort, never blocks exit.
- [ ] B2. Static `--output` flag completion (cobra `RegisterFlagCompletionFunc`) on every verb that carries `--output`. List = format set from cli-output-formats acceptance.
- [ ] B3. Auth gating: any 401/403 from `Lookup` returns `cobra.ShellCompDirectiveError` silently — never surface a prompt or error string to the shell.
- [ ] B4. Tests: `cmd/handler_test.go` invokes `__complete repo get ""` against a fixture server and asserts the slug list comes back sorted + deduped.
- [ ] B5. Tests: cache TTL behavior — second call within 60 s uses cache (zero round trips), third call after TTL expiry refreshes once.

## P2

- [ ] C1. README + HUMANS.md: document the new completion behavior, the cache path, and the `CITADEL_NO_COMPLETION_CACHE=1` escape hatch (proposed bypass for debugging).
- [ ] C2. [HUMAN] Operator smoke: tab-complete `repo get` against a real namespace; verify ≤ 200 ms cache-hit latency.
- [ ] C3. Spec close.
