# citadel-cli

[![CI](https://github.com/Rethunk-Tech/citadel-cli/actions/workflows/ci.yml/badge.svg)](https://github.com/Rethunk-Tech/citadel-cli/actions/workflows/ci.yml)
[![Release](https://github.com/Rethunk-Tech/citadel-cli/actions/workflows/cli-release.yml/badge.svg)](https://github.com/Rethunk-Tech/citadel-cli/actions/workflows/cli-release.yml)
[![Go Reference](https://pkg.go.dev/badge/github.com/Rethunk-Tech/citadel-cli.svg)](https://pkg.go.dev/github.com/Rethunk-Tech/citadel-cli)
[![Go Report Card](https://goreportcard.com/badge/github.com/Rethunk-Tech/citadel-cli)](https://goreportcard.com/report/github.com/Rethunk-Tech/citadel-cli)
[![Go Version](https://img.shields.io/github/go-mod/go-version/Rethunk-Tech/citadel-cli)](go.mod)
[![Latest Release](https://img.shields.io/github/v/release/Rethunk-Tech/citadel-cli?include_prereleases&sort=semver)](https://github.com/Rethunk-Tech/citadel-cli/releases)
[![License: Proprietary](https://img.shields.io/badge/license-proprietary-red.svg)](LICENSE)

Operator and developer command-line interface for [Citadel](https://github.com/Rethunk-Tech/citadel).

`citadel-cli` is the official client for managing repositories, namespaces, agents, OAuth clients, and the Citadel Knowledge Graph. It also embeds an MCP client for integrating Citadel into agentic workflows.

## Getting started

Install, authenticate, and first-run commands live in [HUMANS.md § Getting started](HUMANS.md#getting-started). Repository defaults (`-R`, `CITADEL_REPO`, CWD inference, `CITADEL_GIT_HOSTS`) are documented in [HUMANS.md § Repo context](HUMANS.md#repo-context).

## Shell completion

Cobra emits integration scripts via `citadel-cli completion bash|zsh|fish|powershell`. How dynamic completion uses your session, what gets completed, on-disk caching, and related environment variables are covered in [HUMANS.md § Shell completion](HUMANS.md#shell-completion).

## JSON error envelope

When a command fails with `--output json`, `yaml`, or `ndjson`, the CLI writes one structured **error** object to **stdout** (stderr stays empty) and exits with a class-specific code (for example `6` for rate limits). Human/table modes keep the usual `Error: …` line on stderr. See [HUMANS.md § Structured errors](HUMANS.md#structured-errors-output-json--yaml--ndjson) for the full shape, `kind` values, and exit-code table.

## Documentation

- [docs/cli.md](docs/cli.md) — full command reference
- [HUMANS.md](HUMANS.md) — maintainer primer
- [AGENTS.md](AGENTS.md) — agent / LLM working conventions
- [CONTRIBUTING.md](CONTRIBUTING.md) — commits, branches, pre-push checklist

## License

Proprietary — see [LICENSE](LICENSE). Third-party notices in [NOTICE](NOTICE).
