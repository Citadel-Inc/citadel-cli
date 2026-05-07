# Plan — cli-issues

## Problem

`citadel-cli` still lacks first-class issue verbs even though `citadel` already
ships the Phase 0 namespace-issue platform. The prior draft (`cli-issue-pr`)
mixed a real gap (issues) with an explicitly out-of-scope surface (PRs), so it
was not a trustworthy execution plan.

## Exact daemon surfaces to target

- `GET /api/namespaces/{slug}/issues?state=&label=&assignee=&cursor=&limit=` →
  `{ issues: []Issue, next_cursor }`
- `POST /api/namespaces/{slug}/issues` with
  `{ title, body_markdown, labels?, milestone_id? }` → `Issue`
- `GET /api/namespaces/{slug}/issues/{number}` →
  `{ issue: Issue, comments: []Comment, labels: []Label }`
- `PATCH /api/namespaces/{slug}/issues/{number}` with `{ state? }` for
  close/reopen → `{ status: "ok", ... }`
- `POST /api/namespaces/{slug}/issues/{number}/comments` with
  `{ body_markdown }` → `Comment`
- `POST /api/namespaces/{slug}/issues/{number}/labels` with
  `{ add: [], remove: [] }` → `204 No Content`
- `GET /api/namespaces/{slug}/issues/{number}/close-refs` →
  `{ close_refs: []IssueCloseRef }`
- `GET|POST|PATCH|DELETE /api/namespaces/{slug}/labels...` exists, but namespace
  label CRUD itself is deferred from the first CLI cut; only issue label
  assignment is in scope here.

## CLI shape

```text
citadel-cli issue list       -R org/project     [filters...]
citadel-cli issue view       -R org/project 10  [--web]
citadel-cli issue create     -R org/project --title ... --body ...
citadel-cli issue comment    -R org/project 10 --body ...
citadel-cli issue close      -R org/project 10
citadel-cli issue reopen     -R org/project 10
citadel-cli issue label      -R org/project 10 --add bug --remove triage
citadel-cli issue close-refs -R org/project 10
```

## Reference patterns

- `cmd/audit.go` / `cmd/audit_sessions.go` — list + show verbs with shared output
  contracts and namespace scoping
- `cmd/oauth_clients.go` — CRUD command-tree layout and typed confirmations
- `cmd/project.go` — multi-verb parent command plus mixed read/write handlers
- `cmd/auth.go` / `cmd/agent.go` — browser helper + shell completion hooks for
  future ergonomic follow-ons

## Risks / decisions already made

- **No PR carry-along** — PRs stay out until the daemon has a real PR substrate.
- **Binary close lifecycle** — no `--reason` enum at v1; labels remain the
  richer semantics layer.
- **Close-ref visibility matters** — the PoL demo and operator workflows need a
  direct CLI read of issue close refs, not only MCP.
