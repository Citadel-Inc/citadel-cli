# citadel-cli

[![CI](https://github.com/Rethunk-Tech/citadel-cli/actions/workflows/ci.yml/badge.svg)](https://github.com/Rethunk-Tech/citadel-cli/actions/workflows/ci.yml)
[![Release](https://github.com/Rethunk-Tech/citadel-cli/actions/workflows/cli-release.yml/badge.svg)](https://github.com/Rethunk-Tech/citadel-cli/actions/workflows/cli-release.yml)
[![Go Reference](https://pkg.go.dev/badge/github.com/Rethunk-Tech/citadel-cli.svg)](https://pkg.go.dev/github.com/Rethunk-Tech/citadel-cli)
[![Go Report Card](https://goreportcard.com/badge/github.com/Rethunk-Tech/citadel-cli)](https://goreportcard.com/report/github.com/Rethunk-Tech/citadel-cli)
[![Go Version](https://img.shields.io/github/go-mod/go-version/Rethunk-Tech/citadel-cli)](go.mod)
[![Latest Release](https://img.shields.io/github/v/release/Rethunk-Tech/citadel-cli?include_prereleases&sort=semver)](https://github.com/Rethunk-Tech/citadel-cli/releases)
[![License: Proprietary](https://img.shields.io/badge/license-proprietary-red.svg)](LICENSE)

Operator and developer command-line interface for [Citadel](https://github.com/Rethunk-Tech/citadel).

`citadel-cli` is the official client for managing repositories, namespaces, agents, OAuth clients, audit log queries, and the Citadel Knowledge Graph. It also embeds an MCP client for integrating Citadel into agentic workflows.

## Getting started

Install, authenticate, and first-run commands live in [HUMANS.md § Getting started](HUMANS.md#getting-started). Repository defaults (`-R`, `CITADEL_REPO`, CWD inference, `CITADEL_GIT_HOSTS`) are documented in [HUMANS.md § Repo context](HUMANS.md#repo-context).

## Shell completion

Cobra emits integration scripts via `citadel-cli completion bash|zsh|fish|powershell`. How dynamic completion uses your session, list pagination flags (`--limit` / `--cursor` / `--all`), live list streaming (`--watch` / `-w`), on-disk caching, and related environment variables are covered in [HUMANS.md § Shell completion](HUMANS.md#shell-completion) and [HUMANS.md § List pagination](HUMANS.md#list-pagination).

## JSON error envelope

When a command fails with `--output json`, `yaml`, or `ndjson`, the CLI writes one structured **error** object to **stdout** (stderr stays empty) and exits with a class-specific code (for example `6` for rate limits). Human/table modes keep the usual `Error: …` line on stderr. See [HUMANS.md § Structured errors](HUMANS.md#structured-errors-output-json--yaml--ndjson) for the full shape, `kind` values, and exit-code table.

## Output formats

Machine-readable list output uses `--output json|yaml|ndjson|csv|table` (default human table). **Get/show** verbs accept `json|yaml|table` only — `csv` and `ndjson` are list/stream shapes.

- **`json`** — one indented JSON value per invocation; with pagination that is a single page (array of rows). **`--all` with `--output json` is rejected**; use **`ndjson`** (or human/`table`) to stream.
- **`ndjson`** — one compact JSON object per row and per line; newline after every record, including the last.
- **`csv`** — RFC 4180-style rows to stdout; **header emitted once** at the first data batch (empty lists emit a header-only row). Column order is **frozen per command** (see table below).
- **`yaml`** — one YAML document; for lists, a sequence. Keys match **`json`** output (stable for scripting).

| List command | CSV columns (exact order) |
|--------------|---------------------------|
| `repo list` | slug, path, visibility, default_branch, description, namespace_id, parent_slug, created_at |
| `agent list` | id, owner_user_id, name, model_hint |
| `token list` | id, agent_id, created_at, expires_at, revoked_at, scopes |
| `oauth clients list` | id, client_id, name, allowed_scopes, is_public, owner_slug, created_at, updated_at, revoked_at |
| `namespace list` | namespace_id, slug, display_name, legal_entity_name, created_at |
| `namespace members` | user_id, email, slug, display_name, is_owner, permissions, joined_at |
| `namespace transfer list-pending` | id, org_namespace_id, org_slug, org_name, from_user_id, from_user_slug, to_user_id, to_user_slug, expires_at, created_at |
| `ssh-key list` | id, fingerprint, public_key, label, created_at |
| `account passkey list` | id, name, created_at |
| `account device list` | id, name, user_agent, last_seen_at, created_at |
| `org invitation list` / `org invitation list-pending` | id, org_slug, email, user_slug, status, permissions, created_at, expires_at |
| `audit list` | id, ts, kind, actor_slug, actor_id, namespace_slug, namespace_id, subject_id, actor_type |
| `audit sessions list` | session_id, id, actor_slug, actor_id, actor_type, namespace_slug, namespace_id, started_at, last_event_at, event_count |

Time-like CSV fields use **RFC3339 UTC** (`…Z`). See [HUMANS.md § List pagination](HUMANS.md#list-pagination) for `--all` + `--output ndjson` streaming.

## Documentation

- [docs/cli.md](docs/cli.md) — full command reference
- [HUMANS.md](HUMANS.md) — maintainer primer
- [AGENTS.md](AGENTS.md) — agent / LLM working conventions
- [CONTRIBUTING.md](CONTRIBUTING.md) — commits, branches, pre-push checklist

## License

Proprietary — see [LICENSE](LICENSE). Third-party notices in [NOTICE](NOTICE).
