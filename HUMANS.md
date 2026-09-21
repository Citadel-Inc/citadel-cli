# citadel-cli — maintainer primer

Human operators: read this first. LLM context: [AGENTS.md](AGENTS.md). Commits and pre-push: [CONTRIBUTING.md](CONTRIBUTING.md).

## What it is

Official CLI for [Citadel](https://src.land/) — repos, namespaces, agents, OAuth, audit, knowledge graph, and an embedded MCP client.

## Install

Prefer a tagged binary from [GitHub Releases](https://github.com/Citadel-Inc/citadel-cli/releases) (linux-amd64, linux-arm64, darwin-arm64, windows-amd64). Put it on `PATH` as `citadel-cli`.

With Go 1.27.1 or newer:

```bash
go install github.com/Citadel-Inc/citadel-cli@latest
```

That lands in `~/go/bin`. `@latest` is the newest `v*` tag whose `go.mod` path is `github.com/Citadel-Inc/citadel-cli` (v0.1.0 is the old `Rethunk-Tech` module and will not install from this path).

Prerequisites for local work: Go 1.27.1+, `golangci-lint` (for `make verify`).

## Usage

```bash
citadel-cli auth login         # browser OAuth → agent token
citadel-cli auth status
citadel-cli repo list
citadel-cli repo insights org/repo
```

Local build:

```sh
go build -o ./citadel-cli .
./citadel-cli --help
make verify
```

Repo targeting: `-R namespace/slug`, or `CITADEL_REPO`, or CWD `origin` on a Citadel git host (`--no-cwd-repo` disables inference). Full command surface: [docs/cli.md](docs/cli.md).

## Configuration

| Variable | Purpose |
| --- | --- |
| `CITADEL_SERVER` | API base URL (overrides config) |
| `CITADEL_ACCESS_TOKEN` | Short-lived bearer for CI/scripts |
| `CITADEL_AGENT_TOKEN` | Bearer for `mcp` subcommands only |
| `CITADEL_REPO` | Default repo slug |
| `CITADEL_GIT_HOSTS` | Extra git hosts for CWD inference |
| `CITADEL_PAGER` / `GIT_PAGER` / `PAGER` | Pager chain (`''` or `--no-pager` disables) |
| `CITADEL_NO_COMPLETION_CACHE` | Skip completion disk cache |

Config file: `~/.config/citadel/config.toml` (`server_url`, tokens mode 0600). `auth login` stores an agent token; `auth set-token` accepts a Supabase JWT and upgrades on next launch when the API is reachable.

## Verify

```bash
make verify   # vet + race tests + golangci-lint
```

Live integration tests self-skip unless env gates are set (`CITADEL_TEST_OAUTH_JWT`, `CITADEL_TEST_*_LIVE=1`, etc.).

## Uninstall

```bash
rm "$(command -v citadel-cli)"
rm -rf ~/.config/citadel
```

Completion caches under `$XDG_CACHE_HOME/citadel-cli/` may be removed manually.
