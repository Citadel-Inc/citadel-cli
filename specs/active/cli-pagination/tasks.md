# Tasks — cli-pagination

Status: IN_PROGRESS 050956ZMAY26 — Bastion (J-3) claims execution

Server-side delivery to be split into a citadel-repo spec once Q-table ratifies. Tasks below cover the CLI surface; server tasks are placeholders.

## P0

- [x] [HUMAN] NOMAD ratifies Q-table (Q1-Q6).
- [x] A1. [SERVER] Cursor codec + paging on every list endpoint. Response envelope adds `next_cursor`.
- [x] A2. [CLI] Add `--limit` / `--cursor` / `--all` to every list verb (`repo list`, `agent list`, `token list`, `oauth clients list`, `namespace list`, `namespace members`, `namespace transfer list-pending`).

## P1

- [x] B1. [CLI] Page-streaming `--all`: emit rows incrementally to stdout via tabwriter Flush per page. Honor `--output ndjson` for clean piping.
- [x] B2. [CLI] Default-mode tail line `(use --cursor <next> for more, or --all to fetch everything)` when `next_cursor` is non-empty.
- [x] B3. [CLI] `--cursor` validation: reject obviously malformed cursors with a friendly error (delegate to server otherwise).
- [x] B4. Tests: 3-page handler fixture in `cmd/handler_test.go` covering `--limit`, `--cursor`, `--all` exhaustion paths.
- [x] B5. README + HUMANS.md: document the new flags, the limit cap, the streaming guarantees of `--all` + `--output ndjson`.

## P2

- [ ] C1. Live integration test (`CITADEL_TEST_PAGINATION_LIVE=1`) — populate 250 repos in a test namespace, walk via `--all`, assert count.
- [ ] C2. [HUMAN] Production smoke: list a real namespace with > 50 entities once server-side paging lands.
- [ ] C3. Spec close.
