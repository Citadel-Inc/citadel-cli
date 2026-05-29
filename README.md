<h1 align="center">citadel-cli</h1>

<div align="center">

[![CI](https://github.com/Rethunk-Tech/citadel-cli/actions/workflows/ci.yml/badge.svg)](https://github.com/Rethunk-Tech/citadel-cli/actions/workflows/ci.yml)
[![Release](https://github.com/Rethunk-Tech/citadel-cli/actions/workflows/cli-release.yml/badge.svg)](https://github.com/Rethunk-Tech/citadel-cli/actions/workflows/cli-release.yml)
[![Go Reference](https://pkg.go.dev/badge/github.com/Rethunk-Tech/citadel-cli.svg)](https://pkg.go.dev/github.com/Rethunk-Tech/citadel-cli)
[![Go Report Card](https://goreportcard.com/badge/github.com/Rethunk-Tech/citadel-cli)](https://goreportcard.com/report/github.com/Rethunk-Tech/citadel-cli)
[![Go Version](https://img.shields.io/github/go-mod/go-version/Rethunk-Tech/citadel-cli)](go.mod)
[![Latest Release](https://img.shields.io/github/v/release/Rethunk-Tech/citadel-cli?include_prereleases&sort=semver)](https://github.com/Rethunk-Tech/citadel-cli/releases)
[![License: Proprietary](https://img.shields.io/badge/license-proprietary-red.svg)](LICENSE)

</div>

---

`citadel-cli` is the operator and developer interface for [Citadel](https://github.com/Rethunk-Tech/citadel). Citadel operators use it to administer namespaces, manage org members and permissions, query audit logs, and oversee the full platform lifecycle from the terminal. Developers reach for it to clone and push repositories, browse commits and blobs, manage deploy tokens and webhooks, file issues, track milestones, and interact with the Citadel Knowledge Graph — all without leaving the shell.

Beyond surface-level resource management, `citadel-cli` embeds a full MCP client, making it a first-class participant in agentic workflows. Agents and LLM-powered tools can authenticate, enumerate resources, and execute operations through the same structured command surface that humans use, with machine-readable output modes and a structured error envelope designed for reliable programmatic consumption.

## Highlights

- **Repository management** — clone, push, pull, commit browsing, tree/blob traversal, topics, repo insights, deploy tokens, and webhooks
- **Namespace and org operations** — create and configure namespaces, manage org members, handle transfer requests and invitations
- **Agent registration and token management** — register agents, issue and revoke agent tokens, manage token scopes
- **OAuth client registry** — register and administer OAuth clients and providers
- **Knowledge Graph traversal** — query and navigate the Citadel project graph
- **Audit log queries** — search and stream audit events and session logs
- **Embedded MCP client** — integrates Citadel into agentic and LLM-powered workflows as a first-class tool provider
- **Multiple output formats** — `json`, `yaml`, `ndjson`, `csv`, and `table` for scripting and human use
- **Shell completion** — generated completion scripts for bash, zsh, fish, and PowerShell

## Documentation

| Document | Description |
|---|---|
| [docs/cli.md](docs/cli.md) | Full command reference |
| [HUMANS.md](HUMANS.md) | Maintainer primer — install, auth, output formats, shell completion |
| [AGENTS.md](AGENTS.md) | Agent and LLM working conventions |
| [CONTRIBUTING.md](CONTRIBUTING.md) | Commits, branches, pre-push checklist |
| [CHANGELOG.md](CHANGELOG.md) | Release notes |

## License

Proprietary — see [LICENSE](LICENSE). Third-party notices in [NOTICE](NOTICE).
