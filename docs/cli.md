# Citadel CLI — Installation and usage

The **`citadel-cli`** binary is the command-line **client** for authentication, agent tokens, and MCP tool calls against the Citadel **server**. The server binary lives in the [Rethunk-Tech/citadel](https://github.com/Rethunk-Tech/citadel) repository and is named **`citadel`** (HTTP, SSH, MCP); do not confuse the two names on disk. All CLI commands authenticate via Supabase Auth (OAuth/PKCE) and store credentials locally in `~/.config/citadel/config.toml` (mode 0600).

## Installation

### From source (development)

If you have a local checkout:

```bash
cd /path/to/citadel-cli
make build
cp ./citadel-cli /usr/local/bin/citadel-cli
```

### Via `go install` (latest)

```bash
go install github.com/Rethunk-Tech/citadel-cli@latest
```

This installs to `~/go/bin/citadel-cli`; add `~/go/bin` to your `PATH` if it is not already there.

### Binary releases (future)

Once v1 is stable, pre-built binaries for linux-amd64, linux-arm64, and darwin-arm64 will be published to GitHub Releases. Check <https://github.com/Rethunk-Tech/citadel-cli/releases/> for availability.

## First-run flow

### Login

```bash
citadel-cli auth login
```

This opens your default browser to Supabase's OAuth authorization endpoint. After you authenticate (GitHub or your configured provider), the browser redirects to a local loopback server running on your machine, which exchanges the authorization code for an access token and refresh token. Both are stored in `~/.config/citadel/config.toml` (mode 0600).

The CLI defaults to server URL `https://api.src.land`; if your server is at a different URL, set the `CITADEL_SERVER` environment variable or edit the config file directly (key: `server_url`).

### Check authentication status

```bash
citadel-cli auth status
```

Prints the authenticated user UUID, access token expiry time, and configured server URL. If not authenticated, prints "not authenticated" and exits 0.

### Logout

```bash
citadel-cli auth logout
```

Removes the authentication session from the config file, preserving the server URL setting for future logins.

## Shell completion

Generate scripts for your shell (bash, zsh, fish, or PowerShell):

```bash
citadel-cli completion bash   # often: source <(citadel-cli completion bash)
```

Dynamic completion for resource arguments (repos, namespaces, agents, OAuth clients, tokens) uses your stored access token to query list endpoints. Cache layout, TTL, and the `CITADEL_NO_COMPLETION_CACHE` bypass are described under **Configuration** in the repo’s [HUMANS.md](../HUMANS.md#configuration) (not duplicated here).

## Daily commands

### List agent tokens

```bash
citadel-cli token list --agent <agent-name>
```

Lists all tokens (active and revoked) for the given agent. Columns: token ID, agent name, scopes, created_at, expires_at, revoked_at.

Example:

```bash
citadel-cli token list --agent my-app
```

### Issue a new agent token

```bash
citadel-cli token issue --agent <name> [--scopes <scope>[,<scope>...]] [--expires <duration>]
```

Creates or finds an agent with the given name and issues a new token. Prints the clear-text token exactly once to stdout (with no debug output). Subsequent `citadel-cli token list` calls will show only metadata, not the secret.

Parameters:
- `--agent <name>` (required): Agent name; if the agent does not exist, it is created.
- `--scopes <scope>[,<scope>...]` (optional): Comma-separated list of scopes (e.g., `mcp:read,mcp:write`). Default: no scopes.
- `--expires <duration>` (optional): Token expiry time (e.g., `24h`, `7d`, `no-expiry`). Default: no expiration.

Example:

```bash
$ citadel-cli token issue --agent my-indexer --scopes mcp:read --expires 7d
sb_at_xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx
```

Save this token immediately — it is never displayed again. Store it in your application's environment or credentials file.

### Revoke an agent token

```bash
citadel-cli token revoke <token-id>
```

Sets the `revoked_at` timestamp on the token, deactivating it immediately. Revoked tokens are rejected on every MCP call. This command is idempotent — revoking an already-revoked token is a no-op.

Example:

```bash
citadel-cli token revoke 550e8400-e29b-41d4-a716-446655440000
```

## Org invitations

Org invitations use the daemon **`orgsmembersapi`** routes behind your normal CLI session (`citadel-cli auth login`). You need **`members:read`** to list invitations for an org and **`members:write`** to create or revoke them; **`accept`** uses your session JWT plus the invitation token.

### List invitations pending for your account

```bash
citadel-cli org invitation pending
citadel-cli org invitation pending --output json
```

### List invitations for one org

```bash
citadel-cli org invitation list <org-slug>
```

### Create an invitation

Provide **`--email`** and/or **`--slug`** (the invitee's public user namespace slug). Repeat **`--permission`** for each grant, or pass comma-separated permission atoms that match the server's allow-list (for example `members:read`, `members:write` — confirm on your server).

```bash
citadel-cli org invitation create myorg --email colleague@example.com --permission members:read
citadel-cli org invitation create myorg --slug publichandle --permission members:read,members:write
```

On a TTY, if you omit both **`--email`** and **`--slug`**, the CLI prompts for an email.

### Revoke

```bash
citadel-cli org invitation revoke <org-slug> <invite-id>
```

### Accept

Pass the token from the invitation link as an argument, or use **`--token-file`** so the secret is not stored in your shell history.

```bash
citadel-cli org invitation accept <token>
citadel-cli org invitation accept --token-file ~/invite-token.txt
```

Treat invitation tokens like passwords.

### Live integration test

Opt-in: set **`CITADEL_TEST_ORG_INVITATIONS_LIVE=1`** together with **`CITADEL_ACCESS_TOKEN`** and optional **`CITADEL_SERVER`** (see `cmd/org_invitation_live_test.go`). CI skips when unset.

## SSH public keys (Git)

Register SSH **public** keys for your account so Git SSH endpoints can authorize pushes and pulls. The CLI never uploads private keys and rejects PEM-style private key blobs.

### List keys

```bash
citadel-cli ssh-key list
citadel-cli ssh-key list --output json
```

### Add a key

Preferred: point at a `.pub` file.

```bash
citadel-cli ssh-key add --key-file ~/.ssh/id_ed25519.pub --label work-laptop
citadel-cli ssh-key add --public-key "$(cat ~/.ssh/id_ed25519.pub)"
```

When stdin is not a TTY, you may pipe a one-line public key. Use **`--key-file -`** to read from stdin on a TTY.

### Delete a key

```bash
citadel-cli ssh-key delete <key-id>
```

### Live integration test

Opt-in: **`CITADEL_TEST_SSH_KEYS_LIVE=1`** with **`CITADEL_ACCESS_TOKEN`** (see `cmd/ssh_key_live_test.go`). CI skips when unset.

## Audit sessions

Inspect grouped audit **sessions** for a namespace (distinct from **`citadel-cli audit list`**, which lists raw events with cursor pagination).

### List sessions

**`--ns`** or **`--namespace` / `-n`** selects the namespace (maps to the daemon `ns` query parameter). Optional **`--since`** accepts durations (`1h`, `7d`, …) or RFC3339. Pagination uses **`--limit`** and **`--offset`** (not `--cursor`).

```bash
citadel-cli audit sessions list --ns myorg
citadel-cli audit sessions list -n myorg --since 7d --limit 50 --offset 0 --output json
```

### Show session detail

Prints the drill-down JSON from the server (including operator-only fields only when the API returns them).

```bash
citadel-cli audit sessions show <session-id>
citadel-cli audit sessions show <session-id> --output yaml
```

### Live integration test

Opt-in: **`CITADEL_TEST_AUDIT_SESSIONS_LIVE=1`** with **`CITADEL_ACCESS_TOKEN`** and **`CITADEL_TEST_AUDIT_SESSIONS_NS`** set to a namespace you can read audit for (see `cmd/audit_sessions_live_test.go`). CI skips when unset.

## Server URL configuration

The CLI defaults to `https://api.src.land`. Override it via:

1. **`--server` flag** (highest precedence; persistent on the root command).
2. **Environment variable**: `CITADEL_SERVER`.
3. **Config file**: Edit `~/.config/citadel/config.toml` and set `server_url = "https://your-server.com"`.

Example:

```bash
citadel-cli --server http://localhost:8080 token list --agent my-app
export CITADEL_SERVER=https://api.internal.example.com
citadel-cli token list --agent my-app
```

## MCP tool calls

The `citadel-cli mcp` group speaks the [Streamable HTTP MCP protocol](https://modelcontextprotocol.io/specification/2025-11-25/server) against the Citadel MCP server. The server URL defaults to `https://mcp.src.land/mcp` (resolved from `--server` / `CITADEL_SERVER`; `api.src.land` is auto-swapped to `mcp.src.land`).

Authentication uses your **Supabase JWT** from `citadel-cli auth login` by default. Override with `--token` or `CITADEL_AGENT_TOKEN` for agent / CI workflows — both work because the MCP server's bearer-validator tries JWTs first and falls through to opaque agent tokens.

### List tools

```bash
$ citadel-cli mcp tools
get_namespace	Look up a namespace by slug or path
kg_find_symbol	Search the knowledge graph for symbols matching a query
kg_list_files	List indexed files in a namespace
kg_walk	Walk symbol edges from a starting symbol
```

### Call a tool

```bash
$ citadel-cli mcp call get_namespace --arg path=damon
{
  "slug": "damon",
  "kind": "user",
  "owner_user_id": "..."
}
```

Use `--json` for the raw JSON-RPC `tools/call` response (useful for scripting):

```bash
$ citadel-cli mcp call get_namespace --arg path=damon --json
```

### Argument coercion

`--arg key=value` coerces the value automatically. Use `--arg-string key=value` to opt out for a single argument.

| Input form           | Coerced type | Example                          |
|----------------------|--------------|----------------------------------|
| `key=hello`          | string       | `"hello"`                        |
| `key=true` / `false` | bool         | `true` / `false`                 |
| `key=5`              | int64        | `5`                              |
| `key=-7`             | int64        | `-7`                             |
| `key=07823`          | string       | `"07823"` (leading zero kept)    |
| `key=1.5`            | float64      | `1.5`                            |
| `key=a,b,c`          | array        | `["a","b","c"]`                  |
| `key=1,2,3`          | array of int | `[1, 2, 3]`                      |
| `key=1,foo,true`     | mixed array  | `[1, "foo", true]`               |

Edge cases that fall through to string: `.5`, `5.`, `1.2.3`, anything with non-digit non-dot non-comma characters.

### Flags

- `--server <url>` (root, persistent) — override server URL.
- `--token <bearer>` (mcp group, persistent) — override default JWT.
- `--timeout <secs>` (mcp group, persistent) — per-call HTTP timeout (default 60).
- `--arg key=value` (call) — repeatable; coerced.
- `--arg-string key=value` (call) — repeatable; raw string.
- `--json` (call) — emit raw JSON-RPC result instead of pretty-printed text content.

### Exit codes

| Code | Meaning                                                                |
|------|------------------------------------------------------------------------|
| 0    | Success.                                                               |
| 1    | Local error: bad flags, no token, transport failure, server JSON-RPC error. |
| 2    | Tool returned `isError: true` (the call reached the tool; the tool failed).|

### Phase 0 operator cookbook (HTTPS MCP)

Agents should configure the IDE or runtime to talk to **`https://mcp.src.land/mcp`** (or your deployment’s MCP URL) directly. For **operators** — debugging, runbooks, CI smoke — `citadel-cli mcp call` hits the **same** Streamable HTTP MCP surface as agents; there is no separate stdio MCP in this client ([parked specs](../specs/parked/README.md)).

**Discover names on your server** (tool lists drift with Citadel releases):

```bash
citadel-cli mcp tools
```

Below, treat **`<tool>`** as a name from that list. Argument shapes mirror the Proof-of-Life dossier **appendix K** tool groups (`namespace.*`, `repo.*`, `issue.*`, `project.*`, `kg.*`, `agent.*`, `audit.*`, `key.*`) — **wire names differ** (often `snake_case`); always confirm with `mcp tools`.

| Intent | Pattern |
|--------|---------|
| Resolve namespace / org | `citadel-cli mcp call <tool> --arg path=<slug>` — typical discovery tool (historically `get_namespace`; confirm via `mcp tools`). |
| Knowledge graph | Tools such as **`kg_find_symbol`**, **`kg_list_files`**, **`kg_walk`** (examples; verify list). Pass repo/namespace args your server’s schema expects, often `--arg namespace_path=…` or `--arg-string` for opaque IDs. |
| Project-as-graph | Project tools accept **`project_path`** or **`namespace_path`**-style args per server registration — use `--json` when responses are large. |
| Issues | When issue MCP tools are enabled, use **`issue.list` / `issue.get`**-style names from `mcp tools` (not yet mirrored by dedicated `citadel-cli issue` verbs — see spec **`cli-issue-pr`**). |
| Audit | **`audit.list`** / session tools — align filters (`namespace_path`, `since`) with MCP schema; parity with `citadel-cli audit list` when both exist. |
| SSH keys | **`key.list` / `key.add` / `key.delete`** per appendix K — same semantics as future `citadel-cli ssh-key` REST wrappers. |

**Recipes**

```bash
# Namespace lookup (replace get_namespace if your server lists a different name)
citadel-cli mcp call get_namespace --arg path=Rethunk-Tech

# Raw JSON-RPC envelope (scripting)
citadel-cli mcp call get_namespace --arg path=Rethunk-Tech --json

# Agent with explicit bearer (CI / rotate-token output)
citadel-cli mcp --token "$CITADEL_AGENT_TOKEN" tools
```

**REST parity:** Most MCP tools have equivalent **`GET/POST /api/...`** handlers on the Citadel API host (`api.src.land`). Prefer MCP for agent-shaped workflows; prefer **`citadel-cli <verb>`** when we ship first-class commands (repos, namespaces, audit events) — fewer typed arguments to assemble by hand.

### Auth failures

A 401 / `-32001 unauthorized` from the server prints:

```
unauthorized: run `citadel-cli auth login` to refresh your session, or pass --token / set CITADEL_AGENT_TOKEN
```

The CLI does **not** auto-refresh tokens. Re-authenticate explicitly.

## Agent token semantics

For comprehensive token lifecycle documentation, see [Rethunk-Tech/citadel docs/agents.md](https://github.com/Rethunk-Tech/citadel/blob/main/docs/agents.md). In brief:

- **Tokens are opaque secrets.** Never log them, commit them, or pass them on the command line. Store in environment files (e.g., `.env.local`) or CI secrets with restricted access.
- **Hashing.** The CLI never stores the clear-text token; only the server stores a sha256 hash. Once you close the terminal, you cannot recover the token — you must revoke and issue a new one.
- **Scopes.** Tokens carry a list of permissions (scopes) enforced server-side. The MCP server checks token scopes before allowing tool access.
- **Revocation.** Revoked tokens are rejected immediately; no cache delay.

## Troubleshooting

### "not authenticated" after login

The OAuth flow may have timed out or the browser may have closed. Try `citadel-cli auth login` again.

### Token expires quickly

The default expiration is "no expiry" (tokens live until revoked). If you set `--expires 1h` and saw the token expire quickly, that is expected. Adjust the expiration when issuing the token, or revoke and re-issue with a longer (or infinite) expiration.

### Config file not found

If `~/.config/citadel/` does not exist, `citadel-cli auth login` creates it automatically with mode 0700 (directory) and 0600 (config file). If the file exists but is unreadable, check its permissions with `ls -la ~/.config/citadel/config.toml`.

### "parse error in config"

The config file may be corrupted. Back it up and delete it:

```bash
mv ~/.config/citadel/config.toml ~/.config/citadel/config.toml.bak
citadel-cli auth login
```

This will create a fresh config.

## Distribution

The CLI binary is built by the GitHub Actions release workflow on every annotated tag matching `v*`, producing static binaries for `linux-amd64`, `linux-arm64`, and `darwin-arm64`. Channels:

- **GitHub Releases (canonical, today).** Each tag publishes a release at `github.com/Rethunk-Tech/citadel-cli/releases/tag/<tag>` with the three binaries + a `SHA256SUMS` file. Manual download:

  ```bash
  # Replace v0.x.y with the latest tag.
  curl -L -o citadel-cli-linux-amd64 \
    https://github.com/Rethunk-Tech/citadel-cli/releases/download/v0.x.y/citadel-cli-linux-amd64
  curl -L -o SHA256SUMS \
    https://github.com/Rethunk-Tech/citadel-cli/releases/download/v0.x.y/SHA256SUMS
  sha256sum -c SHA256SUMS --ignore-missing
  chmod +x citadel-cli-linux-amd64
  sudo mv citadel-cli-linux-amd64 /usr/local/bin/citadel-cli
  ```

- **Homebrew tap (deferred).** Suggested formula path `rethunk-tech/tap/citadel-cli`. Land when a second non-operator user adopts the CLI; until then, GH Releases is sufficient.

- **Static mirror at `cli.src.land` (deferred).** Operator-managed mirror behind Caddy. Only worth standing up if GH Releases is unavailable to a target audience (corporate networks blocking github.com download paths). Not currently planned.

### Versioning

Until v1.0 the **citadel-cli** release tags track the **citadel** server binary releases; both bump together on every release. After v1.0 the CLI versions independently — server-side compatibility is JWT/JWKS-based, not version-pinned.

### Verifying a download

```bash
# After downloading binary + SHA256SUMS for the same tag.
sha256sum -c SHA256SUMS --ignore-missing
# Expect: citadel-linux-amd64: OK
```

If verification fails, do not run the binary. Re-download from a fresh session and re-verify.
