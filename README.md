# citadel-cli

Operator and developer command-line interface for [Citadel](https://github.com/Rethunk-Tech/citadel).

`citadel-cli` is the official client for managing repositories, namespaces, agents, OAuth clients, and the Citadel Knowledge Graph. It also embeds an MCP client for integrating Citadel into agentic workflows.

## Install

```bash
go install github.com/Rethunk-Tech/citadel-cli@latest
```

This installs to `~/go/bin/citadel-cli`; ensure `~/go/bin` is on your `PATH`.

Pre-built release binaries (linux-amd64, linux-arm64, darwin-arm64) are published to GitHub Releases on every `v*` tag — see <https://github.com/Rethunk-Tech/citadel-cli/releases>.

## Quick start

```bash
citadel-cli auth login        # OAuth flow via the configured Citadel server
citadel-cli auth status       # confirm authentication
citadel-cli repo list         # query the API
```

Full reference: [docs/cli.md](docs/cli.md).

## Documentation

- [docs/cli.md](docs/cli.md) — full command reference
- [HUMANS.md](HUMANS.md) — maintainer primer
- [AGENTS.md](AGENTS.md) — agent / LLM working conventions
- [CONTRIBUTING.md](CONTRIBUTING.md) — commits, branches, pre-push checklist

## License

Proprietary — see [LICENSE](LICENSE). Third-party notices in [NOTICE](NOTICE).
