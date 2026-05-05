# Contributing to citadel-cli

This guide is for **human** contributors. If you are an LLM or automation working in the repo, read [AGENTS.md](AGENTS.md) (`CLAUDE.md` is a symlink) first — it carries agent-specific workflow (MCP-first Git, spec MCP tools, and the same conventions below in agent-shaped form).

## Commit conventions

We use [Conventional Commits](https://www.conventionalcommits.org/):

- **Subject:** `type(scope): subject` — types include `feat`, `fix`, `docs`, `chore`, `test`, `refactor`, `ci`, and similar; `scope` is optional but preferred when the change touches a clear area (e.g. `auth`, `mcp`, `repo`).
- **Body:** optional; when present it should explain **why** the change is needed, not merely restate **what** the diff does.

## Branch and push policy

- Use **short-lived topic branches** off `main`.
- **Do not force-push to `main`.** Rewrite history only on your own branch when coordinating with anyone else who has checked it out.

## Spec discipline

Specs live under `specs/active/` and `specs/done/`. Authoring rules and lifecycle are defined by [`@rethunk/citadel-sdd`](https://github.com/Rethunk-AI/citadel-sdd). Use the `mcp__citadel-sdd__*` MCP tools — see [AGENTS.md](AGENTS.md) for the tool→operation map. Strict spec lint (`mcp__citadel-sdd__spec_lint`) must pass before merge.

## Pre-commit checklist

Before you ask for review or push a change you care about:

1. **`make verify`** — runs `go vet`, race tests, and `golangci-lint`. Fix anything that fails.
2. **Spec edits** — if you touch `specs/**`, the strict SDD linter must pass before merge; use the Citadel SDD MCP `spec_lint` (or the equivalent CLI from `@rethunk/citadel-sdd`).

## Code style (mechanical)

- **Go:** `gofmt` / `go vet`; the `Makefile` exposes `make fmt`, `make lint`, `make test`, `make verify`.

Do not replace these tools with informal style rules in prose — extend `.golangci.yml` if a new rule is needed.

## IP and license posture

`citadel-cli` is **proprietary software**. Copyright and all rights are reserved by **Rethunk.Tech, LLC.** — see the repository root **[LICENSE](LICENSE)**. Third-party open-source components are listed in **[NOTICE](NOTICE)**.

By submitting a patch, pull request, or other contribution that is merged into this repository, you certify that:

1. You have the legal right to grant the permissions described below, and the contribution does not include third-party material you are not authorised to submit.
2. You grant **Rethunk.Tech, LLC.** a perpetual, irrevocable, worldwide, royalty-free license to use, reproduce, modify, distribute, and otherwise exploit your contribution as part of `citadel-cli` and related products, without further obligation to you except as required by law.

**Sign-off (optional but preferred):** include a `Signed-off-by` trailer with your real name and a reachable email address (e.g. `git commit -s`) so maintainers can trace provenance.
