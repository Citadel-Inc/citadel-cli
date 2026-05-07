# Tasks — cli-repo-insights

## P0

- [ ] A1. Implement `repo insights [<namespace>/<repo>]` with human and JSON output.
- [ ] A2. Wire `repoInsightsCmd` into `cmd/repo.go`.
- [ ] A3. Write handler tests covering: happy path, empty repo (no git commits), 404, 401.

## P1

- [ ] B1. Document `repo insights` surface in `docs/cli.md`.
- [ ] B2. `make verify` green (0 lint issues).

## P2

- [ ] C1. Run live smoke for repo insights against a real Citadel instance.
- [ ] C2. Spec close.
