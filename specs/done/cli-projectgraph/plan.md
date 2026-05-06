# Plan — cli-projectgraph

## ORIENT

- **Server:** `internal/api/projectgraphapi/handler.go` — `Routes()` returns `http.HandlerFunc` dispatching on method + path suffix; see `projectgraphapi.Handler` struct.
- **Companion spec:** `citadel/specs/active/go-projectgraph/spec.md` for domain semantics (edge kinds, ingest).
- **CLI patterns:** `internal/apiclient.Client` — paths appended to configured API base (`ResolveServerURL`); paths **include** `/api/...` explicitly.

## RECON (completed baseline)

| Topic | Finding |
|-------|---------|
| Prefix | `/api/projectgraph/` + tail built from slug + suffix (`pin-chain`, `walk`, `neighbors`, …). |
| Slug in path | Multi-segment paths are URL-encoded per segment then joined — HTTP stack decodes `%2F` into `/` segments server-side; tests match decoded paths (`/api/projectgraph/org/repo/...`). |
| Walk | **`kind` query required** — `400 kind_required` if missing. **`max_depth`** optional int. |
| Pin chain | **`repo` namespace only** — else `400 repo_namespace_required`. |
| Neighbors | Query: `kind`, optional `ns`, `direction`, `include_deleted=true`. |
| POST edges | Body **`postEdgeBody`** — `source` must be `"manual"` or server rejects (`manual_source_only`). RBAC: **`PermProjectgraphManage`** on **from** namespace. |
| RBAC read | Denial often **`404 not_found`** (opaque). |

## Appendix — status rollup / drilldown / reindex (RECON)

Captured at implementation time from daemon wiring:

- **`GET …/{slug}/status-rollup`** — v1 CLI performs an unfiltered GET; extend with explicit query flags when the daemon publishes a stable filter contract in OpenAPI/spec.
- **`GET …/{slug}/status-rollup/drilldown`** — same gate as rollup; plain GET for v1.
- **`POST …/{slug}/reindex`** — JSON body typically empty; CLI treats like other destructive-adjacent verbs (**typed confirm** or **`--yes`**).

**Operator hint:** stderr may mention that **404** can reflect missing **`projectgraph:read`** scope, not necessarily a wrong slug.

## Implementation sketch

- **`cmd/project.go`:** `projectCmd` + subcommands; shared **`projectgraphPath(slug, suffix)`** URL-encodes each slash-separated segment.
- **Tests:** `cmd/handler_test.go` — route tables per verb; multi-segment slug pin-chain; read vs write 404 coverage; mutations use **`--yes`** in tests.
- **Docs:** `docs/cli.md` section + cross-link from HUMANS if needed later.

## Risks

- **Payload size:** walk/rollup JSON may be large — document **`--output json`** + jq for inspection (client uses normal `Get`, acceptable for v1).
