# Plan — cli-watch

## SSE event shape

```
event: add
id: 8125
data: {"slug":"alpha","namespace":"myorg","created_at":"2026-05-05T08:26:01Z"}

:keepalive

event: remove
id: 8126
data: {"slug":"beta","namespace":"myorg"}
```

Event types: `init` (snapshot), `add` (new row), `update` (existing row mutated), `remove` (row deleted), `error` (terminal — server can't continue).

## CLI handler shape

Same REST list path as pagination; the HTTP client sets `Accept: text/event-stream` (see `apiclient.Client.GetEventStream`). Example:

```go
// cmd/watch.go — repos
path := "/namespaces/" + url.PathEscape(ns) + "/repos?" + sseWatchQuery(limit, cursor, all, nil)
return consumeSSEWatch(cmd, c, path, h)
```

`sseclient.Open` owns reconnect + `Last-Event-ID`; `consumeSSEWatch` loops `Next()` until SIGINT/context cancel.

## Reconnect policy

Re-use `internal/httpx.Backoff`. On stream drop, `sseclient` reopens with `Last-Event-ID: <id-of-last-good-event>` and the same Bearer token. Max single-attempt timeout: 30 s for the connect; the stream itself runs without timeout. Heartbeat absence > 30 s = treat as drop.

## Output mode matrix

| `--output` | TTY?   | Behavior                                                |
|------------|--------|---------------------------------------------------------|
| (default)  | yes    | tabwriter snapshot, redrawn in place via `\033[<n>A`    |
| (default)  | no     | tabwriter snapshot, then per-event line `+slug` / `-slug` |
| ndjson     | any    | one JSON object per event per line                      |
| json       | any    | error: `--watch` requires --output ndjson or default`   |
| yaml       | any    | error (same)                                            |
| csv        | any    | error (same)                                            |

`colorEnabled(cmd)` gates the ANSI redraw. NO_COLOR / `--color=never` falls into the non-TTY behavior even on a TTY.

---
050826ZMAY26
