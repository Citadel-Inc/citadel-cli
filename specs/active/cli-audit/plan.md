# Plan — cli-audit

## Server response shape

```jsonc
// GET /api/audit/events?since=1h&kind=repo.*&namespace=myorg
{
  "events": [
    {
      "id": "01H...",
      "ts": "2026-05-05T08:00:00Z",
      "kind": "repo.deleted",
      "actor_id": "...",
      "actor_slug": "alice",
      "namespace_id": "...",
      "namespace_slug": "myorg",
      "subject_id": "...",         // affected entity
      "payload": { ... },          // event-specific
      "request_id": "..."          // best-effort
    },
    ...
  ],
  "next_cursor": "..."
}
```

Mirrors the cli-pagination envelope so `--all` works for free.

## Filter parsing

```go
func parseTimeFilter(s string) (time.Time, error) {
    if d, err := time.ParseDuration(s); err == nil {
        return time.Now().Add(-d), nil
    }
    return time.Parse(time.RFC3339, s)
}
```

Glob-kind matched server-side via `path.Match` semantics for portability (`repo.*` matches `repo.deleted` but not `repo.permission.granted`; `repo.**` two-level).

## CLI render

Default human output is one event per line, tabwriter:

```
TS                    KIND              ACTOR  NAMESPACE  SUBJECT
2026-05-05T08:00:00Z  repo.deleted      alice  myorg      old-repo
2026-05-05T08:02:31Z  agent.created     alice  -          myagent
2026-05-05T08:05:12Z  oauth.client.rev  alice  myorg      <uuid>
```

`audit show` prints structured detail:

```
ID:        01H...
Time:      2026-05-05T08:00:00Z
Kind:      repo.deleted
Actor:     alice (uuid 00000000-...)
Namespace: myorg
Subject:   old-repo
Request:   req_abc123

Payload:
  reason: operator_purge
  cascade: { kg_files: 421, kg_symbols: 1893, ... }
```

## Estimated delta

| Component | LOC (rough) |
|-----------|-------------|
| Daemon `/api/audit/events` endpoint + tests | 300 |
| CLI cmd/audit.go (list + show) | 250 |
| Time-filter parser + glob | 60 |
| Actor-slug local cache | 60 |
| Tests | 180 |
| Docs | 50 |
| **Total** | **~900** |

## Risks

- **Server-side audit-events endpoint may not exist yet**: file companion spec before A2 starts.
- **Audit volume**: a busy namespace might emit thousands of events/day. Default `--since 24h` + pagination handles it; document the `--all` cost in HUMANS.
- **PII redaction**: payloads may contain user UUIDs or IPs (operator-only fields). Server-side RBAC handles redaction per Q6; CLI just renders what comes back.
- **Tail-mode UX**: deferred to v2 once daemon supports SSE. Document in the spec close that this is the next step.
