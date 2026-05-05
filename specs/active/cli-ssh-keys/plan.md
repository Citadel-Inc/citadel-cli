# Plan — cli-ssh-keys

## ORIENT

- **Server:** `internal/api/accountapi/ssh_keys.go` — structs **`sshKeyRow`**, **`sshKeysListResponse`**, **`sshKeyCreateRequest`** (exact JSON tags).
- **CLI auth:** same `newAPIClient` as other account routes (JWT from `auth login`).

## RECON

- Read **`HandleSSHKeysCreate`** validation branches (empty key, malformed SSH public key parsing via `golang.org/x/crypto/ssh`) — document user-visible errors for errmap.

## Implementation sketch

- **`cmd/ssh_key.go`** — `sshKeyCmd`; delete takes UUID string; optional **`ValidArgsFunction`** completion for IDs from list cache **P2**.

## Risks

- **Windows paths** for `--key-file` — use stdlib `os.ReadFile`; no shell expansion required.
