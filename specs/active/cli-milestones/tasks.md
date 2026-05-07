# Tasks — cli-milestones

Status: DRAFT 071745ZMAY26

## P0

- [ ] [HUMAN] NOMAD ratifies Q-table (Q1–Q3).
- [ ] A1. Implement `issue milestone` subcommand tree under existing `issue` command (`cmd/issue_milestone.go`).
- [ ] A2. Implement `list` verb: `GET /api/namespaces/{slug}/milestones` with `--state` filter and pagination.
- [ ] A3. Implement `view` verb: `GET /api/namespaces/{slug}/milestones/{id}` — display progress bar.
- [ ] A4. Implement `create` verb: `POST` with `--title`, `--description`, `--due-on`.
- [ ] A5. Implement `edit` verb: `PUT` with optional flag updates.
- [ ] A6. Implement `delete` verb: `DELETE` with typed-confirm guard.
- [ ] A7. Wire `--milestone <id>` flag into existing `issue create` verb.

## P1

- [ ] B1. httptest coverage for all 5 verbs + 404 path.
- [ ] B2. Shell completion: milestone ID completion for `view`/`edit`/`delete`.
- [ ] B3. `docs/cli.md` milestone section.
- [ ] B4. `make verify` green.

## P2

- [ ] C1. [HUMAN] Live smoke: `issue milestone create`, `list`, and `delete` against a real namespace.
- [ ] C2. Spec close.
