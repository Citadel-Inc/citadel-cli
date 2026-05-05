# citadel-cli — agent primer

If you are an LLM working in this repository, read this file first. `CLAUDE.md` is a symlink to it — **always edit `AGENTS.md`, never `CLAUDE.md`** (a Write to the symlink path replaces the link with a regular file and silently breaks the alias).

For human maintainer onboarding see [HUMANS.md](HUMANS.md). For commit conventions, branch policy, and contributor checklist see [CONTRIBUTING.md](CONTRIBUTING.md).

## Repository shape

See [HUMANS.md § Repository layout](HUMANS.md#repository-layout) for the canonical tree. Spec lifecycle uses the `mcp__citadel-sdd__*` MCP tools (table below); release workflow fires on `v*` tags.

## Working conventions

- **Conventional commits.** `type(scope): subject`; body explains motivation (WHY), not a file list. See [CONTRIBUTING.md](CONTRIBUTING.md).
- **Continuous commit + push** authorised for the duration of `citadel-cli` work, using MCP Git tools when available.
- **MCP-first Git/GitHub.** Prefer `mcp__rethunk-git__*` and `mcp__rethunk-github__*` over Bash `git` / `gh`. Bash is fallback only when an MCP tool genuinely lacks the operation.
- **Specs** MUST pass `mcp__citadel-sdd__spec_lint` before commit. Canonical bullet shape `- [ ]` / `- [x]`. Priority headings (`## P0` / `## P1` / `## P2`) live in `tasks.md` only.

## Spec lifecycle — use the MCP

**Hard rule: use `mcp__citadel-sdd__*` tools for all spec lifecycle operations.** Never hand-edit status fields, DTG stamps, or `tasks.md` state lines. The tools enforce lint rules, write correct frontmatter, stamp accurate DTGs, and commit with the right message style.

| What you want | Tool |
|---|---|
| Claim (DRAFT/APPROVED → IN_PROGRESS) | `spec_claim` |
| Approve (DRAFT → APPROVED) | `spec_approve` |
| Close (IN_PROGRESS → DONE) | `spec_close` |
| Block / unblock | `spec_block` / `spec_unblock` |
| Reopen (DONE → IN_PROGRESS) | `spec_reopen` |
| Hand off owner | `spec_handoff` |
| Check / uncheck task | `spec_task_check` |
| Add task item | `spec_task_add` |
| Lint (strict, cross-cutting) | `spec_lint` |
| List by state | `spec_list` |
| Read spec files | `spec_read` |
| Health + finding report | `sdd_doctor` |

**MCP ergonomics:** `spec_claim` requires `claimer` equal to the spec **Owner** line verbatim. `spec_close` requires a non-empty `summary`; use `allow_open` when closing with deliberate unchecked rows in a phase. `spec_task_check`: prefer `dryRun: true` to preview flips.

### When hand-editing IS correct

- **Creating a new spec** — no `spec_create` tool. Scaffold `spec.md`, `tasks.md`, `plan.md` manually, then call `spec_claim` to stamp state and commit.
- **Body prose edits** — acceptance criteria, plan narrative, Q-table rationale.
- **Bulk hygiene in `specs/done/`** — run `sdd_doctor` first, make targeted edits, verify with `spec_lint --include_done`.
- **Parking a spec** — `citadel-sdd` has no `spec_park` tool yet. To retire an idea without shipping it, move the tree to `specs/parked/<slug>/`, stamp **PARKED** + resolution in `spec.md`, and record the rationale in `specs/parked/README.md` (or the spec’s Resolution section). Prefer a conventional commit that names the superseding decision.

## Test conventions

- `go test -race ./...` is the canonical gate.
- Live integration tests gate on env vars (e.g., `CITADEL_TEST_OAUTH_JWT`) and self-skip when unset — safe to run in CI.

## Pre-push checklist

`make verify` — vet, race tests, golangci-lint. Fix anything that fails before pushing.
