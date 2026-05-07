# citadel-cli — maintainer primer

If you are an engineer onboarding to `citadel-cli`, read this file first. For LLM / agent context see [AGENTS.md](AGENTS.md) (`CLAUDE.md` is a symlink). For commits, branches, and pre-push verification see [CONTRIBUTING.md](CONTRIBUTING.md).

## What it is

`citadel-cli` is the official command-line client for [Citadel](https://github.com/Rethunk-Tech/citadel) — Rethunk's substrate for repos, deploy tokens, namespace issues, namespaces, agents, OAuth clients, and the knowledge graph. It also embeds an MCP client for agentic-workflow integrations.

## Getting started

### Install

```bash
go install github.com/Rethunk-Tech/citadel-cli@latest
```

This installs to `~/go/bin/citadel-cli`; ensure `~/go/bin` is on your `PATH`.

Pre-built release binaries (linux-amd64, linux-arm64, darwin-arm64) are published to GitHub Releases on every `v*` tag — see <https://github.com/Rethunk-Tech/citadel-cli/releases>.

### First run

```bash
citadel-cli auth login        # browser OAuth (PKCE) → long-lived agent token
citadel-cli auth status       # confirm authentication
citadel-cli repo list         # query the API
```

### Local development

Prerequisites: Go 1.25+, `golangci-lint` (for `make verify`).

```sh
go build -o ./citadel-cli .          # local build
./citadel-cli --help                  # explore subcommands
./citadel-cli auth login              # browser OAuth → agent token (see `auth set-token` for headless)
make verify                           # vet + race tests + golangci-lint
```

## Repository layout

```
main.go                          Cobra entrypoint
cmd/                             Subcommand implementations
  agent.go auth.go confirm.go kg.go mcp.go namespace.go
  oauth_clients.go output.go repo.go token.go
  doc.go                         Package comment
  *_test.go                      Unit + integration tests (live tests env-gated)
internal/clicfg/                 Config (XDG_CONFIG_HOME / ~/.config/citadel/config.toml; doc.go for package comment)
internal/completion/             Shell-completion cache + API-backed candidate lists
internal/mcpclient/              HTTP MCP client (cobra mcp subcommands)
docs/cli.md                      Full command reference
specs/active/, specs/done/, specs/parked/   SDD specs (+ parked = not pursued)
.github/workflows/               ci.yml, cli-release.yml
Makefile                         build / build-all / test / vet / lint / verify
```

## Day-to-day

| Need | Run |
|------|-----|
| Local build | `make build` (binary at `./citadel-cli`) |
| Cross-compile release artefacts | `make build-all` (3 platforms into `dist/`) |
| Run tests | `make test` |
| Lint | `make lint` (golangci-lint) |
| Pre-push gate | `make verify` |
| Cut a release | tag `vX.Y.Z`; `cli-release.yml` builds + publishes to GH Releases |

Live integration tests self-skip unless their env gates are set. Examples: `oauth_clients_live_test.go` uses `CITADEL_TEST_OAUTH_JWT`; repository list pagination uses `CITADEL_TEST_PAGINATION_LIVE=1`; audit list round-trip uses `CITADEL_TEST_AUDIT_LIVE=1` plus `CITADEL_TEST_OAUTH_JWT`. Full browser OAuth login coverage lives in `auth_live_test.go`, gated on `CITADEL_TEST_OAUTH_FULL=1` plus either `CITADEL_TEST_OAUTH_STORAGE_STATE=/abs/path/to/playwright-storage-state.json` or `CITADEL_TEST_OAUTH_REFRESH_TOKEN=<citadel refresh token for client_id=citadel-cli>`, and a local Playwright Chromium install.

## Audit log access

Use `citadel-cli audit list` and `citadel-cli audit show <id>` against the Citadel HTTP API (`/api/audit/events`). Filters include `--since` / `--until` (Go duration or RFC3339), `--kind` (dot-separated glob: `*` is one segment, `**` matches a multi-segment suffix), `--namespace`, and `--actor` (UUID or user slug). Output follows the standard `--output` formats plus cursor pagination (`--limit`, `--cursor`, `--all`).

Operator-only fields such as client IP are omitted for non-operator callers; the server decides redaction per RBAC.

**Kinds commonly surfaced for investigations** (non-exhaustive; your deployment may emit more): `repo.deleted`, `org.deleted`, `namespace.hard_purged` (and other cleanup actions), `agent.created`, OAuth flows under `oauth.*`, org transfer and membership actions, `mcp.tools.call` for agent tool usage. Use `audit list --kind 'prefix.**'` to explore a family.

## Configuration

- **Server URL:** `~/.config/citadel/config.toml` (key: `server_url`) or `CITADEL_SERVER` env var.
- **Auth tokens:** `~/.config/citadel/config.toml` (mode 0600); `citadel-cli auth login` stores a Citadel-issued **agent token** (plus agent id/name). For headless hosts, `citadel-cli auth set-token` accepts a Supabase JWT; on the next CLI launch the client **eagerly upgrades** a JWT-only file to an agent token when the API is reachable.
- **Override access token:** `CITADEL_ACCESS_TOKEN` env var (1-hour pinned expiry; for CI / scripting).

### Repo context

Commands that target a single repository accept `-R <namespace>/<slug>` (same meaning as `gh -R`). If you omit it, the CLI uses the `CITADEL_REPO` environment variable, then (unless `--no-cwd-repo` is set) infers the repo from `git remote get-url origin` when that remote uses a [Citadel git host](https://src.land) (for example `src.land` or `git.src.land`). Comma-separated `CITADEL_GIT_HOSTS` extends the default host list for self‑hosted deployments.

`--no-cwd-repo` disables CWD inference so scripts never pick up a surprise repo from the current directory; combine it with `-R` or `CITADEL_REPO` when you need an explicit path.

```bash
cd ~/code/myorg/myrepo          # citadel clone
citadel-cli repo get            # inferred when origin is a Citadel remote
citadel-cli repo get -R other/ns   # explicit repo
```

Issue verbs (`citadel-cli issue ...`) reuse the same `-R` / `CITADEL_REPO` / CWD-origin rules, but interpret the target as a **namespace path**. For repository issues, omission can still infer `org/repo` from the current checkout; for org-level or deeper non-repo namespaces, pass `-R` explicitly.

Full reference: [docs/cli.md](docs/cli.md).

### List pagination

Every list verb (`repo list`, `repo deploy-token list`, `namespace deploy-token list`, `agent list`, `token list`, `oauth clients list`, `namespace list`, `namespace members`, `namespace transfer list-pending`) accepts **`--limit`** (default 50, maximum 200), **`--cursor`** (opaque token from the prior response’s `next_cursor` field), and **`--all`** (walk pages serially until exhausted). In human/table mode, when more rows exist the CLI prints a trailing hint: `(use --cursor … for more, or --all to fetch everything)`.

**`--output json`** returns a single JSON array for one server round-trip only; **`--output ndjson`** emits one JSON object per row and is the supported mode for **`--all`** when you want a machine-readable stream without buffering the entire result set. Passing **`--all` with `--output json`** is rejected with an error directing you to `ndjson`.

**`--watch` / `-w`** on the same list verbs opens the Server-Sent Events stream (`Accept: text/event-stream`) on the corresponding REST path: rows arrive as `init` / `add` / `update` / `remove` events until you interrupt (Ctrl-C). Use **`--output ndjson --watch`** for one JSON object per event (`type`, `ts`, `payload`) suitable for piping to `jq`. **`--output json --watch`** is rejected (same hint as pagination). Human/table mode redraws on a color-capable TTY when color is enabled; otherwise it prints snapshot blocks and short `+`/`-`/`~` delta lines.

Malformed **`--cursor`** values (before any HTTP call) produce a clear `invalid --cursor` error; valid cursors that point past the end yield an empty success (exit 0).

### Shell completion

Install scripts with `citadel-cli completion bash|zsh|fish|powershell` (see `citadel-cli completion --help`). Dynamic tab completion for resource arguments — repo slugs (scoped via `-R`, `CITADEL_REPO`, or CWD inference), org namespace slugs on `namespace get|members|delete|transfer initiate`, agent names on `agent get|delete|rotate-token`, OAuth client resource UUIDs on `oauth clients show|revoke`, token UUIDs on `token revoke`, deploy-token UUIDs on `repo deploy-token revoke` / `namespace deploy-token revoke`, SSH key UUIDs on `ssh-key delete` — issues authenticated list calls against the Citadel API. With no access token, completion yields no candidates and never prompts for credentials. JSON cache files live under `$XDG_CACHE_HOME/citadel-cli/completion/<server-host>/` with a 60-second TTL. Set **`CITADEL_NO_COMPLETION_CACHE=1`** to skip disk cache reads and writes (in-memory only; useful for debugging).

## Structured errors (`--output json` / `yaml` / `ndjson`)

On failure, machine-readable output modes use a single top-level object with an `error` field (so success payloads and errors never share the same keys). Only `kind` and the process exit code are stable contracts; human-readable `message` text may change between releases.

### Envelope shape (JSON)

```jsonc
{
  "error": {
    "kind": "rate_limited",
    "message": "rate limit exceeded — slow down or wait a few minutes before retrying",
    "http_status": 429,
    "retry_after_seconds": 60,
    "hint": "https://status.src.land",
    "details": {}
  }
}
```

`http_status`, `retry_after_seconds`, `hint`, and `details` are omitted when they do not apply. `retry_after_seconds` is taken from the HTTP `Retry-After` header when present (integer seconds or HTTP-date). `details` holds a small curated subset of JSON from the server body on validation-style failures, never the full raw payload (use `--debug-http` for wire dumps).

### `kind` values (v1)

| `kind` | Typical meaning |
|--------|-----------------|
| `auth_required` | Missing token, HTTP 401, or MCP unauthorized |
| `mfa_required` | HTTP 412 (recent MFA / verification) |
| `forbidden` | HTTP 403 |
| `not_found` | HTTP 404, or MCP “method not found” |
| `conflict` | HTTP 409 |
| `rate_limited` | HTTP 429 |
| `validation` | HTTP 400 with a JSON body, or MCP parameter / parse errors |
| `server_unavailable` | HTTP 502 / 503 / 504, or connection cut mid-stream |
| `server_error` | Other HTTP 5xx |
| `timeout` | Context deadline exceeded |
| `network` | DNS or dial failures |
| `dry_run` | Reserved for dry-run sentinel paths |
| `internal` | Catch-all, including unmapped errors |

### Exit code map (v1)

| Code | Meaning |
|------|---------|
| 0 | Success |
| 1 | `internal` and other generic failures |
| 2 | `validation`, `dry_run`, and MCP `tools/call` `isError` (`ErrToolCallFailed`) |
| 3 | `auth_required`, `mfa_required`, `forbidden` |
| 4 | `not_found` |
| 5 | `conflict` |
| 6 | `rate_limited` |
| 7 | `server_unavailable`, `server_error`, `network`, `timeout` |

Human (non–machine-readable) output still prints `Error: <message>` on stderr; the same `kind` drives the exit code.

## Documentation

- [docs/cli.md](docs/cli.md) — full command reference
- [AGENTS.md](AGENTS.md) — LLM / agent conventions
- [CONTRIBUTING.md](CONTRIBUTING.md) — commits, branches, pre-push checklist
- [LICENSE](LICENSE) — proprietary
- [NOTICE](NOTICE) — third-party notices
