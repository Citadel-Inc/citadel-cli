# Plan — cli-audit-sessions

## ORIENT

- **Reference CLI:** `cmd/audit.go` — mirror flag parsing style from `audit list` (`since`/`until` on events != sessions — **different semantics**: sessions list uses **`since` start-of-window only**, no `until` in handler surveyed — verify before documenting user-facing flags).
- **Server:** `internal/api/auditapi/handler.go`.

## RECON delta (must verify before UX freeze)

1. **`handleListSessions`**: confirm whether **`until`** exists — if absent, CLI must not advertise `until` for sessions list (avoid lying parity with `audit list`).
2. **`handleGetSession`**: capture JSON struct (`sess` fields, cascade children) from `audit` package types — paste type definitions into appendix for implementers.

### Appendix — session list row (CLI projection)

The daemon returns **`{"sessions":[…]}`**. Each element decodes into at least:

```json
{
  "session_id": "<uuid>",
  "actor_slug": "<slug>",
  "actor_id": "<uuid>",
  "actor_type": "user",
  "namespace_slug": "<slug>",
  "started_at": "RFC3339",
  "last_event_at": "RFC3339",
  "event_count": 0
}
```

Unknown fields are ignored. **`handleGetSession`** responses are passed through for **`audit sessions show`** (operator-only keys omitted server-side for non-operators).

## Implementation sketch

- Add `auditSessionsCmd` with subcommands `list`, `show`.
- Reuse **`decodeSlugPath`** parity for `--ns` user input (slashes).

## Risks

- **Opaque 404** (`audit_sessions_unavailable`) vs wrong slug — stderr hint pattern matches events spec.
