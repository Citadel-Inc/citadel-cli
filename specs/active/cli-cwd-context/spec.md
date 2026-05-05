# Spec — cli-cwd-context

| | |
|---|---|
| Status | DRAFT 081550ZMAY26 |
| Authored | 081550ZMAY26 |
| Owner | Bastion (J-3) |
| Carry-forward from | third-pass review of `citadel-cli` (2026-05-05): smart-context detection from CWD git remote. Mirrors `gh -R` defaulting from local repo. |

## Why

Operators inside a Citadel-cloned repo working tree always know which repo they're in — but the CLI doesn't. Today every verb that needs `<ns>/<slug>` requires the user to type it explicitly:

```
$ pwd
/home/alice/code/myorg/myrepo
$ citadel-cli repo get myorg/myrepo     # but I'm already in it
$ citadel-cli issue list -R myorg/myrepo  # ditto
```

`gh` solves this by reading `git remote get-url origin` and parsing the path component. Add the same to citadel-cli.

## In scope

- **`-R <ns>/<slug>` (or `--repo <ns>/<slug>`) standardised** across every verb that targets a repo. (Today's `repo get` takes a positional arg; that's retained as a shortcut, but `-R` is the canonical name aligned with `gh`.)
- **CWD git-remote inference**: when `-R` is omitted *and* CWD is inside a git repo with an `origin` remote pointing at a Citadel-served host, parse `<host>:<ns>/<slug>` (SSH) or `https://<host>/<ns>/<slug>(.git)` (HTTPS) and fill in. Failure to parse → friendly "could not infer repo from CWD; pass -R <ns>/<slug>" error.
- **Host whitelist**: only treat `api.src.land`, `src.land`, `mcp.src.land`, and `git.src.land` (or whatever the canonical Citadel git host turns out to be) as Citadel hosts. Other hosts → ignore + fall through to "specify -R" error.
- **`--repo` env var**: `CITADEL_REPO=ns/slug` overrides CWD inference. Honored when no `-R` flag and CWD inference doesn't apply.
- **Opt-out flag**: `--no-cwd-repo` forces strict mode (don't infer; fail fast). For scripts that must not pick up arbitrary CWD context.
- **Verbs in scope**: `repo get`, `repo delete` (when ratified Q3 below covers positional vs `-R`), `issue *` (cli-issue-pr spec), `pr *` (cli-issue-pr spec), `kg impact`. Anything else taking `<ns>/<slug>` as an explicit positional remains explicit at v1.

## Out of scope

- **`--namespace` (no slug) inference**: only repos infer from CWD. Top-level namespace verbs still take `--namespace` explicitly.
- **Multi-remote handling**: only `origin`. If `origin` is non-Citadel and `upstream` is, that's an edge case; document in HUMANS, don't auto-prefer.
- **Branch-scoped inference**: e.g., `pr create` could read current branch as `--head`. That's a separate spec (workflow integration).
- **Submodule traversal**: only the closest `.git` dir wins; submodule URLs are not parsed.

## Decision log

| Q | Proposal | Status |
|---|----------|--------|
| Q1 | URL parse: shell out to `git remote get-url origin` vs. read `.git/config` directly? | **Open** — `git` shell-out; handles git-config alias rewrites + worktrees correctly without us reimplementing. |
| Q2 | Host whitelist: hardcoded list vs. env override (`CITADEL_GIT_HOSTS`)? | **Open** — both: hardcoded defaults + env override for self-hosted Citadel installs. |
| Q3 | `repo get <ns>/<slug>` positional shape: keep + add -R, or migrate to -R only? | **Open** — keep both (positional shortcut survives; -R works everywhere). |
| Q4 | Inference failure: warn-and-prompt vs. silent fail-with-error? | **Open** — silent fail with friendly error (`could not infer repo from CWD; pass -R <ns>/<slug>`); no surprise. |
| Q5 | TTY warning when -R is omitted but inference succeeds (`Inferred -R myorg/myrepo from CWD`)? | **Open** — yes, on stderr; CI suppresses with `--quiet` or `CI=1` once we add a stderr-quiet flag. |

## Acceptance

- A1. `cmd/repocontext.go`: `resolveRepoFlag(cmd) (ns, slug string, err error)` helper handling `-R` flag, `CITADEL_REPO` env, CWD inference, `--no-cwd-repo` opt-out, and the friendly failure path.
- A2. `repo get`, `repo delete` (positional retained), and `kg impact` accept `-R` at v1; the cli-issue-pr spec uses the same helper.
- A3. CWD inference works against SSH (`git@src.land:myorg/myrepo.git`) and HTTPS (`https://src.land/myorg/myrepo`, with or without trailing `.git`) origin URLs.
- A4. Friendly inference TTY hint on stderr (`Inferred -R myorg/myrepo from CWD`).
- A5. `CITADEL_GIT_HOSTS` env override accepts comma-separated list.
- A6. Tests cover: -R explicit, env-only, inference happy paths (ssh + https + .git suffix), inference failure (non-Citadel host), --no-cwd-repo opt-out.
- A7. Q-table ratified.
