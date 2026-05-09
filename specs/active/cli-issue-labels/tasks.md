# Tasks — cli-issue-labels

Status: IN_PROGRESS 091315ZMAY26 — Bastion (J-3) claims execution
Priority: Medium

## P0

- [x] A1. `label list -R org/repo` — renders table with slug, name, color, description; supports `--output`.
- [x] A2. `label create -R org/repo --name <name> --color <hex> [--description <text>]` — creates label; prints slug on success.
- [x] A3. `label edit -R org/repo <slug>` — patches name/color/description; errors if no fields supplied.
- [x] A4. `label delete -R org/repo <slug>` — `--yes` bypass; `labelDeleteGuard` 422 surfaced as actionable error.
- [x] A5. Shell completion for label slugs on `edit` and `delete`.
- [x] A6. Handler tests for all verbs including error paths (404, 409, 422).

## P1

- [x] B1. `docs/cli.md` updated with label subcommand reference.
- [x] B2. `make verify` passes (vet, race tests, golangci-lint).

## P2

- [ ] C1. Live smoke test behind `CITADEL_TEST_LABEL_LIVE=1`.
- [ ] C2. Spec close.
