# Plan — cli-pagination

## Cursor encoding

Opaque base64-url over `(created_at int64 unix-nanos, id uuid)`:

```go
type Cursor struct {
    CreatedAt int64
    ID        uuid.UUID
}

func (c Cursor) Encode() string { /* binary.BigEndian + base64.RawURLEncoding */ }
func DecodeCursor(s string) (Cursor, error) { /* inverse */ }
```

Server applies `WHERE (created_at, id) > (cursor.created_at, cursor.id)` ordered the same way as today (`order by created_at desc`). Tuple ordering avoids the offset-skip bug on concurrent inserts.

## Server response shape

```jsonc
// GET /api/namespaces/myorg/repos?limit=50&cursor=<opaque>
{
  "repos": [ ... 50 rows ... ],
  "next_cursor": "AAAA..."  // null/empty when this was the last page
}
```

Same shape pattern for every endpoint. Existing clients that read `repos[]` and ignore `next_cursor` keep working.

## CLI flag plumbing

Add helpers to `cmd/output.go`:

```go
func addPaginationFlags(cmds ...*cobra.Command) {
    for _, c := range cmds {
        c.Flags().Int("limit", 0, "Max rows per page (server caps at 200)")
        c.Flags().String("cursor", "", "Opaque cursor from a previous page")
        c.Flags().Bool("all", false, "Auto-paginate until exhausted (overrides --cursor)")
    }
}

func paginationFlags(cmd *cobra.Command) (limit int, cursor string, all bool)
```

Each list-verb runE calls a new shared `walkPages(ctx, fetchPage, render)` helper that handles the `--all` loop:

```go
func walkPages[T any](
    ctx context.Context,
    fetchPage func(cursor string) (rows []T, next string, err error),
    render func(rows []T) error,
    all bool,
    cursor string,
) error
```

`render` flushes one tabwriter block per page when called from default mode; under `--output ndjson` it emits one JSON object per row directly.

## Streaming guarantees

`--all` + `--output ndjson` is the canonical "give me every row, one JSON object per line" mode:

```
$ citadel-cli repo list -n big-org --all --output ndjson | jq -r .slug
r1
r2
...
r723
```

`--all` + default human output flushes the tabwriter after each page; columns may misalign across pages because each block is sized independently. Document this trade-off — strict alignment requires buffering the whole stream, which defeats the point.

## Estimated delta

| Component | LOC (rough) |
|-----------|-------------|
| Server cursor codec + 6 endpoint patches | 200 |
| Server tests | 80 |
| CLI `addPaginationFlags` / `walkPages` helpers | 60 |
| CLI per-verb call-site updates (×7 list verbs) | 80 |
| CLI tests (3-page fixture) | 80 |
| Docs (README + HUMANS) | 20 |
| **Total** | **~520** |

## Risks

- **Cursor stability across schema migrations**: if `created_at` precision ever changes (e.g., truncation), old cursors silently misalign. Mitigation: encode the column unit explicitly into the codec version byte; bump the byte on schema changes.
- **`--all` runaway**: a user with 50,000 rows running `--all` against a slow link is going to hate themselves. Mitigation: print a "fetching N pages..." progress line under TTY, and document the cost in the help text.
- **Mixed-pagination clients**: a future third-party client using the same endpoints needs to know about `next_cursor`. Document in the daemon spec's discovery metadata follow-on.
