# Tasks — cli-repo-topics

## P0

- [ ] A1. Implement `repo topic list [<namespace>/<repo>]` with human and JSON output.
- [ ] A2. Implement `repo topic set [<namespace>/<repo>] <topic>...` — PUT full topic set.
- [ ] A3. Implement `repo topic popular [--limit N]` with human and JSON output.
- [ ] A4. Wire `repoTopicCmd` into `cmd/repo.go`.
- [ ] A5. Write handler tests covering: list happy, list empty, set happy, set error, popular happy, 401/404.

## P1

- [ ] B1. Document `repo topic` surface in `docs/cli.md`.
- [ ] B2. `make verify` green (0 lint issues).

## P2

- [ ] C1. Run live smoke for topic list/set/popular against a real Citadel instance.
- [ ] C2. Spec close.
