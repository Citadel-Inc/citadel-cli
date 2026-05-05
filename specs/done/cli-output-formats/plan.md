# Plan — cli-output-formats

## Plumbing

`cmd/output.go` grows two new case branches in `emitList` + one in `emitOne`:

```go
func emitList[T any](output string, rows []T, emptyMsg string,
    table func(w *tabwriter.Writer, rows []T)) error {
    switch output {
    case "json":
        return emitJSON(rows)
    case "ndjson":
        return emitNDJSON(rows)
    case "csv":
        return emitCSV(rows)
    case "yaml":
        return emitYAML(rows)
    }
    // human (default)
    if len(rows) == 0 { ... }
    w := newTabWriter(); table(w, rows); return w.Flush()
}
```

ndjson and yaml are content-type-only; csv needs per-row column projection.

## CSV projection

Add a small interface:

```go
type csvRow interface {
    CSVHeader() []string
    CSVRecord() []string
}
```

Each row type (`repoRow`, `agentRow`, `nsRow`, `nsMemberRow`, etc.) gets a method pair. `emitCSV` uses encoding/csv against stdout, calling `CSVHeader()` once + `CSVRecord()` per row.

Time-shaped columns format via `time.Time.Format(time.RFC3339)` (Q3).

Per-verb column order — frozen as part of the spec contract — example for `repoRow`:

```
slug,path,visibility,branch,description,created
r1,org/r1,private,main,"a repo",2026-01-01T00:00:00Z
```

## ndjson encoder

```go
func emitNDJSON[T any](rows []T) error {
    enc := json.NewEncoder(os.Stdout)
    enc.SetEscapeHTML(false)
    for _, r := range rows {
        if err := enc.Encode(r); err != nil { return err }
    }
    return nil
}
```

`json.Encoder.Encode` already emits trailing LF per record (Q4 ratification: keep).

## yaml encoder

Use `yaml.NewEncoder(os.Stdout)`. Single document, no `---` prefix on the first document (Q5). For lists: each row encoded as its own document is wrong shape; instead emit a single yaml sequence — `yaml.Marshal(rows)` does the right thing.

## Per-verb test matrix

A single parameterised test in `cmd/output_formats_test.go`:

```go
for _, format := range []string{"json", "ndjson", "csv", "yaml"} {
    t.Run("repo-list-"+format, ...)
    t.Run("agent-get-"+format, ...) // get verbs only json+yaml; csv+ndjson are list-shaped
}
```

Golden outputs live under `cmd/testdata/output/<verb>-<format>.golden`.

## Estimated delta

| Component | LOC (rough) |
|-----------|-------------|
| `cmd/output.go` format branches | 60 |
| Per-row `CSVHeader/CSVRecord` methods (×8 row types) | 100 |
| ndjson + yaml encoders | 30 |
| Tests (parameterised + golden files) | 150 |
| README + HUMANS docs | 50 |
| **Total** | **~390** |

## Risks

- **Frozen csv schema**: column ordering becomes a compatibility surface. Adding columns is fine (append to end); removing or reordering breaks scripts. Document as part of the per-verb contract.
- **Map fields in yaml**: `repoRow.Description` is a string, but if any future row type has a `map[string]string` (e.g., metadata labels), yaml.v3 emits them in stable-key order — fine, just call out so reviewers don't accidentally introduce non-stable shapes.
- **Time-zone surprises**: ISO-8601 emits the local zone unless `.UTC()` is called first. Always `.UTC()` before `.Format(time.RFC3339)`.
