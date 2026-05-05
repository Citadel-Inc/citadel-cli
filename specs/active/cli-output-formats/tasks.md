# Tasks — cli-output-formats

Status: DRAFT 080800ZMAY26

## P0

- [ ] [HUMAN] NOMAD ratifies Q-table (Q1-Q5).
- [ ] A1. `cmd/output.go`: extend `emitList` + `emitOne` with `ndjson`, `csv`, `yaml` cases. Reject unknown formats with `--output: unknown format X (use json|ndjson|csv|yaml)`.
- [ ] A2. Per-verb csv column order: define a `csvColumns()` method + `csvRow(row)` formatter on each row type (`repoRow`, `agentRow`, `nsRow`, `nsMemberRow`, `nsTransferRow`, `nsOrgRow`, `oauthClient`, `token`).

## P1

- [ ] B1. yaml support via `gopkg.in/yaml.v3`. Add to go.mod; encode time.Time as ISO-8601 strings.
- [ ] B2. ndjson encoder: stream one JSON object per line; LF-terminated including last line.
- [ ] B3. README "Output formats" section: per-verb csv schema table; usage examples for each format.
- [ ] B4. Tests: parameterised test running each list + get verb against each of `json|ndjson|csv|yaml` with golden-output comparisons.

## P2

- [ ] C1. Cross-cutting test: `--all` (cli-pagination spec) under `--output ndjson` produces one valid JSON object per line across pages.
- [ ] C2. [HUMAN] Operator smoke: spreadsheet-paste a real csv export.
- [ ] C3. Spec close.
