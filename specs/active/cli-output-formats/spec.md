# Spec — cli-output-formats

| | |
|---|---|
| Status | DRAFT 080800ZMAY26 |
| Authored | 080800ZMAY26 |
| Owner | Bastion (J-3) |
| Carry-forward from | second-pass review of `citadel-cli` (2026-05-05): operators piping CLI output into `jq` / spreadsheets need ndjson + csv, not just `--output json | <empty>`. |

## Why

Every list/get verb today supports `--output json` (one JSON value per call) or default human-readable tabwriter. That covers programmatic consumption of single objects but is hostile for two real workflows:

1. **Streaming pipes** (`citadel-cli repo list -n org | jq` etc.) — the current `--output json` emits a single array per call, so cumulative consumption requires staging the whole list in memory. ndjson (one JSON object per line) is the canonical Unix shape; works with `jq -c`, `xargs -L 1`, `tail -f`-style consumption, and `wc -l` for cheap row counts.

2. **Spreadsheets / paste-into-sheets** — operators frequently want a quick CSV snapshot (`csvkit`, Excel import, Google Sheets paste). Adding `csv` is ~30 LOC per verb once the framework lands.

`yaml` is the third standard format and threads through the same `emitList`/`emitOne` machinery; sprinkle for free.

## In scope

- **`--output ndjson`** on every list verb (`repo list`, `agent list`, `token list`, `oauth clients list`, `namespace list`, `namespace members`, `namespace transfer list-pending`). One JSON object per line, no array wrapper, no trailing newline after the last line.
- **`--output csv`** on every list verb. Per-verb fixed column ordering (frozen as part of acceptance — changing it breaks scripts). Headers always emitted unless `--no-header` is also specified.
- **`--output yaml`** on every list + get verb. `gopkg.in/yaml.v3` for the encoder. Maps `time.Time` → ISO-8601 strings to keep the output round-trippable.
- **Documentation**: per-verb help text spells out the csv schema (column names + ordering); README gets a "Output formats" section.
- **`emitList` / `emitOne` plumbing**: the existing helpers in `cmd/output.go` grow case branches for the new formats. No new helpers per verb — the per-format logic stays centralised.

## Out of scope

- **Format auto-detection from terminal vs. pipe**: explicit `--output` only. `gh` does this and it confuses CI users; we don't.
- **`--output template` Go-template support**: `gh` ships it; useful but a separate spec — needs decisions about template syntax, partials, and the per-verb data model.
- **`--output yaml` for the existing `mcp` verbs**: those return JSON-RPC payloads with arbitrary nested types that yaml.v3 may render awkwardly. Stick with `--json` there.
- **Per-format headers via flags** (`--no-header`, `--separator`): nice-to-have; defer to v2 unless a concrete user reports the need.
- **Integration with the cli-pagination spec's `--all` mode**: covered there. ndjson under `--all` streams naturally; csv emits one header line at start of stream, then row lines per page.

## Decision log

| Q | Proposal | Status |
|---|----------|--------|
| Q1 | yaml dep: `gopkg.in/yaml.v3` vs. stdlib-only? | **Open** — yaml.v3 (BurntSushi/toml is already a dep; one more is fine). |
| Q2 | csv: emit per-verb hardcoded column order, or allow `--columns slug,path`? | **Open** — hardcoded at v1; `--columns` is a clean v2 add. |
| Q3 | csv `time.Time` rendering: ISO-8601 vs. unix epoch? | **Open** — ISO-8601; spreadsheet-friendly. |
| Q4 | ndjson trailing-newline policy: present (LF after every record incl. last) vs. omitted on last? | **Open** — present (LF after every record); RFC 8259 friendly + `wc -l` correct. |
| Q5 | yaml for `--output yaml` on get verbs: emit document separator (`---`) prefix or bare doc? | **Open** — bare doc; `---` breaks `yq r` on single-doc inputs. |

## Acceptance

- A1. `--output ndjson` works on every list verb. Output is one valid JSON value per line, no array wrapper.
- A2. `--output csv` works on every list verb. Header line emitted, column order frozen per verb.
- A3. `--output yaml` works on every list + get verb. Round-trippable through `yq`.
- A4. README "Output formats" section documents all formats + the per-verb csv schema table.
- A5. Tests cover all four formats (`json`, `ndjson`, `csv`, `yaml`) for at least one list verb + one get verb.
- A6. Q-table ratified.
