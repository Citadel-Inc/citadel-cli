# Tasks — cli-kg-extended

Status: DRAFT 050506ZMAY26

## P0

- [ ] [HUMAN] NOMAD ratifies Q-table (Q1–Q3).
- [ ] A1. Path reconciliation audit — document in `plan.md` appendix; add spike PR or inline comment if `kg impact` migrates paths.
- [ ] A2. Implement `kg search`, `kg symbols`, `kg files`, `kg walk`, `kg fulltext`, `kg diff` + httptest suite.
- [ ] A3. Register subcommands under `KgCmd`.

## P1

- [ ] B1. Pagination UX (`--cursor`, `--limit`, `--all`) mirroring `cli-pagination` where server emits `next_cursor`.
- [ ] B2. Table output for symbols/search when `--output table`.

## P2

- [ ] C1. `docs/cli.md` KG section expansion.
- [ ] C2. Live test `CITADEL_TEST_KG_EXTENDED_LIVE=1`.
- [ ] C3. Spec close.
