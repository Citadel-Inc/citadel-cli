# Tasks — cli-org-members

Status: IN_PROGRESS 080751ZMAY26 — Bastion (J-3) claims execution
Priority: Medium

## P0

- [x] A1. Implement `org member list <org-slug>` with pagination and `--output json|yaml|table|csv`.
- [x] A2. Implement `org member set-permissions <org-slug> <member> --permission <atoms>` (replace permission set).
- [x] A3. Implement `org member remove <org-slug> <member>` with clear error surfaces for lockout/owner-guard.
- [x] A4. Slug-resolution: if `<member>` is not UUID-shaped, resolve via list call; 404 if not found.
- [x] A5. Unit/handler tests for all three verbs including error paths.

## P1

- [x] B1. `docs/cli.md` updated with member subcommand reference.
- [x] B2. `make verify` passes (vet, race tests, golangci-lint).

## P2

- [ ] C1. Live smoke against real daemon (gated on `CITADEL_TEST_ORG_MEMBERS_LIVE=1`).
- [ ] C2. Spec close.
