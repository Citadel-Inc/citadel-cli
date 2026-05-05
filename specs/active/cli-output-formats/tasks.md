# Tasks — cli-output-formats

Status: IN_PROGRESS 051044ZMAY26 — Bastion (J-3) claims execution

## P0

- [x] [HUMAN] NOMAD ratifies Q-table (Q1-Q5).
- [x] A1. `cmd/output.go`: extend `emitList` + `emitOne` with `ndjson`, `csv`, `yaml` cases. Reject unknown formats with `--output: unknown format X (use json|ndjson|csv|yaml)`.
- [x] A2. Per-verb csv column order: define a `csvColumns()` method + `csvRow(row)` formatter on each row type (`repoRow`, `agentRow`, `nsRow`, `nsMemberRow`, `nsTransferRow`, `nsOrgRow`, `oauthClient`, `token`).

## P1

- [x] B1. yaml support via `gopkg.in/yaml.v3`. Add to go.mod; encode time.Time as ISO-8601 strings.
- [x] B2. ndjson encoder: stream one JSON object per line; LF-terminated including last line.
- [x] B3. README "Output formats" section: per-verb csv schema table; usage examples for each format.
- [x] B4. Tests: parameterised test running each list + get verb against each of `json|ndjson|csv|yaml` with golden-output comparisons.

## P2

- [x] C1. Cross-cutting test: `--all` (cli-pagination spec) under `--output ndjson` produces one valid JSON object per line across pages.
- [ ] C2. [HUMAN] Operator smoke: spreadsheet-paste a real csv export.
- [ ] C3. Spec close.
