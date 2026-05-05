# Tasks — cli-projectgraph

Status: DRAFT 050506ZMAY26

## P0

- [ ] [HUMAN] NOMAD ratifies Q-table (Q1–Q3).
- [ ] A1. `cmd/project.go`: `ProjectCmd` + read subcommands (`pin-chain`, `walk`, `neighbors`, `status`, `status drilldown` or nested).
- [ ] A2. Integration with `newAPIClient` + correct `/api/projectgraph/` URL builder for multi-segment slugs.
- [ ] A3. Handler tests (httptest) for at least pin-chain + status-rollup happy paths.

## P1

- [ ] B1. Write verbs: `edge add/delete/restore`, `reindex`, `admin recovery-scan` (as applicable to RBAC).
- [ ] B2. Human-readable summaries + `--output` for all verbs.
- [ ] B3. Documentation in `docs/cli.md` + HUMANS pointer.

## P2

- [ ] C1. Live integration test (`CITADEL_TEST_PROJECTGRAPH_LIVE=1`).
- [ ] C2. Spec close after Q-table + verification.
