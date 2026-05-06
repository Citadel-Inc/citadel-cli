# Spec — cli-git-wrappers

| | |
|---|---|
| Status | DRAFT |
| Authored | 120000ZMAY26 |
| Owner | unassigned |

## Why

Operators who use `citadel-cli` for repo lifecycle often context-switch to
plain `git` for clone/push/pull. Adding thin wrappers that inject the
correct auth headers and remote URLs removes that friction and lets a single
tool handle the full repository workflow.

## In scope

- `citadel clone <repo-path> [<local-dir>]` — equivalent to
  `git clone <server>/<repo-path>` with credentials injected via
  `GIT_ASKPASS` or credential helper; prints the local dir path on success
- `citadel push [<repo-path>] [--remote <name>]` — equivalent to
  `git push` in the current working repo with auth wiring
- `citadel pull [<repo-path>] [--remote <name>]` — equivalent to
  `git pull` with auth wiring

## Out of scope

- SSH key management for git operations (covered by `ssh-key` commands)
- Custom git hooks or rebase strategies
- Operations that have no direct git equivalent (branch protection, PR creation)

## Decision log

| Q | Proposal | Status |
|---|----------|--------|
| Q1 | Auth injection: `GIT_ASKPASS` script vs. credential helper vs. HTTP header? | OPEN — `GIT_ASKPASS` is simplest; credential helper is more durable |
| Q2 | Should wrappers fall through to system `git` or re-implement git semantics? | OPEN — strong preference for exec-based passthrough (`exec.Command("git", ...)`) |
| Q3 | `citadel clone` vs. `citadel repo clone`? | OPEN — top-level mirrors `gh repo clone` pattern; avoids deep nesting |
| Q4 | Handle repos not yet on the server (auto-create on push)? | OPEN — defer to explicit `repo create` first at v1 |
