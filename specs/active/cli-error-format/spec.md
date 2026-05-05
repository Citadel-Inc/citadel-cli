# Spec — cli-error-format

| | |
|---|---|
| Status | IN_PROGRESS 051044ZMAY26 — Bastion (J-3) claims execution |
| Authored | 050900ZMAY26 |
| Owner | Bastion (J-3) |
| Carry-forward from | 2026-05-05 enhancement sweep: every CLI error today goes to stderr as plain text via `Error: %v` in main.go. Scripts piping `--output json` get a JSON document on stdout for the success path and an unparseable English sentence on stderr for the failure path. Operators must regex-match error strings to branch on error class. |

## Why

The CLI already commits to JSON-shape contracts on success (`--output json` per verb). The error path has no such contract. `cmd/errmap.go` carries rich classification logic — DNS / refused / timeout / 401 / 403 / 412 mfa-required / 429 / 5xx — but collapses every class to free-text English before main.go's `Fprintf(stderr, "Error: %v\n", ...)`. A script that wants to "retry on rate limit, hard-fail on auth, nag the user on timeout" must string-match the message.

This spec adds an error-envelope contract so callers can branch on `kind` / exit code, and threads it through stdout for machine-readable `--output` modes (`json`, `yaml`, `ndjson`) and stderr for human/table modes (friendly text preserved).

## In scope

### Error taxonomy

A small fixed set of `kind` values, frozen as part of the v1 contract:

| kind | When |
|---|---|
| `auth_required` | no token / 401 / refresh failed |
| `mfa_required` | 412 (oauth/oauth-clients write) |
| `forbidden` | 403 |
| `not_found` | 404 |
| `conflict` | 409 |
| `rate_limited` | 429 |
| `validation` | 400 with a structured server payload |
| `server_unavailable` | 502 / 503 / 504 |
| `server_error` | other 5xx |
| `timeout` | context deadline exceeded |
| `network` | DNS / dial failures |
| `dry_run` | `--dry-run` shortcut path (handler returned the dry-run sentinel) |
| `internal` | catch-all for unmapped errors |

### Envelope shape

```jsonc
{
  "error": {
    "kind": "rate_limited",
    "message": "rate limit exceeded — slow down or wait a few minutes before retrying",
    "http_status": 429,         // present iff the error originated from an HTTP response
    "retry_after_seconds": 60,  // present iff the server returned Retry-After
    "hint": "https://status.src.land",  // present iff a URL hint applies
    "details": { ... }          // optional, kind-specific structured payload
  }
}
```

### CLI surface

- **Machine-readable `--output` (`json` / `yaml` / `ndjson`) + error**: stdout receives the envelope; stderr stays empty; exit code is the kind-mapped code (table below).
- **Human mode + error**: stderr receives `Error: <message>\n` exactly as today — no regression. The taxonomy still drives the exit code.
- **Exit code map** (frozen as part of v1 contract):
  - 0 — success
  - 1 — generic / `internal`
  - 2 — `validation`, `dry_run` (already used for ErrToolCallFailed)
  - 3 — `auth_required`, `mfa_required`, `forbidden`
  - 4 — `not_found`
  - 5 — `conflict`
  - 6 — `rate_limited`
  - 7 — `server_unavailable`, `server_error`, `network`, `timeout`
- **`cmd/errmap.go` rewrite**: returns a `*cmd.CLIError` (new type) instead of `error` with classification baked in. Existing tests stay green via `Error()` string compatibility.
- **Verb migration**: each verb already calls `apiclient.Get/Post/...` and propagates `error`. Verbs that wrap the error with their own `fmt.Errorf("...: %w", err)` keep the wrap; the envelope unwraps to find the deepest `*CLIError` for classification.

## Out of scope

- **Stable error messages**: only `kind` / exit code are contractual. Free-text `message` may evolve.
- **Localization**: messages stay English. i18n is a separate spec if ever needed.
- **`--output yaml` / `--output ndjson` envelope**: render the same shape as JSON; ndjson emits the single envelope as one line.
- **Server-side error payload contract**: this spec consumes whatever the server sends today. A companion server spec for "structured 4xx payloads with a stable code field" is separate.
- **Logging the envelope to a file**: `--debug-http` already gives the full wire dump.

## Decision log

| Q | Proposal | Status |
|---|----------|--------|
| Q1 | Envelope key `error` vs top-level `{kind, message, ...}`? | **Ratified 051800ZMAY26** — nested `{"error": {...}}`. |
| Q2 | Exit code map: collapse to 0/1/2 (POSIX-ish) vs the per-class 0–7 above? | **Ratified 051800ZMAY26** — per-class 1–7 map as specified. |
| Q3 | `details` payload: include the raw server body (truncated) or a curated subset? | **Ratified 051800ZMAY26** — curated subset only. |
| Q4 | `Retry-After`: parse seconds-only or also HTTP-date form? | **Ratified 051800ZMAY26** — delta-seconds or HTTP-date per RFC 7231 (`apiclient.ParseRetryAfterSeconds`). |
| Q5 | `--output json` only, or also `--error-format json` independent of `--output`? | **Ratified 051800ZMAY26** — structured envelope when `--output` is `json`, `yaml`, or `ndjson` (single surface area). |

## Acceptance

- A1. `*CLIError` type with `Kind`, `Message`, `HTTPStatus`, `RetryAfter`, `Hint`, `Details` fields.
- A2. `errmap.go` returns `*CLIError` for every classified error class.
- A3. `main.go` top-level error path branches on output mode:
  - machine-readable `--output` (`json`, `yaml`, `ndjson`) → marshal envelope to stdout, no stderr text.
  - default → `Error: <message>` to stderr (today's behavior).
- A4. Exit code table A2 above is honored.
- A5. Golden tests in `cmd/errmap_test.go`: each kind produces the expected envelope + exit code under both output modes.
- A6. README + HUMANS.md document the envelope shape, the kind set, and the exit code map.
- A7. Q-table ratified.
