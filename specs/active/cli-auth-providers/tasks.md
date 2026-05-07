# Tasks — cli-auth-providers

Status: DRAFT

## P0

- [ ] A1. Implement `auth provider list` against `GET /api/auth/providers`.
- [ ] A2. Add provider ID completion support for auth-provider verbs.
- [ ] A3. Add tests covering command wiring and request/response handling.

## P1

- [ ] B1. Add `auth provider link <provider>` with browser-open default and structured output option.
- [ ] B2. Add `auth provider unlink <provider>` with confirmation guard and direct daemon-error surfacing.
- [ ] B3. Document auth-provider discovery, link, and unlink workflows.
- [ ] B4. `make verify` green.

## P2

- [ ] C1. Live smoke provider list and link initiation against the real daemon.
- [ ] C2. Spec close.
