# Spec — cli-git-wrappers

| | |
|---|---|
| Status | IN_PROGRESS 071745ZMAY26 — unblocked — citadel#7 resolved: `GET /api/namespaces/{slug}/{repo_slug}` routing fixed and `git_ssh_remote` field added to repo responses. SSH is the canonical transport; HTTPS git was never supported. Implementation updated to use SSH remote URLs. |
| Authored | 120000ZMAY26 |
| Owner | Bastion (J-3) |

## Why

Operators who use `citadel-cli` for repo lifecycle often context-switch to
plain `git` for clone/push/pull. Adding thin wrappers that inject the
correct auth headers and remote URLs removes that friction and lets a single
tool handle the full repository workflow.

## In scope

- `citadel repo clone <repo-path> [<local-dir>]` — equivalent to
  `git clone <server>/<repo-path>` with credentials injected via
  a short-lived `GIT_ASKPASS` helper; prints the local dir path on success
- `citadel repo push [<repo-path>] [--remote <name>] [--create]` — equivalent to
  `git push` in the current working repo with auth wiring; prompts to create the
  remote repo first when Citadel does not know it yet
- `citadel repo pull [<repo-path>] [--remote <name>]` — equivalent to
  `git pull` with auth wiring

## Out of scope

- SSH key management for git operations (covered by `ssh-key` commands)
- Custom git hooks or rebase strategies
- Operations that have no direct git equivalent (branch protection, PR creation)

## Decision log

| Q | Proposal | Status |
|---|----------|--------|
| Q1 | Auth injection: `GIT_ASKPASS` script vs. credential helper vs. HTTP header? | **Ratified 071639ZMAY26** — short-lived `GIT_ASKPASS`; matches git's credential prompt flow without persisting host credentials. |
| Q2 | Should wrappers fall through to system `git` or re-implement git semantics? | **Ratified 071639ZMAY26** — exec-based passthrough via `exec.Command("git", ...)`. |
| Q3 | `citadel clone` vs. `citadel repo clone`? | **Ratified 071639ZMAY26** — second-level verbs under `repo` to match the existing CLI shape. |
| Q4 | Handle repos not yet on the server (auto-create on push)? | **Ratified 071639ZMAY26** — prompt to create on `repo push`; `--create` bypasses the prompt for non-interactive use. |

