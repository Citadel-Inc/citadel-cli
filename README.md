<h1 align="center">citadel-cli</h1>

<div align="center">

[![CI](https://github.com/Citadel-Inc/citadel-cli/actions/workflows/ci.yml/badge.svg)](https://github.com/Citadel-Inc/citadel-cli/actions/workflows/ci.yml)
[![Release](https://github.com/Citadel-Inc/citadel-cli/actions/workflows/cli-release.yml/badge.svg)](https://github.com/Citadel-Inc/citadel-cli/actions/workflows/cli-release.yml)
[![Go Reference](https://pkg.go.dev/badge/github.com/Citadel-Inc/citadel-cli.svg)](https://pkg.go.dev/github.com/Citadel-Inc/citadel-cli)
[![Go Report Card](https://goreportcard.com/badge/github.com/Citadel-Inc/citadel-cli)](https://goreportcard.com/report/github.com/Citadel-Inc/citadel-cli)
[![Go Version](https://img.shields.io/github/go-mod/go-version/Citadel-Inc/citadel-cli)](go.mod)
[![Latest Release](https://img.shields.io/github/v/release/Citadel-Inc/citadel-cli?include_prereleases&sort=semver)](https://github.com/Citadel-Inc/citadel-cli/releases)
[![License: Proprietary](https://img.shields.io/badge/license-proprietary-red.svg)](LICENSE)

</div>

---

`citadel-cli` is the terminal interface for [Citadel](https://src.land/): namespaces, repos, agents, OAuth, audit, and the knowledge graph. Operators administer the platform; developers clone, push, and script against the API without leaving the shell.

The CLI embeds an MCP client and structured error envelopes for agentic workflows — same commands humans use, with machine-readable output modes.

## Quick start

```bash
go install github.com/Citadel-Inc/citadel-cli@latest
```

Install, auth, and local development: [HUMANS.md](HUMANS.md).

## Highlights

- **Repo and namespace lifecycle** — clone, push, issues, deploy tokens, webhooks, org members
- **Agents, OAuth, and tokens** — registration, scopes, provider admin
- **Knowledge Graph** — traverse the Citadel project graph and repo insights
- **Audit logs** — search and stream events and session logs
- **Embedded MCP client** — first-class agentic and LLM workflow integration
- **Scriptable output** — json, yaml, ndjson, csv, table, and shell completion

## Documentation

| Document | Description |
| --- | --- |
| [docs/cli.md](docs/cli.md) | Full command reference |
| [HUMANS.md](HUMANS.md) | Maintainer primer — install, auth, output formats, shell completion |
| [AGENTS.md](AGENTS.md) | Agent and LLM working conventions |
| [CONTRIBUTING.md](CONTRIBUTING.md) | Commits, branches, pre-push checklist |
| [CHANGELOG.md](CHANGELOG.md) | Release notes |

## License

Proprietary — see [LICENSE](LICENSE). Third-party notices in [NOTICE](NOTICE).
