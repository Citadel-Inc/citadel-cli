# Tasks — cli-cwd-context

Status: IN_PROGRESS 050912ZMAY26 — Bastion (J-3) claims execution

## P0

- [ ] [HUMAN] NOMAD ratifies Q-table (Q1-Q5).
- [ ] A1. cmd/repocontext.go: resolveRepoFlag helper with `-R` flag, CITADEL_REPO env, CWD inference (git remote get-url origin), --no-cwd-repo opt-out, friendly failure.
- [ ] A2. CWD inference: parse SSH (`git@host:ns/slug.git`) + HTTPS (`https://host/ns/slug(.git)`); host whitelist + CITADEL_GIT_HOSTS env override.

## P1

- [ ] B1. Wire resolveRepoFlag into `repo get` (positional retained as shortcut), `repo delete`, and `kg impact`.
- [ ] B2. TTY-only stderr hint `Inferred -R <ns>/<slug> from CWD` when inference succeeds.
- [ ] B3. Tests: cwd_context_test.go covering -R explicit, CITADEL_REPO env, ssh + https + .git inference, non-Citadel host failure, --no-cwd-repo opt-out, CITADEL_GIT_HOSTS override.

## P2

- [ ] C1. README "Repo context" section: examples + opt-out guidance.
- [ ] C2. [HUMAN] Operator smoke: cd into a Citadel-cloned repo, run `repo get` without args, confirm round trip.
- [ ] C3. Spec close.
