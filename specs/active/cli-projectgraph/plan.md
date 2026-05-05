# Plan — cli-projectgraph

## ORIENT

- **Server:** `internal/api/projectgraphapi/handler.go` — `Routes()` returns `http.HandlerFunc` dispatching on method + path suffix; see `projectgraphapi.Handler` struct.
- **Companion spec:** `citadel/specs/active/go-projectgraph/spec.md` for domain semantics (edge kinds, ingest).
- **CLI patterns:** `internal/apiclient.Client` — paths are appended to configured API base (`ResolveServerURL`); no automatic `/api` prefix — follow existing commands (`GET "/orgs"`, etc.) → paths **include** `/api/...` where required.

## RECON (completed baseline)

| Topic | Finding |
|-------|---------|
| Prefix | `/api/projectgraph/` + tail built from slug + suffix (`pin-chain`, `walk`, `neighbors`, …). |
| Slug in path | `r.PathValue("slug")` set by dispatcher after stripping suffix; **walk/neighbors** also accept `?ns=` overriding target namespace. |
| Walk | **`kind` query required** — `400 kind_required` if missing. **`max_depth`** optional int. |
| Pin chain | **`repo` namespace only** — else `400 repo_namespace_required`. |
| Neighbors | Query: `kind`, `direction`, `include_deleted=true`. |
| POST edges | Body struct **`postEdgeBody`** in handler.go — `source` must be `"manual"` or server rejects (`manual_source_only`). RBAC: **`PermProjectgraphManage`** on **from** namespace (`403 forbidden` vs opaque `404` depending on branch — tests lock actual behaviour). |
| RBAC read | `requireProjectgraphRead` → resolver `PermProjectgraphRead`; denial **`404 not_found`** (opaque). |

**P0 implementation task:** Read `handleStatusRollup`, `handleStatusRollupDrilldown`, `handleReindex`, `handleRecoveryScan` for exact query/body contracts and paste into this plan appendix when implementing.

## Implementation sketch

- **`cmd/project.go`:** `projectCmd` + subcommands; shared **`buildProjectgraphURL(slug, suffix string)`** that URL-encodes each segment of `slug` when joining (never split on `/` incorrectly).
- **Tests:** `cmd/handler_test.go` style — map route → JSON fixture; include **`429 rate_limited`** stub optional.

## Risks

- **Payload size:** walk/rollup JSON may be MB-scale — document memory/streaming expectations (客户端 uses normal `Get`, acceptable for v1).
- **Permission UX:** users confuse **404** with “wrong slug” — stderr hint when stderr allowed: “may be permissions — verify projectgraph:read”.
