# Tasks — cli-repo-browse

## P0

- [ ] A1. Implement `repo browse tree [<namespace>/<repo>] [--ref] [--path]` with human table and JSON output.
- [ ] A2. Implement `repo browse blob [<namespace>/<repo>] <path> [--ref]`; human mode prints raw content; binary files print informational line.
- [ ] A3. Wire `repoBrowseCmd` into `cmd/repo.go` alongside existing subcommands.
- [ ] A4. Write handler tests covering: tree happy, tree with path filter, tree not-found, blob happy, blob binary, blob not-found, 401 cases.

## P1

- [ ] B1. Document `repo browse` surface in `docs/cli.md`.
- [ ] B2. `make verify` green (0 lint issues).

## P2

- [ ] C1. Run live smoke for browse tree and blob against a real Citadel instance.
- [ ] C2. Spec close.
