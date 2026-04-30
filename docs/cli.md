# Citadel CLI — Installation and usage

`citadel` is a command-line client for managing authentication, agent tokens, and MCP tool interactions with the Citadel server. All commands authenticate via Supabase Auth (OAuth/PKCE) and store credentials locally in `~/.config/citadel/config.toml` (mode 0600).

## Installation

### From source (development)

If you have a local checkout:

```bash
cd /path/to/citadel
make build-cli
cp ./citadel-cli /usr/local/bin/citadel
```

### Via `go install` (latest)

```bash
go install github.com/Rethunk-Tech/citadel/cmd/citadel-cli@latest
```

This installs to `~/go/bin/citadel-cli`; add `~/go/bin` to your `PATH` if it is not already there.

### Binary releases (future)

Once v1 is stable, pre-built binaries for linux-amd64, linux-arm64, and darwin-arm64 will be published to GitHub Releases. Check <https://github.com/Rethunk-Tech/citadel/releases/> for availability.

## First-run flow

### Login

```bash
citadel auth login
```

This opens your default browser to Supabase's OAuth authorization endpoint. After you authenticate (GitHub or your configured provider), the browser redirects to a local loopback server running on your machine, which exchanges the authorization code for an access token and refresh token. Both are stored in `~/.config/citadel/config.toml` (mode 0600).

The CLI defaults to server URL `https://api.src.land`; if your server is at a different URL, set the `CITADEL_SERVER_URL` environment variable or edit the config file directly (key: `server_url`).

### Check authentication status

```bash
citadel auth status
```

Prints the authenticated user UUID, access token expiry time, and configured server URL. If not authenticated, prints "not authenticated" and exits 0.

### Logout

```bash
citadel auth logout
```

Removes the authentication session from the config file, preserving the server URL setting for future logins.

## Daily commands

### List agent tokens

```bash
citadel token list --agent <agent-name>
```

Lists all tokens (active and revoked) for the given agent. Columns: token ID, agent name, scopes, created_at, expires_at, revoked_at.

Example:

```bash
citadel token list --agent my-app
```

### Issue a new agent token

```bash
citadel token issue --agent <name> [--scopes <scope>[,<scope>...]] [--expires <duration>]
```

Creates or finds an agent with the given name and issues a new token. Prints the clear-text token exactly once to stdout (with no debug output). Subsequent `citadel token list` calls will show only metadata, not the secret.

Parameters:
- `--agent <name>` (required): Agent name; if the agent does not exist, it is created.
- `--scopes <scope>[,<scope>...]` (optional): Comma-separated list of scopes (e.g., `mcp:read,mcp:write`). Default: no scopes.
- `--expires <duration>` (optional): Token expiry time (e.g., `24h`, `7d`, `no-expiry`). Default: no expiration.

Example:

```bash
$ citadel token issue --agent my-indexer --scopes mcp:read --expires 7d
sb_at_xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx
```

Save this token immediately — it is never displayed again. Store it in your application's environment or credentials file.

### Revoke an agent token

```bash
citadel token revoke <token-id>
```

Sets the `revoked_at` timestamp on the token, deactivating it immediately. Revoked tokens are rejected on every MCP call. This command is idempotent — revoking an already-revoked token is a no-op.

Example:

```bash
citadel token revoke 550e8400-e29b-41d4-a716-446655440000
```

## Server URL configuration

The CLI defaults to `https://api.src.land`. Override it via:

1. **Environment variable**: `CITADEL_SERVER_URL`
2. **Config file**: Edit `~/.config/citadel/config.toml` and set `server_url = "https://your-server.com"`
3. **Command-line flag**: (not yet supported; defer to follow-on CLI enhancements)

Example:

```bash
export CITADEL_SERVER_URL=https://api.internal.example.com
citadel token list --agent my-app
```

## Agent token semantics

For comprehensive token lifecycle documentation, see [docs/agents.md](agents.md). In brief:

- **Tokens are opaque secrets.** Never log them, commit them, or pass them on the command line. Store in environment files (e.g., `.env.local`) or CI secrets with restricted access.
- **Hashing.** The CLI never stores the clear-text token; only the server stores a sha256 hash. Once you close the terminal, you cannot recover the token — you must revoke and issue a new one.
- **Scopes.** Tokens carry a list of permissions (scopes) enforced server-side. The MCP server checks token scopes before allowing tool access.
- **Revocation.** Revoked tokens are rejected immediately; no cache delay.

For hand-rolled token issuance (operator-only, until this CLI shipped), see [docs/agents.md § Token issuance](agents.md#token-issuance-hand-rolled-phase-b).

## Troubleshooting

### "not authenticated" after login

The OAuth flow may have timed out or the browser may have closed. Try `citadel auth login` again.

### Token expires quickly

The default expiration is "no expiry" (tokens live until revoked). If you set `--expires 1h` and saw the token expire quickly, that is expected. Adjust the expiration when issuing the token, or revoke and re-issue with a longer (or infinite) expiration.

### Config file not found

If `~/.config/citadel/` does not exist, `citadel auth login` creates it automatically with mode 0700 (directory) and 0600 (config file). If the file exists but is unreadable, check its permissions with `ls -la ~/.config/citadel/config.toml`.

### "parse error in config"

The config file may be corrupted. Back it up and delete it:

```bash
mv ~/.config/citadel/config.toml ~/.config/citadel/config.toml.bak
citadel auth login
```

This will create a fresh config.
