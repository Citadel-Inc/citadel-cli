# Plan — cli-projectgraph

## ORIENT

- Server package: `github.com/Rethunk-Tech/citadel/internal/api/projectgraphapi` — subtree mounted at `/api/projectgraph/` (`cmd/citadel/main.go`).
- Slug path uses a **manual dispatcher** because `{slug}` can contain `/` (multi-segment); CLI must use `url.PathEscape` per segment or join path safely — mirror how integration tests build URLs.
- CLI patterns: `internal/apiclient.Client`, `newAPIClient`, output helpers from `cli-output-formats`, repo-style `-R` **not** applicable (project slug is namespace-shaped, not repo pair).

## RECON (routes)

From `projectgraphapi.Handler.Routes()`:

| Method | Path suffix | Handler |
|--------|-------------|---------|
| GET | `{slug}/pin-chain` | pin chain rows |
| GET | `{slug}/walk` | bounded walk |
| GET | `{slug}/neighbors` | neighbors |
| GET | `{slug}/status-rollup` | rollup |
| GET | `{slug}/status-rollup/drilldown` | drilldown |
| POST | `{slug}/edges` | create edge |
| DELETE | `{slug}/edges/{edge_id}` | delete |
| POST | `{slug}/edges/{edge_id}/restore` | restore |
| POST | `{slug}/reindex` | reindex ingest |
| POST | `admin/recovery-scan` | admin recovery |

## Implementation sketch

- New `cmd/project.go` with `ProjectCmd` and subcommands; positional `<slug>` as first arg after subcommand (e.g. `project pin-chain Rethunk-Tech/Bastion`).
- Shared helper `projectAPIPath(slug, suffix string) string` building `/api/projectgraph/` + escaped segments.
- Tests: extend `handler_test.go` pattern from `kg` / `audit` with mux stubs for each verb.

## Risks

- **Slug ambiguity**: users may pass single-segment vs full path — document examples in `--help`.
- **Large JSON**: walk/pin-chain payloads may be huge; recommend `--output json` + jq for scripting.
