# Tasks — cli-repo-commits

## P0

- [ ] A1. Implement `repo commit list [<namespace>/<repo>]` with `--ref`, `--path`, pagination flags, and standard list outputs.
- [ ] A2. Implement `repo commit get [<namespace>/<repo>] <sha>` with `--path` flag for per-file unified diff.
- [ ] A3. Wire `repoCommitCmd` into `cmd/repo.go` alongside `BranchCmd` and `TagCmd`.
- [ ] A4. Add CSVRow implementation for `commitItem` in `cmd/list_csv.go`.
- [ ] A5. Write handler tests covering list, get, get-with-diff, 404, and no-auth scenarios.

## P1

- [ ] B1. Document `repo commit` surface in `docs/cli.md`.
- [ ] B2. `make verify` green (0 lint issues).

## P2

- [ ] C1. Run live smoke for commit list and get against a real Citadel instance.
- [ ] C2. Spec close.
