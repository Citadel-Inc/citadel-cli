# Tasks — cli-org-members

Status: IN_PROGRESS 080751ZMAY26 — Bastion (J-3) claims execution
Priority: Medium

## P0

- [ ] A1. Implement `org member list <org-slug>` with pagination and `--output json|yaml|table|csv`.
- [ ] A2. Implement `org member set-permissions <org-slug> <member> --permission <atoms>` (replace permission set).
- [ ] A3. Implement `org member remove <org-slug> <member>` with clear error surfaces for lockout/owner-guard.
- [ ] A4. Slug-resolution: if `<member>` is not UUID-shaped, resolve via list call; 404 if not found.
- [ ] A5. Unit/handler tests for all three verbs including error paths.

## P1

- [ ] B1. `docs/cli.md` updated with member subcommand reference.
- [ ] B2. `make verify` passes (vet, race tests, golangci-lint).

## P2

- [ ] C1. Live smoke against real daemon (gated on `CITADEL_TEST_ORG_MEMBERS_LIVE=1`).
- [ ] C2. Spec close.
