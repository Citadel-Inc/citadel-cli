# Spec — cli-watch

| | |
|---|---|
| Status | DRAFT 050826ZMAY26 |
| Authored | 050826ZMAY26 |
| Owner | Bastion (J-3) |
| Carry-forward from | 2026-05-05 enhancement sweep: operators today poll `repo list` / `agent list` in a `watch -n` loop to monitor namespace activity. The CLI should ship native streaming. |

## Why

`kubectl get pods -w` is the gold-standard live-list UX. citadel-cli has no equivalent — operators wrap list verbs in `watch(1)` and re-paint the whole table every interval, with no diff between runs and no event stream when something changes mid-poll. The server already has the data; this spec adds an SSE stream + a `--watch / -w` flag on every list verb.

## In scope

### Server (Citadel) — companion repo

- **SSE endpoints**: every list endpoint gains a streaming sibling at the same path with `Accept: text/event-stream`. Initial event = the current snapshot (one event per row, type=`init`); subsequent events fire on row create / update / delete (type=`add` / `update` / `remove`).
- **Heartbeat**: comment-line `:keepalive` every 15 s so middleboxes don't kill idle connections.
- **Auth**: same Bearer auth as the REST verb. `Last-Event-ID` reconnect resumes from the last event seen (server keeps a small ring buffer per namespace).
- **Resources covered**: same set as cli-pagination (repo, agent, oauth client, namespace member, transfer, token).

### CLI

- **`--watch / -w` on every list verb**: switches the handler from the REST snapshot path to the SSE path. Stream events to stdout incrementally; flush after each.
- **Output format interaction**:
  - `--output ndjson --watch`: one JSON object per line per event, `{type, ts, payload}`. Canonical for piping into `jq` / log tailers.
  - `--output table --watch` (default): re-render the tabwriter on every event. Uses ANSI cursor-up only when `colorEnabled(cmd)` is true (TTY + NO_COLOR unset); falls back to append-only blocks otherwise.
  - `--output json --watch`: explicitly disallowed (single-array shape doesn't stream). Hard error with a hint pointing at `--output ndjson`.
- **Reconnect**: on stream drop, retry with expo backoff (250 ms → 4 s, capped) re-using the existing `internal/apiclient/transport.go` retry policy. Send `Last-Event-ID` from the most recent server event.
- **Exit**: `--watch` runs until Ctrl-C / SIGTERM / context-cancel; clean exit 0 on signal. Exit 1 on auth failure, exit 2 on non-recoverable HTTP (404 / 410).

## Out of scope

- **Filter expressions on the watch stream** (`-w --selector status=active`): server-side filtering is a separate spec; v1 streams everything.
- **Watch on get verbs** (`repo get foo -w`): tail a single resource's edits. Useful but separate; needs a different server contract.
- **MCP `tools/list` watching**: tool-set is essentially static; no demand.
- **Local file-system watch on auth state** (login/logout in another terminal): unrelated.
- **WebSocket fallback**: SSE is HTTP/1.1-friendly and the CLI doesn't need bidirectional. Skip.

## Decision log

| Q | Proposal | Status |
|---|----------|--------|
| Q1 | SSE vs long-poll vs WebSocket? | **Open** — SSE; one-way, framing built in, replays via Last-Event-ID. |
| Q2 | Default `--output` under `--watch`: table-redraw vs append-only blocks? | **Open** — table-redraw on TTY, append on non-TTY. Auto. |
| Q3 | Reconnect: max attempts vs unbounded? | **Open** — unbounded with capped backoff; Ctrl-C is the user's exit. |
| Q4 | Heartbeat interval 15 s vs 30 s? | **Open** — 15 s; matches typical idle-timeout floors (Cloudflare 100 s, AWS ALB 60 s). |
| Q5 | `--watch` + `--limit/--cursor`/`--all` (cli-pagination) interplay: snapshot only, or paginate the init burst? | **Open** — paginate the init burst transparently; emit init events page-by-page. |

## Acceptance

- A1. `citadel-cli repo list -w -n org` streams the current set as `init` events, then live `add`/`update`/`remove` as the namespace mutates.
- A2. `citadel-cli repo list -w --output ndjson` emits one JSON object per line of shape `{"type":"add","ts":"...","payload":{...}}`.
- A3. Stream drop → reconnect within ≤ retry-backoff bounds; resumes from `Last-Event-ID`.
- A4. SIGINT during watch → clean exit 0, stdout flushed.
- A5. `--output json --watch` errors with a friendly hint.
- A6. Heartbeat keepalives received every ≤ 15 s; not surfaced to the user except in `--debug-http`.
- A7. Q-table ratified.

## Open questions for NOMAD

- Server SSE contract (Q1 + event shape).
- Default-mode rendering on TTY (Q2).
- Init-burst pagination semantics (Q5) — depends on cli-pagination landing first.
