# Plan — cli-ssh-keys

## ORIENT

- Handlers: `accountapi/ssh_keys.go` — list/create/delete.
- Client: same `newAPIClient` JWT path as other account routes.

## RECON

Read request/response structs from `ssh_keys.go` for JSON field names (`label`, `public_key`, etc.).

## Implementation sketch

- New `cmd/ssh_key.go` (or `sshkey.go` per package naming).
- `delete` requires UUID arg — completion hook optional P2.

## Risks

- **Fingerprint collision errors** — surface server message verbatim via errmap.
