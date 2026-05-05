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

## Documentation

- [docs/cli.md](docs/cli.md) — full command reference
- [AGENTS.md](AGENTS.md) — LLM / agent conventions
- [CONTRIBUTING.md](CONTRIBUTING.md) — commits, branches, pre-push checklist
- [LICENSE](LICENSE) — proprietary
- [NOTICE](NOTICE) — third-party notices
