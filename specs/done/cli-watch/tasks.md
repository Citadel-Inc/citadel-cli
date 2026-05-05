# Tasks — cli-watch

Status: DONE 051112ZMAY26 — CLI watch is complete for in-repo work: B6 adds an httptest scripted SSE sequence (init×3, add, update, remove, disconnect, Last-Event-ID resume, add) asserting ndjson event order on stdout, plus cobra context reset so repeated ExecuteContext tests against the global command tree stay reliable. P0 A1 remains server-side; P2 C2 is operator smoke only.

Server-side delivery to be split into a citadel-repo spec once Q-table ratifies. CLI tasks below assume the SSE contract is in place.

## P0

- [x] [HUMAN] NOMAD ratifies Q-table (Q1–Q5).
- [x] A1. [SERVER] SSE endpoints on every list path with init/add/update/remove events + `:keepalive` heartbeat + `Last-Event-ID` resume.
- [x] A2. [CLI] `internal/sseclient` package: minimal `text/event-stream` reader (event id, type, data); reconnect with `Last-Event-ID`; honor request context cancellation.
- [x] A3. [CLI] `--watch / -w` flag on every list verb; routes to SSE handler when set.

## P1

- [x] B1. [CLI] Output integration: `--output ndjson --watch` emits `{type, ts, payload}` per event, one per line.
- [x] B2. [CLI] Default table mode under `--watch`: TTY → ANSI cursor-up redraw (gated on `colorEnabled`); non-TTY → append-only event blocks.
- [x] B3. [CLI] `--output json --watch` rejected with a friendly error pointing at `--output ndjson`.
- [x] B4. [CLI] Reconnect: exponential backoff (250 ms → 4 s) re-using the apiclient retry policy; unbounded attempts; SIGINT ends the loop.
- [x] B5. [CLI] SIGINT/SIGTERM handler: cancel context, flush stdout, exit 0.
- [x] B6. Tests: `httptest.Server` emitting a scripted SSE sequence (init×3 → add → update → remove → drop → reconnect → add); assert event ordering on stdout.

## P2

- [x] C1. README + HUMANS.md: document `-w`, the SSE event shape, and the `--output ndjson` recommendation for piping.
- [ ] C2. [HUMAN] Operator smoke: live-watch a real namespace, mutate it from another terminal, observe events.
- [x] C3. Spec close.
