# Changelog

All notable changes to `citadel-cli` are documented here.

## v0.1.0 - 2026-05-07

Initial visible release of the Citadel command-line client.

### Added

- Added the Cobra-based `citadel-cli` binary with version injection, global server selection, quiet/verbose/debug HTTP flags, color controls, pager support, shell completion, manpage generation, and `doctor` diagnostics.
- Added browser OAuth login with PKCE, headless `auth set-token`, eager JWT-to-agent-token migration, token status/logout flows, OAuth provider list/link/unlink, and one-shot 401 recovery via agent-token rotation.
- Added agent-token management, `agent create` with initial-token issuance, and agent list/get/delete/rotate-token workflows.
- Added repository, namespace, and account surfaces: repo/namespace/agent CRUD, namespace alias `ns`, account passkeys/devices, account SSH keys, org invitations, namespace profiles, and namespace notifications.
- Added repository workflows for clone/push/pull over SSH, branch and tag management, deploy tokens, webhooks, commit list/get, tree/blob browsing, topics, and repository insights.
- Added namespace issue workflows including list/view/create/comment/close/reopen/label/close-refs and issue milestone list/view/create/edit/delete.
- Added top-level project graph verbs, authenticated global search, audit event list/show, audit session list/show, and extended Knowledge Graph queries.
- Added MCP client commands for tools, calls, resources, and prompts, including typed MCP error handling and protocol-version mismatch reporting.
- Added machine-readable output modes (`json`, `yaml`, `ndjson`, `csv`, and table where supported), cursor pagination, watch/SSE streaming for list verbs, frozen CSV projections, and structured error envelopes.
- Added dynamic API-backed shell completion with on-disk caching, repo-context resolution from `-R`, `CITADEL_REPO`, and git remotes, and dry-run support on destructive verbs.
- Added cross-platform release builds for linux-amd64, linux-arm64, and darwin-arm64, published by the GitHub Actions release workflow on `v*` tags.

### Changed

- Default server routing now uses `https://mcp.src.land`, with production REST calls coerced to the correct API host while OAuth and MCP continue through the MCP host.
- Repository git wrappers now use Citadel-provided SSH remotes instead of HTTPS askpass flows, preserving normal system `git` behavior.
- API access is centralized through shared HTTP clients, retry/trace transports, typed HTTP errors, Retry-After handling, per-client timeouts, and context propagation across command handlers.
- Command output helpers, flag helpers, destructive-confirmation paths, and get/list emitters were consolidated for consistent behavior across the command tree.
- The release toolchain now targets Go `1.25.10` and refreshed indirect module dependencies, including the `github.com/go-jose/go-jose/v3` update that supersedes the Dependabot bump.
- Documentation was reorganized into README, HUMANS, AGENTS, CONTRIBUTING, and `docs/cli.md`, with the command reference expanded to cover the shipped surface.

### Fixed

- Fixed auth login so the authorization URL is always printed before browser opening is attempted.
- Fixed repo push to use explicit branch refspecs and corrected repo wrapper behavior around SSH remotes.
- Fixed watch/SSE streams so long-running streams are not closed by request timeouts.
- Fixed local validation and `--dry-run` paths so they do not require authentication before returning validation output.
- Fixed the release workflow so tag builds run lint through the pinned `golangci-lint` action available on GitHub-hosted runners.
- Fixed completion ordering, MCP unauthorized detection, KG impact repo resolution, API host handling, and several lint/staticcheck issues.
- Bumped dependencies and Go tooling, including the Go toolchain update needed for standard-library vulnerability coverage.

### Tests

- Added broad handler, helper, and command-tree coverage across auth, repo, issue, notification, webhook, audit, project graph, completion, MCP, config, HTTP, SSE, pager, and output-format paths.
- Added opt-in live smoke tests for OAuth, repositories, issues, milestones, deploy tokens, audit, project graph, search, SSH keys, account security, and related API-backed workflows.
- Established `make verify` as the release gate: `go vet`, `go test -race ./...`, and `golangci-lint run`.
