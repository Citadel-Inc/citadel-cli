# Tasks — cli-projectgraph

Status: DRAFT 050506ZMAY26

## P0

- [ ] [HUMAN] NOMAD ratifies Q-table (Q1–Q4).
- [ ] A1. Append **appendix** to `plan.md`: copy query-param tables + JSON structs from `handleStatusRollup`, `handleStatusRollupDrilldown`, `handleReindex`, recovery-scan handler (exact server RECON).
- [ ] A2. Implement URL builder + read verbs (`pin-chain`, `walk`, `neighbors`, `status`, `status drilldown`) + httptest coverage per verb.
- [ ] A3. Wire `ProjectCmd` in `cmd/root.go`.

## P1

- [ ] B1. Write verbs (`edge add/delete/restore`, `reindex`) + `--yes` / confirmation policy per Q3.
- [ ] B2. Admin `recovery-scan` if Q4 ratifies inclusion.
- [ ] B3. Human-readable tables + `--output` integration.

## P2

- [ ] C1. `docs/cli.md` + HUMANS cross-links.
- [ ] C2. Live test `CITADEL_TEST_PROJECTGRAPH_LIVE=1`.
- [ ] C3. Spec close.
