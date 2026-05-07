# Tasks — cli-milestones

Status: DONE 071809ZMAY26 — Shipped `issue milestone` list/view/create/edit/delete plus milestone UUID completion and `issue create --milestone` wiring. Added handler and completion coverage, documented the workflow in docs/cli.md, and completed live smoke on namespace `rethunk-ai` by creating milestone `93a00575-4530-4a7c-8a59-aeccbb47a5ef`, listing it, creating issue `#1` attached to that milestone, and deleting the milestone successfully.

## P0

- [x] [HUMAN] NOMAD ratifies Q-table (Q1–Q3).
- [x] A1. Implement `issue milestone` subcommand tree under existing `issue` command (`cmd/issue_milestone.go`).
- [x] A2. Implement `list` verb: `GET /api/namespaces/{slug}/milestones` with `--state` filter and pagination.
- [x] A3. Implement `view` verb: `GET /api/namespaces/{slug}/milestones/{id}` — display progress bar.
- [x] A4. Implement `create` verb: `POST` with `--title`, `--description`, `--due-on`.
- [x] A5. Implement `edit` verb: `PUT` with optional flag updates.
- [x] A6. Implement `delete` verb: `DELETE` with typed-confirm guard.
- [x] A7. Wire `--milestone <id>` flag into existing `issue create` verb.

## P1

- [x] B1. httptest coverage for all 5 verbs + 404 path.
- [x] B2. Shell completion: milestone ID completion for `view`/`edit`/`delete`.
- [x] B3. `docs/cli.md` milestone section.
- [x] B4. `make verify` green.

## P2

- [x] C1. [HUMAN] Live smoke: `issue milestone create`, `list`, and `delete` against a real namespace.
- [x] C2. Spec close.
