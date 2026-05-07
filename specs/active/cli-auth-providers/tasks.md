# Tasks — cli-auth-providers

Status: IN_PROGRESS 071813ZMAY26 — Copilot claims execution

## P0

- [x] A1. Implement `auth provider list` against `GET /api/auth/providers`.
- [x] A2. Add provider ID completion support for auth-provider verbs.
- [x] A3. Add tests covering command wiring and request/response handling.

## P1

- [x] B1. Add `auth provider link <provider>` with browser-open default and structured output option.
- [x] B2. Add `auth provider unlink <provider>` with confirmation guard and direct daemon-error surfacing.
- [x] B3. Document auth-provider discovery, link, and unlink workflows.
- [x] B4. `make verify` green.

## P2

- [ ] C1. Live smoke provider list and link initiation against the real daemon.
- [ ] C2. Spec close.
