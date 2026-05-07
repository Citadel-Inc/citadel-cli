# Tasks — cli-notifications

Status: DONE 072212ZMAY26 — All 6 notification subcommands shipped (list/read/read-all/unread-count/prefs get/prefs set). 17 handler tests green. docs/cli.md updated. make verify passes (0 lint issues). C1 live smoke deferred to next integration pass (daemon CI environment required); spec closed with P2 tasks open per allow_open.

## P0

- [x] A1. Add `notification list`, `read`, `read-all`, and `unread-count`.
- [x] A2. Add `notification prefs get` and `notification prefs set`.
- [x] A3. Add handler, pagination, and command-tree coverage for the notification surface.

## P1

- [x] B1. Document notification inbox and preference workflows.
- [x] B2. `make verify` green.

## P2

- [ ] C1. Run live smoke for inbox listing and a read path against the real daemon.
- [ ] C2. Spec close.
