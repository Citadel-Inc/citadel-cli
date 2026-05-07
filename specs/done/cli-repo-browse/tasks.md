# Tasks — cli-repo-browse

## P0

- [x] A1. Implement `repo browse tree [<namespace>/<repo>] [--ref] [--path]` with human table and JSON output.
- [x] A2. Implement `repo browse blob [<namespace>/<repo>] <path> [--ref]`; human mode prints raw content; binary files print informational line.
- [x] A3. Wire `repoBrowseCmd` into `cmd/repo.go` alongside existing subcommands.
- [x] A4. Write handler tests covering: tree happy, tree with path filter, tree not-found, blob happy, blob binary, blob not-found, 401 cases.

## P1

- [x] B1. Document `repo browse` surface in `docs/cli.md`.
- [x] B2. `make verify` green (0 lint issues).

## P2

- [x] C1. Run live smoke for browse tree and blob against a real Citadel instance.
- [x] C2. Spec close.
