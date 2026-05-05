# citadel-cli — maintainer primer

If you are an engineer onboarding to `citadel-cli`, read this file first. For LLM / agent context see [AGENTS.md](AGENTS.md) (`CLAUDE.md` is a symlink). For commits, branches, and pre-push verification see [CONTRIBUTING.md](CONTRIBUTING.md).

## What it is

`citadel-cli` is the official command-line client for [Citadel](https://github.com/Rethunk-Tech/citadel) — Rethunk's substrate for repos, namespaces, agents, OAuth clients, and the knowledge graph. It also embeds an MCP client for agentic-workflow integrations.

## Quick start

Prerequisites: Go 1.25+, `golangci-lint` (for `make verify`).

```sh
go build -o ./citadel-cli .          # local build
./citadel-cli --help                  # explore subcommands
./citadel-cli auth login              # OAuth login against configured server
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
specs/active/, specs/done/       SDD specs
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

Live integration tests (e.g. `oauth_clients_live_test.go`) self-skip without `CITADEL_TEST_OAUTH_JWT` set.

## Configuration

- **Server URL:** `~/.config/citadel/config.toml` (key: `server_url`) or `CITADEL_SERVER` env var.
- **Auth tokens:** `~/.config/citadel/config.toml` (mode 0600); written by `citadel-cli auth login`.
- **Override access token:** `CITADEL_ACCESS_TOKEN` env var (1-hour pinned expiry; for CI / scripting).
- **Repo context:** `CITADEL_REPO=<namespace>/<slug>` selects a repo without `-R`. Only the **`origin`** remote is used for CWD inference; if `origin` is not a Citadel host, pass `-R` explicitly (see README “Repo context”). For private git endpoints, set comma-separated **`CITADEL_GIT_HOSTS`**.
- **Shell completion:** install scripts with `citadel-cli completion bash|zsh|fish|powershell` (see `citadel-cli completion --help`). Dynamic tab completion for resource arguments — repo slugs (scoped via `-R`, `CITADEL_REPO`, or CWD inference), org namespace slugs on `namespace get|members|delete|transfer initiate`, agent names on `agent get|delete|rotate-token`, OAuth client resource UUIDs on `oauth clients show|revoke`, token UUIDs on `token revoke` — issues authenticated list calls against the Citadel API. With no access token, completion yields no candidates and never prompts for credentials. JSON cache files live under `$XDG_CACHE_HOME/citadel-cli/completion/<server-host>/` with a 60-second TTL. Set **`CITADEL_NO_COMPLETION_CACHE=1`** to skip disk cache reads and writes (in-memory only; useful for debugging).

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
