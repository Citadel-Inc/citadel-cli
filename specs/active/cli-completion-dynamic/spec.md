# Spec — cli-completion-dynamic

| | |
|---|---|
| Status | IN_PROGRESS 050921ZMAY26 — Bastion (J-3) claims execution |
| Authored | 050826ZMAY26 |
| Owner | Bastion (J-3) |
| Carry-forward from | 2026-05-05 enhancement sweep: `citadel-cli completion` ships shells but every positional argument (repo slug, namespace, agent id, oauth client id) has zero `ValidArgsFunction` coverage. Tab completion today returns nothing for any resource identifier. |

## Why

Operators use the CLI interactively against namespaces with dozens of repos / agents. Every `repo get <slug>` or `agent rotate-token <id>` is a manual paste from a prior `repo list`. cobra's `ValidArgsFunction` API supports server-driven dynamic completion — we just don't wire it. This spec adds dynamic completion for the resource identifiers below, with auth-aware fallback (no token → no completion, never a prompt).

## In scope

- **Resources covered**:
  - Namespace slug — used as `-n / --namespace` value or first positional on `namespace get/members/transfer ...`. Source: `GET /api/namespaces`.
  - Repo slug — second positional on `repo get/delete/...`, scoped to the resolved namespace. Source: `GET /api/namespaces/{ns}/repos`.
  - Agent id — positional on `agent get/delete/rotate-token`. Source: `GET /api/agents`.
  - OAuth client id — positional on `oauth clients get/delete`. Source: `GET /api/oauth/clients`.
  - Token id — positional on `token revoke`. Source: `GET /api/agent-tokens`.
- **`ValidArgsFunction` per verb**: each handler registers a function that opens an apiclient (with the resolved `--server`), queries the listing endpoint, and returns slugs/ids matching the partial. Auth errors / network errors return `cobra.ShellCompDirectiveError` silently — never block the prompt.
- **Caching**: short-lived disk cache at `${XDG_CACHE_HOME:-~/.cache}/citadel-cli/completion/<resource>.json` keyed by `(server, resource)`, TTL 60 s. Refresh on TTL expiry; on any cache-write error fall back to in-memory only.
- **Cache invalidation**: any write verb that mutates a resource (create/delete/rotate) deletes the matching cache file before exit. Best-effort; never fails the verb.
- **Pagination interplay**: when [cli-pagination](../cli-pagination/spec.md) lands, completion always passes `--limit 200` (server cap) and stops — no `--all` walking inside completion to keep latency bounded.
- **Output flag completion**: register a static completion list (`json`, `yaml`, `ndjson`, `csv`, `table`) on every `--output` flag, gated by [cli-output-formats](../cli-output-formats/spec.md) shipping. Static — no server round trip.

## Out of scope

- **Per-shell custom completion scripts**: stick with cobra's built-in `__complete` mechanism; no hand-rolled bash/zsh completion.
- **Fuzzy matching**: server returns canonical prefixes only. cobra/shell handles substring narrowing locally.
- **Org-scoped slugs**: `org/repo` parsing is out of scope for v1; complete the bare repo slug only and rely on `-n` for namespace.
- **MCP `tools/call` argument completion**: tool-name dynamic listing is interesting but separate. Defer.
- **Completion for `auth set-token` / OAuth login flows**: those take JWT/secret values, not slugs.

## Decision log

| Q | Proposal | Status |
|---|----------|--------|
| Q1 | Cache TTL 60 s vs 30 s vs no cache? | **Ratified 050919ZMAY26** — 60 s; trades freshness for ≤ one round trip per minute per resource. |
| Q2 | On unauthenticated client, complete with `[]` (silent) or surface `auth login required` as a hint message? | **Ratified 050919ZMAY26** — silent + `ShellCompDirectiveError`; cobra does not render hints across all shells consistently. |
| Q3 | Cache file format: JSON array vs newline-delimited slugs? | **Ratified 050919ZMAY26** — JSON envelope with `values` array; future-proof for adding metadata. |
| Q4 | Invalidation on write verbs: blocking delete vs fire-and-forget? | **Ratified 050919ZMAY26** — fire-and-forget goroutine on cmd.PostRun; never delays user exit. |
| Q5 | Completion for the persistent `--server` flag itself? | **Ratified 050919ZMAY26** — defer; no obvious source set without a config-stored profile list (separate spec). |

## Acceptance

- A1. `citadel-cli repo get <TAB>` (with valid auth) returns repo slugs from the resolved namespace within ≤ 200 ms (cache hit) or ≤ server RTT + 50 ms (cache miss).
- A2. `citadel-cli namespace members <TAB>` returns namespace slugs.
- A3. `citadel-cli agent rotate-token <TAB>` returns agent ids.
- A4. `citadel-cli oauth clients delete <TAB>` returns client ids.
- A5. Unauthenticated invocation returns no candidates (no error, no prompt for credentials).
- A6. Cache file written under `$XDG_CACHE_HOME/citadel-cli/completion/`. TTL 60 s; cache miss triggers a single API call.
- A7. Mutation verbs delete the matching cache file before returning.
- A8. Q-table ratified.
