# Tasks — cli-repo-insights

## P0

- [x] A1. Implement `repo insights [<namespace>/<repo>]` with human and JSON output.
- [x] A2. Wire `repoInsightsCmd` into `cmd/repo.go`.
- [x] A3. Write handler tests covering: happy path, empty repo (no git commits), 404, 401.

## P1

- [x] B1. Document `repo insights` surface in `docs/cli.md`.
- [x] B2. `make verify` green (0 lint issues).

## P2

- [x] C1. Run live smoke for repo insights against a real Citadel instance.
- [x] C2. Spec close.
