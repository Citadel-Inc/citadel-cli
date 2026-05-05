# Spec — cli-ssh-keys

| | |
|---|---|
| Status | DRAFT 050506ZMAY26 |
| Authored | 050506ZMAY26 |
| Owner | Bastion (J-3) |
| Carry-forward from | Phase 0 L0 git wire: users need SSH public keys registered for `git@git.src.land`. Server: `internal/api/accountapi/ssh_keys.go`. |

## Daemon HTTP contract

| Method | Path | Handler |
|--------|------|---------|
| GET | `/api/account/ssh-keys` | `HandleSSHKeysList` |
| POST | `/api/account/ssh-keys` | `HandleSSHKeysCreate` |
| DELETE | `/api/account/ssh-keys/{id}` | `HandleSSHKeysDelete` |

**Auth:** JWT (`requireUser`); no separate scope beyond authenticated account.

**List response**

```json
{"keys":[{"id":"uuid","fingerprint":"…","public_key":"…","label":null,"created_at":"…"}, …]}
```

**Create request**

```json
{"public_key":"<ssh-ed25519 AAA… or rsa …>","label":"optional"}
```

- `DisallowUnknownFields()` on decoder — **unknown JSON keys → 400**.
- Duplicate fingerprint → server-dependent error (survey handler — likely conflict / validation).

**Delete**

- UUID `{id}` path segment; wrong user → not found / forbidden per handler.

**Errors:** `db_unavailable`, `list_failed`, **trimmed validation messages** — map via `httputil.WriteJSONError` codes in handler source.

## In scope

**Parent command:** `citadel-cli ssh-key` (see Q-table).

| CLI | HTTP |
|-----|------|
| `ssh-key list` | GET |
| `ssh-key add --key-file PATH` **and/or** `--public-key` string | POST |
| `ssh-key delete <id>` | DELETE |

**UX**

- Reading key material from **`--key-file`** (preferred) or stdin when piped.
- Optional **`--label`** flag mapping to request `label`.

## Out of scope

- **`ssh-keygen`** — users generate keys locally.
- **Deploy keys / per-repo keys** — not in this API surface.

## Decision log

| Q | Proposal | Status |
|---|----------|--------|
| Q1 | Command group name `ssh-key` vs `account ssh-key`? | **Open** — top-level `ssh-key`. |
| Q2 | stdin adds key vs require explicit `-` flag? | **Open** — mirror patterns from other stdin-reading verbs. |

## Acceptance

- A1. Three verbs + httptest matrix (happy, bad JSON, delete missing id).
- A2. Never echo full private key material (only public key surfaces — enforce in review).
- A3. `docs/cli.md`.
- A4. Q-table ratified.
- A5. Optional `CITADEL_TEST_SSH_KEYS_LIVE=1`.
