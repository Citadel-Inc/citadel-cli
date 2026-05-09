# Citadel CLI — Installation and usage

The **`citadel-cli`** binary is the command-line **client** for authentication, agent tokens, and MCP tool calls against the Citadel **server**. The server binary lives in the [Rethunk-Tech/citadel](https://github.com/Rethunk-Tech/citadel) repository and is named **`citadel`** (HTTP, SSH, MCP); do not confuse the two names on disk. The browser login flow brokers through Citadel's OAuth 2.1 endpoints, then stores a Citadel-issued **agent token** locally in `~/.config/citadel/config.toml` (mode 0600).

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

### Binary releases

Pre-built binaries for linux-amd64, linux-arm64, darwin-arm64, and windows-amd64 are published to GitHub Releases on every `v*` tag. Check <https://github.com/Rethunk-Tech/citadel-cli/releases/> for the latest release.

## First-run flow

### Login

```bash
citadel-cli auth login
```

This opens your default browser to Citadel's OAuth authorization endpoint (`/api/oauth/authorize`). After you authenticate, the browser redirects to a local loopback server running on your machine; the CLI exchanges the authorization code against `/api/oauth/token`, then immediately mints and stores a long-lived Citadel **agent token** for `citadel-cli@<hostname>`. The short-lived OAuth JWT is discarded after bootstrap.

The CLI defaults to server URL `https://mcp.src.land`; if your server is at a different URL, set the `CITADEL_SERVER` environment variable or edit the config file directly (key: `server_url`).

### Headless / CI bootstrap

When a browser is unavailable, persist a Supabase JWT directly and let the CLI
upgrade it to a Citadel agent token on a later command when the server is
reachable:

```bash
citadel-cli auth set-token --token "$JWT"
echo "$JWT" | citadel-cli auth set-token
CITADEL_ACCESS_TOKEN="$JWT" citadel-cli auth set-token
```

Use this path for CI, SSH-only hosts, containers, or other non-interactive
environments where `citadel-cli auth login` is unavailable.

### Check authentication status

```bash
citadel-cli auth status
```

Prints the bound agent name + id, token expiry time, authenticated user UUID when known, and configured server URL. If not authenticated, prints "not authenticated" and exits 0.

### Logout

```bash
citadel-cli auth logout
```

Removes the authentication session from the config file, preserving the server URL setting for future logins.

### OAuth provider discovery and linking

The daemon exposes a public provider registry plus authenticated link/unlink flows for third-party identities.

```bash
citadel-cli auth provider list
citadel-cli auth provider list --output json

citadel-cli auth provider link github
citadel-cli auth provider link github --json   # print redirect_url instead of opening a browser

citadel-cli auth provider unlink github        # typed confirmation
citadel-cli auth provider unlink github --yes --json
```

`auth provider list` does **not** require a saved session; `link` and `unlink` do. `link` opens the returned Supabase authorize URL in your browser by default and still prints the URL so headless users can copy-paste it elsewhere. `unlink` uses the daemon's guardrails directly, including `last_provider_blocked` when removing the last non-email provider would strand the account.

## Shell completion

Generate scripts for your shell (bash, zsh, fish, or PowerShell):

```bash
citadel-cli completion bash   # often: source <(citadel-cli completion bash)
```

Dynamic completion for most resource arguments (repos, namespaces, agents, OAuth clients, tokens) uses your stored bearer session to query list endpoints. Auth-provider completion resolves from the public provider registry. Cache layout, TTL, and the `CITADEL_NO_COMPLETION_CACHE` bypass are described under **Configuration** in the repo’s [HUMANS.md](../HUMANS.md#configuration) (not duplicated here).

## Daily commands

### Global search (namespaces & repositories)

Most CLI commands expect a saved session (`citadel-cli auth login`). Search does too: Citadel can attach searches to an identity, enforce per-user rate limits, and protect shared quality of service.

```bash
citadel-cli search "my query"
citadel-cli search "acme" --public
citadel-cli search "foo" --scope repos --limit 15 --output json
```

By default, search uses **`scope=namespaces`** (namespaces you own or belong to). Pass **`--public`** to opt into broader discovery across unrelated namespaces (**`scope=all`**, still authenticated). Use **`--scope`** to override explicitly (`namespaces`, `repos`, or `all`). Queries must be at least two characters.

Repository-scoped knowledge-graph search continues to live under **`citadel-cli kg search`** (different API).

### Project graph (pins, walks, edges)

Inspect or mutate the Citadel **project graph** for a namespace path (flat `org/repo` or nested `org/project/repo`). All verbs require **`citadel-cli auth login`**.

Read-heavy outputs (`walk`, `neighbors`) default to a short human preview; pass **`--output json`** for the full JSON payload.

```bash
citadel-cli project pin-chain org/repo
citadel-cli project walk org/repo --kind repo --output json
citadel-cli project neighbors org/repo --kind repo --direction incoming
citadel-cli project status rollup org/repo
citadel-cli project status drilldown org/repo
citadel-cli project edge add org/repo \
  --from-namespace-id <uuid> --from-kind repo --to-kind repo \
  --edge-type pins --output json
citadel-cli project edge delete org/repo <edge-uuid> --yes
citadel-cli project reindex org/repo --yes
```

HTTP **404** responses may indicate RBAC denial (`projectgraph:read` / `projectgraph:manage`) rather than a wrong path — verify grants before assuming the slug is invalid.

Opt-in live smoke (requires **`CITADEL_TEST_PROJECTGRAPH_LIVE=1`**, **`CITADEL_ACCESS_TOKEN`**, and **`CITADEL_TEST_PROJECTGRAPH_SLUG`**):

```bash
CITADEL_TEST_PROJECTGRAPH_LIVE=1 \
CITADEL_TEST_PROJECTGRAPH_SLUG=org/repo \
go test ./cmd -run TestLiveProjectGraph -count=1
```

### Issues

Issue workflows target any Citadel namespace path with **`-R <ns/path>`**. For
repository namespaces, omission still reuses `CITADEL_REPO` and CWD-origin
inference; for non-repo namespaces, pass `-R` explicitly.

```bash
citadel-cli issue list -R acme/demo
citadel-cli issue list -R acme/demo --state all --label bug --assignee alice --output json
citadel-cli issue view -R acme/demo 42
citadel-cli issue view -R acme/demo 42 --web
citadel-cli issue create -R acme/demo --title "Ship it" --body "Ready for rollout"
citadel-cli issue create -R acme/demo --title "Ship it" --milestone <milestone-uuid>
citadel-cli issue comment -R acme/demo 42 --body "Looks good"
citadel-cli issue close -R acme/demo 42
citadel-cli issue reopen -R acme/demo 42
citadel-cli issue label -R acme/demo 42 --add bug --remove triage
citadel-cli issue close-refs -R acme/demo 42 --output json
```

`issue list` supports the standard cursor pagination flags (`--limit`,
`--cursor`, `--all`) and `--output json|yaml|ndjson|csv|table`.

`issue view` and `issue close-refs` accept `--output json|yaml|table`. `issue
view --web` opens the canonical browser route for that namespace issue.

When `issue create` or `issue comment` omit `--body`, the CLI reads piped stdin
first; on a TTY it falls back to `$EDITOR`.

### Issue milestones

Milestones let you group issues within any Citadel namespace path.

```bash
citadel-cli issue milestone list -R acme/demo
citadel-cli issue milestone list -R acme/demo --state all --output json
citadel-cli issue milestone view -R acme/demo <milestone-uuid>
citadel-cli issue milestone create -R acme/demo --title "v1.0" --description "first release" --due-on 2026-06-01
citadel-cli issue milestone edit -R acme/demo <milestone-uuid> --state closed
citadel-cli issue milestone delete -R acme/demo <milestone-uuid> --yes
```

- `list` filters client-side by `--state open|closed|all` because the current
  server endpoint returns the full milestone set for the namespace.
- `view`, `edit`, and `delete` complete milestone UUIDs dynamically from the
  selected namespace path.
- `create` and `edit` accept `--due-on YYYY-MM-DD`. Passing `--due-on=`
  to `edit` clears the due date.
- `delete` requires typed confirmation of the milestone title unless `--yes`
  is set. `--dry-run` prints the DELETE route without executing it.

### Live integration test

Opt-in: **`CITADEL_TEST_ISSUES_LIVE=1`** with **`CITADEL_TEST_ISSUES_NS`** and
**`CITADEL_ACCESS_TOKEN`**.

```bash
CITADEL_TEST_ISSUES_LIVE=1 \
CITADEL_TEST_ISSUES_NS=acme/demo \
go test ./cmd -run TestLiveIssues_roundTrip_optIn -count=1
```

### Repositories via system git

`citadel-cli repo clone|push|pull` are thin wrappers around the on-system `git`
binary. They use the SSH remote returned by Citadel (or a canonical SSH
fallback) and keep normal git semantics instead of re-implementing
clone/push/pull in the CLI.

```bash
citadel-cli repo clone myorg/myrepo
citadel-cli repo clone myorg/myrepo ./local-dir
citadel-cli repo push
citadel-cli repo push --remote upstream
citadel-cli repo push --create
citadel-cli repo pull
```

- `repo clone <ns>/<repo> [local-dir]` clones the SSH remote for that repo and
  prints the local directory path on success.
- `repo push` / `repo pull` run in the current checkout. With no explicit repo
  path, the target comes from `-R`, `CITADEL_REPO`, or `git remote get-url`.
- `repo push` prompts to create the remote repository when Citadel does not know
  that repo yet; pass `--create` to skip the prompt in CI or other
  non-interactive sessions.
- `--remote <name>` applies when the repo path is inferred from the current
  checkout rather than passed explicitly.

### Webhooks

Webhook management follows Citadel's namespace-scoped API, so the CLI nests it
under the existing parent resources:

```bash
citadel-cli repo webhook list -R acme/demo
citadel-cli repo webhook create -R acme/demo --url https://hooks.example.test/in --events issue.opened,comment.created
citadel-cli repo webhook get -R acme/demo <webhook-id>
citadel-cli repo webhook delete -R acme/demo <webhook-id> --dry-run

citadel-cli namespace webhook list acme
citadel-cli namespace webhook create acme --url https://hooks.example.test/in --events issue.opened --include-descendants
citadel-cli namespace webhook get acme <webhook-id>
citadel-cli namespace webhook delete acme <webhook-id>
```

- Repo webhooks target the repo namespace selected by `-R`, `CITADEL_REPO`, or
  current-checkout inference.
- Namespace webhooks take the namespace path as a positional argument.
- `create` requires `--url` and `--events`. Citadel generates the webhook secret
  server-side and returns the cleartext secret once; save it immediately.
- Supported event kinds currently match the server allow-list:
  `comment.created`, `comment.edited`, `issue.assigned`, `issue.closed`,
  `issue.labeled`, `issue.opened`, `issue.reopened`, `issue.unassigned`, and
  `issue.unlabeled`.
- `--include-descendants` applies to namespace webhooks only.
- `get` is implemented client-side by listing hooks and filtering by ID because
  the current API does not expose a dedicated GET-by-ID route.
- Citadel now exposes a server-side webhook test/ping endpoint; `citadel-cli`
  does not yet wrap it as a `webhook test` command.

### Agents

Manage the agents registered to your account. An agent is a named identity that
holds one or more long-lived tokens for headless access (CI, scripts, other
services).

```bash
citadel-cli agent list
citadel-cli agent create <name> [--org <slug>] [--description "..."] [--output json]
citadel-cli agent get <name>
citadel-cli agent delete <name> [--yes]
citadel-cli agent rotate-token <name>
```

#### `agent create`

```bash
citadel-cli agent create <name> [--org <slug>] [--description "..."]
```

Registers a new agent in the authenticated user's namespace (or an org namespace
with `--org <slug>`) and prints the agent's **one-time initial token**. Save this
token immediately — it will not be displayed again. Subsequent calls to
`agent get` or `agent list` show metadata only.

`--output json` emits a machine-readable object:

```json
{ "id": "...", "name": "...", "token": "...", "created_at": "..." }
```

Error paths:

| Server response | CLI message |
|---|---|
| 409 Conflict | `agent name already taken` |
| 403 Forbidden | `insufficient permission` |
| 422 Unprocessable | field-level validation hint |

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

### Deploy tokens

Deploy tokens are namespace-scoped credentials for CI/CD and automation. Repo
tokens and org-namespace tokens share the same backend route family; the CLI
exposes them under the matching parent resources.

```bash
citadel-cli repo deploy-token list -R acme/demo
citadel-cli repo deploy-token create -R acme/demo --name ci --expires 24h
citadel-cli repo deploy-token revoke -R acme/demo <token-id>

citadel-cli namespace deploy-token list acme
citadel-cli namespace deploy-token create acme --name release-bot --expires 24h
citadel-cli namespace deploy-token revoke acme <token-id>
```

List verbs support the standard `--limit`, `--cursor`, `--all`, `--watch`, and
`--output json|yaml|ndjson|csv|table` flags.

Create prints the one-time cleartext token to stdout in human mode and includes
it in `--output json`. Revoke supports `--dry-run`.

## Org members

Manage members of an org namespace. Requires **`members:read`** to list and **`members:write`** to modify or remove. The org owner cannot have their permissions changed or be removed.

### List members

```bash
citadel-cli org member list <org-slug>
citadel-cli org member list myorg --output json
citadel-cli org member list myorg --all --output ndjson
```

Columns: `USER_ID` (first 8 chars), `SLUG`, `DISPLAY_NAME`, `ROLE` (`owner` or `member`), `PERMISSIONS`, `JOINED`.

### Replace a member's permission set

`<member>` accepts a user UUID or a user slug (resolved via the member list).

```bash
citadel-cli org member set-permissions myorg alice --permission members:read,code:read
citadel-cli org member set-permissions myorg alice --permission code:read --permission code:write
citadel-cli org member set-permissions myorg alice   # clears all grants
```

### Remove a member

Prompts for confirmation on a TTY; suppress with `--yes`.

```bash
citadel-cli org member remove myorg alice
citadel-cli org member remove myorg alice --yes
```

Returns an error (`self_removal_lockout`) if you are the last `members:write` holder, preventing lockout.

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

## Account security (passkeys and devices)

Manage **WebAuthn passkeys** (list, rename, delete registered credentials) and **signed-in devices** (list, revoke). Passkey **enrolment** (begin/finish) is browser-only — use the Citadel web app to add new passkeys.

### Passkeys

```bash
citadel-cli account passkey list
citadel-cli account passkey list --output json
citadel-cli account passkey rename <id> --name "Work laptop"
citadel-cli account passkey delete <id>
```

### Devices

Revoking a device calls the daemon’s DELETE endpoint; the server may require **recent MFA** (HTTP **412**). Complete verification in a **logged-in browser session** for the same account, then retry the CLI command.

```bash
citadel-cli account device list
citadel-cli account device revoke <device-id>
```

### `--debug-http` and secrets

Do not paste **`--debug-http`** traces into tickets: Authorization is redacted, but never assume other fields are safe to publish.

### Live integration test

Opt-in: set **`CITADEL_TEST_ACCOUNT_SECURITY_LIVE=1`** with **`CITADEL_ACCESS_TOKEN`** (and optional **`CITADEL_SERVER`**). See `cmd/account_live_test.go`. CI skips when unset.

## Knowledge graph (HTTP)

Query the Citadel **knowledge-graph REST API** (`kgapi`) with your saved session bearer (normally the agent token minted by `auth login`). Commands follow **`cli-cwd-context`**: pass **`-R namespace/repo`**, set **`CITADEL_REPO`**, or rely on **`git remote origin`** when the remote host is recognised.

### Cross-namespace search

```bash
citadel-cli kg search "auth middleware"
citadel-cli kg search --query "TODO" --mode fts --path-prefix internal/
```

### Namespace-scoped verbs

```bash
citadel-cli kg symbols --q Handler -R myorg/myrepo
citadel-cli kg files -R myorg/myrepo --path-prefix pkg/
citadel-cli kg fulltext --q "panic(" -R myorg/myrepo
citadel-cli kg walk --seed-id <symbol-uuid> -R myorg/myrepo --depth 2
citadel-cli kg diff -R myorg/myrepo --from-ref main --to-ref topic-branch
citadel-cli kg impact myorg/myrepo MyFunc   # symbol name or UUID
```

**`kg impact`** uses **`GET /api/namespaces/{namespace}/kg/impact`** (legacy **`/kg/{owner}/impact`** is no longer called).

Machine output: **`--output json`** or **`yaml`** on each verb.

### Live integration test

Opt-in: **`CITADEL_TEST_KG_EXTENDED_LIVE=1`** with **`CITADEL_ACCESS_TOKEN`** — see `cmd/kg_extended_live_test.go`. CI skips when unset.

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

The CLI defaults to `https://mcp.src.land`. Override it via:

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

Authentication uses your saved Citadel CLI session by default (typically an opaque **agent token** minted by `citadel-cli auth login`). Override with `--token` or `CITADEL_AGENT_TOKEN` for explicit agent / CI workflows; the server accepts both OAuth JWTs and opaque agent tokens where the route supports them.

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
- `--token <bearer>` (mcp group, persistent) — override the stored session bearer.
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
| Issues | When issue MCP tools are enabled, use **`issue_*`** names from `mcp tools` (for example `issue_list`, `issue_get`, `issue_comment`) until first-class CLI issue verbs land; tracked under active spec **`cli-issues`**. |
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

**REST parity:** Today the live production CLI/API surface is exposed from `mcp.src.land` as well as `/mcp`; older `api.src.land` examples are stale for OAuth, agent, search, audit, and OAuth-client routes. Prefer MCP for agent-shaped workflows; prefer **`citadel-cli <verb>`** when we ship first-class commands (repos, namespaces, audit events) — fewer typed arguments to assemble by hand.

### Auth failures

A 401 / `-32001 unauthorized` from the server prints:

```
unauthorized: run `citadel-cli auth login` to refresh your session, or pass --token / set CITADEL_AGENT_TOKEN
```

The CLI retries once on **401** by rotating the stored agent token when it has an agent binding. If that one-shot refresh also fails, re-authenticate explicitly.

## Agent token semantics

For comprehensive token lifecycle documentation, see [Rethunk-Tech/citadel docs/agents.md](https://github.com/Rethunk-Tech/citadel/blob/main/docs/agents.md). In brief:

- **Tokens are opaque secrets.** Never log them, commit them, or pass them on the command line. Store in environment files (e.g., `.env.local`) or CI secrets with restricted access.
- **Hashing.** The CLI never stores the clear-text token; only the server stores a sha256 hash. Once you close the terminal, you cannot recover the token — you must revoke and issue a new one.
- **Scopes.** Tokens carry a list of permissions (scopes) enforced server-side. The MCP server checks token scopes before allowing tool access.
- **Revocation.** Revoked tokens are rejected immediately; no cache delay.

## Namespace profile

`citadel-cli namespace profile` provides read-only access to namespace identity metadata — the same profile data shown in the Citadel web UI.

### Get a namespace profile

```bash
# Human-readable summary
citadel-cli namespace profile get myorg

# Structured output
citadel-cli namespace profile get myorg --output json
citadel-cli namespace profile get myorg --output yaml
```

**Fields rendered:** Slug, Kind, Visibility, Display name, Legal entity name, Bio, Location, Website, Email, Pronouns, Company, Timezone, Sponsor URL, Social links, Repo/member stats, Billing email (owner-only), Verified domains (owner-only).

**Visibility gate:** Private namespaces return a not-found error for non-owners — the server does not leak the existence of private namespaces. If you are the owner and receive a not-found, confirm the slug and try again after re-authenticating.

```bash
# Short alias
citadel-cli ns profile get myorg
```

> **Note:** Profile editing is a settings-panel concern (no `gh profile edit` exists) and is out of scope for `citadel-cli`.

## Notification inbox

`citadel-cli notification` manages your personal Citadel notification inbox — the same feed surfaced in the web UI. All subcommands require authentication (`citadel-cli auth login`).

### List notifications

```bash
# All recent notifications (default limit 25)
citadel-cli notification list

# Unread only
citadel-cli notification list --unread

# Paginate
citadel-cli notification list --limit 50 --cursor <cursor>

# Machine-readable
citadel-cli notification list --output json
citadel-cli notification list --output csv
```

Columns: **ID**, **Kind**, **Summary**, **Namespace**, **Read at**, **Created at**.

### Mark as read

```bash
# Mark a single notification read
citadel-cli notification read <notification-id>

# Mark all notifications read at once
citadel-cli notification read-all
```

### Unread count

```bash
# Human-readable
citadel-cli notification unread-count

# JSON (for scripting / badges)
citadel-cli notification unread-count --output json
```

### Notification preferences

```bash
# View current preferences
citadel-cli notification prefs get
citadel-cli notification prefs get --output json

# Change email-digest cadence (never | daily | weekly)
citadel-cli notification prefs set --email-digest weekly

# Enable / disable individual notification kinds
citadel-cli notification prefs set --enable issue.comment --disable repo.push

# Combine in one call
citadel-cli notification prefs set --email-digest daily --enable issue.comment
```

`--enable` and `--disable` accept the `kind` string from the prefs list (e.g. `issue.comment`, `repo.push`). Multiple flags may be supplied. The API applies the patch atomically — you only send what you want to change.

## Pull requests

`citadel-cli pr` manages pull requests against the Citadel forge. All subcommands require authentication and accept `-R org/repo` (or `-R org/project/repo`) to identify the repository namespace.

### List pull requests

```bash
# Open PRs (default)
citadel-cli pr list -R acme/demo

# Closed PRs
citadel-cli pr list -R acme/demo --state closed

# All PRs, machine-readable
citadel-cli pr list -R acme/demo --state all --output json

# Paginate
citadel-cli pr list -R acme/demo --limit 50 --cursor <cursor>
```

Columns: **#**, **Title**, **State**, **Source → Target**, **Author**, **Created**.

### View a pull request

```bash
# Human-readable summary with reviewer list
citadel-cli pr view -R acme/demo 42

# JSON
citadel-cli pr view -R acme/demo 42 --output json
```

### Create a pull request

```bash
# Minimal (title, source, and target are required)
citadel-cli pr create -R acme/demo --title "Add login page" \
    --source feature/login --target main

# With body inline
citadel-cli pr create -R acme/demo --title "Add login page" \
    --source feature/login --target main \
    --body "Closes #12. Adds OAuth2 login flow."

# Body via $EDITOR (when --body is omitted on a TTY)
citadel-cli pr create -R acme/demo --title "Add login page" \
    --source feature/login --target main
```

### Close a pull request

```bash
# Prompt for confirmation
citadel-cli pr close -R acme/demo 42

# Skip confirmation
citadel-cli pr close -R acme/demo 42 --yes
```

### Merge a pull request

```bash
citadel-cli pr merge -R acme/demo 42
```

Explicit error messages surface `merge_conflict`, `approval_required`, `already_merged`, and `invalid_state`.

### Diff a pull request

```bash
# File-stat table (additions/deletions per file)
citadel-cli pr diff -R acme/demo 42

# Raw unified diff for a single file
citadel-cli pr diff -R acme/demo 42 --file src/main.go
```

`--file` output is pipeable directly to `patch` or language-aware diff viewers.

### Check mergeability

```bash
# Human-readable
citadel-cli pr check -R acme/demo 42

# JSON (for scripting)
citadel-cli pr check -R acme/demo 42 --output json
```

Reports `MERGEABLE` or `NOT MERGEABLE` with a reason: `fast_forward`, `clean`, `conflict`, `no_merge_base`, or `resolve_error`.

### Comments

```bash
# List general comments
citadel-cli pr comment list -R acme/demo 42

# Add a comment
citadel-cli pr comment add -R acme/demo 42 --body "LGTM"
```

### Reviewers

```bash
# List reviewers and their status
citadel-cli pr reviewer list -R acme/demo 42

# Add a reviewer (UUID required)
citadel-cli pr reviewer add -R acme/demo 42 --reviewer <user-uuid>
```

### Submit a review

```bash
# Approve
citadel-cli pr review -R acme/demo 42 --approve

# Request changes
citadel-cli pr review -R acme/demo 42 --request-changes

# General comment (does not change approval state)
citadel-cli pr review -R acme/demo 42 --comment "See inline note above."

# Comment and approve in one call
citadel-cli pr review -R acme/demo 42 --approve --comment "Looks good, minor nit above."
```

`--approve` and `--request-changes` are mutually exclusive. At least one of `--approve`, `--request-changes`, or `--comment` is required.

## Repository commits

Browse the commit log of any repository you have access to.

### List commits

```bash
# List recent commits on the default branch
citadel-cli repo commit list acme/demo

# List commits touching a specific file
citadel-cli repo commit list acme/demo --path src/main.go

# List commits on a branch or tag
citadel-cli repo commit list acme/demo --ref feature/my-branch

# Fetch all pages at once
citadel-cli repo commit list acme/demo --all

# Machine-readable output
citadel-cli repo commit list acme/demo --output json
citadel-cli repo commit list acme/demo --output ndjson
citadel-cli repo commit list acme/demo --output csv
citadel-cli repo commit list acme/demo --output yaml
```

### Get a single commit

```bash
# Show commit metadata (SHA, author, message, parent(s), file stats)
citadel-cli repo commit get acme/demo abc1234

# Show raw unified diff for a specific file in that commit
citadel-cli repo commit get acme/demo abc1234 --path src/main.go

# Same diff, wrapped in JSON
citadel-cli repo commit get acme/demo abc1234 --path src/main.go --output json

# Full commit detail as JSON (metadata + file stats)
citadel-cli repo commit get acme/demo abc1234 --output json
```

SHA values may be full 40-character SHAs, partial SHAs (≥4 characters), branch names, or tag names.

## Repository browsing

Browse a repository's file tree and read file contents at any ref (branch, tag, or commit SHA).

### List directory entries

```bash
# List the root of the default branch
citadel-cli repo browse tree acme/demo

# List a subdirectory at a specific branch
citadel-cli repo browse tree acme/demo --ref main --path cmd

# Machine-readable output
citadel-cli repo browse tree acme/demo --output json
```

The tree output shows an icon (`📄` file, `📁` directory), the entry name, size, and abbreviated SHA.

### Read file content

```bash
# Print a file to stdout (suitable for piping)
citadel-cli repo browse blob acme/demo --path README.md

# Read a file at a specific ref
citadel-cli repo browse blob acme/demo --path src/main.go --ref feature/x

# Binary files print an informational line instead of raw bytes
citadel-cli repo browse blob acme/demo --path assets/logo.png

# Full metadata as JSON (sha, size, binary, encoding, content)
citadel-cli repo browse blob acme/demo --path go.mod --output json
```

## Repository topics

View and manage the topics attached to a repository. Topics are free-form labels used for discovery.

### List topics

```bash
citadel-cli repo topic list acme/demo
citadel-cli repo topic list acme/demo --output json
```

### Set topics

The `set` subcommand is a **full replace** — it overwrites all existing topics. Pass no topic arguments to clear all topics.

```bash
# Replace all topics
citadel-cli repo topic set acme/demo go cli devtools

# Clear all topics
citadel-cli repo topic set acme/demo

# Output result as JSON
citadel-cli repo topic set acme/demo go cli --output json
```

### Popular topics

List the most popular topics across all repositories on the platform.

```bash
citadel-cli repo topic popular
citadel-cli repo topic popular --limit 20
citadel-cli repo topic popular --output json
```

## Repository insights

View aggregate statistics for a repository: topics, open issue/milestone counts, branches, tags, languages, recent contributors, latest release, and a 52-week activity sparkline.

```bash
citadel-cli repo insights acme/demo
citadel-cli repo insights acme/demo --output json
```

Human output sections: **Topics**, **Stars/Pins**, **Counts** (table), **License**, **Languages** (top 5 by bytes), **Latest release**, **Recent contributors (30d)**, and **Activity sparkline** (52-week rolling window, oldest→newest).

Empty repositories (no commits yet) only show topics, star/pin counts, and the counts table.

## Labels

`citadel-cli label` manages namespace-level labels (create, list, edit, delete). All subcommands require authentication and accept `-R org/namespace` to identify the namespace.

### List labels

```bash
citadel-cli label list -R acme/demo
citadel-cli label list -R acme/demo --output json
```

Columns: **SLUG**, **NAME**, **COLOR**, **DEFAULT**, **DESCRIPTION**.

### Create a label

```bash
# Slug is auto-derived from --name (lowercase, spaces → hyphens)
citadel-cli label create -R acme/demo --name "Good First Issue" --color a2eeef

# Override the auto-slug
citadel-cli label create -R acme/demo --name "Good First Issue" --color a2eeef --slug gfi

# With description
citadel-cli label create -R acme/demo --name "Bug" --color d73a4a --description "Something is broken"

# JSON output
citadel-cli label create -R acme/demo --name "Enhancement" --color a2eeef --output json
```

`--color` accepts a 6-character hex string with or without a leading `#`.

### Edit a label

```bash
# Change color only (display_name and description preserved)
citadel-cli label edit -R acme/demo bug --color ff0000

# Rename
citadel-cli label edit -R acme/demo bug --name "Defect"

# Clear description
citadel-cli label edit -R acme/demo bug --description ""
```

At least one of `--name`, `--color`, or `--description` must be supplied. Shell completion suggests label slugs.

### Delete a label

```bash
# Interactive confirmation (type the slug to confirm)
citadel-cli label delete -R acme/demo old-label

# Skip confirmation
citadel-cli label delete -R acme/demo old-label --yes

# Dry run
citadel-cli label delete -R acme/demo old-label --dry-run
```

Attempting to delete the last default label for a semantic role returns a `label_delete_blocked` error.

## Troubleshooting

### Environment health check

Run `citadel-cli doctor` to check server reachability, local auth state, MCP
endpoint reachability, and config-file permissions in one pass.

```bash
citadel-cli doctor
```

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

The CLI binary is built by the GitHub Actions release workflow on every tag matching `v*`, producing static binaries for `linux-amd64`, `linux-arm64`, `darwin-arm64`, and `windows-amd64`. Update `CHANGELOG.md` and run `make verify && make build-all VERSION=<tag>` before pushing the tag. Channels:

- **GitHub Releases (canonical, today).** Each tag publishes a release at `github.com/Rethunk-Tech/citadel-cli/releases/tag/<tag>` with the four binaries + a `SHA256SUMS` file. Manual download:

  ```bash
  # Replace v0.1.0 with the latest tag.
  curl -L -o citadel-cli-linux-amd64 \
    https://github.com/Rethunk-Tech/citadel-cli/releases/download/v0.1.0/citadel-cli-linux-amd64
  curl -L -o SHA256SUMS \
    https://github.com/Rethunk-Tech/citadel-cli/releases/download/v0.1.0/SHA256SUMS
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
# Expect: citadel-cli-linux-amd64: OK
```

If verification fails, do not run the binary. Re-download from a fresh session and re-verify.
