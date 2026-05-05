# Tasks — cli-error-format

Status: DRAFT 050900ZMAY26

## P0

- [ ] [HUMAN] NOMAD ratifies Q-table (Q1–Q5).
- [x] A1. Define `cmd.CLIError` (Kind, Message, HTTPStatus, RetryAfter, Hint, Details) implementing `error` with the today-equivalent `Error()` string.
- [x] A2. Migrate `cmd/errmap.go` to return `*CLIError` for every classification branch (DNS, dial, deadline, 401/403/404/409/412/429/5xx). Keep the existing `error.Error()` text byte-identical so `errmap_test.go` passes unchanged.
- [x] A3. Migrate `main.go` top-level error path: detect `--output=json` (resolved from any verb's `--output` flag), marshal envelope to stdout, exit with the kind-mapped code; default to today's `Error: %v` stderr line.

## P1

- [x] B1. Exit-code table per spec §Acceptance/A4 wired into a single `kindToExitCode(kind)` helper used by main.go.
- [x] B2. Wrap-aware classification: when a verb wraps a `*CLIError` with `fmt.Errorf("...: %w", err)`, top-level handler unwraps to find the deepest `*CLIError` and surfaces its kind/exit while keeping the wrapped message text.
- [x] B3. Golden tests `cmd/errmap_test.go`: matrix of (input_error → expected envelope JSON, expected exit code, expected stderr) per kind, both output modes.
- [x] B4. README + HUMANS.md sections: envelope shape, kind table, exit code table.
- [x] B5. `mcpclient.Error` (Kind/Message) maps to `*CLIError` at the cmd boundary so MCP failures share the envelope.

## P2

- [ ] C1. Live integration assertion: provoke a 429 against a real (or staged) server, confirm `--output json` emits the envelope with `retry_after_seconds`.
- [ ] C2. [HUMAN] Operator review: confirm the new exit-code map doesn't break any in-house wrapper scripts; document any breakage.
- [ ] C3. Spec close.
