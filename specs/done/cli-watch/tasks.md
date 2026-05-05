# Tasks — cli-watch

Status: DONE 051430ZMAY26 — CLI + Citadel deliver cli-watch: polling SSE v1 on every list path (`internal/api/listwatch`: init/add/update/remove, `:keepalive`, Last-Event-ID ring replay), CLI `sseclient` + `--watch`, B6 httptest sequence, and cobra ctx reset for repeatable ExecuteContext. P2 C2 remains operator live smoke only.

Automation covers SSE contract via Citadel handlers + CLI integration tests; operator smoke is still human-owned (C2).

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
