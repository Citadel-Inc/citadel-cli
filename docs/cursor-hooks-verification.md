# Cursor hooks verification

Cursor `cursor-agent` 2026.10.28-4a7b674, against a Citadel server built from
`main` at 58467277d on a local Postgres. Remotes were plain HTTP on loopback
(the helper path HTTPS uses, without TLS) and SSH on a non-22 port. Setup: `HOME` stayed real
so Cursor could read its login in place, with a scratch directory bind-mounted
over `~/.cursor`.

A real session means `cursor-agent -p`. Replay means the hook verbs driven with
payloads recorded from those real sessions, and shell commands run with the
environment a Cursor Shell gets. A real session running `git push` did not run:
the harness refused `cursor-agent -p --force` as a nested agent, and widening the
allowlist to get around that was out of bounds.

| Step | Result | How |
| --- | --- | --- |
| `hook install` writes hooks and allow rule (user and project scope) | Pass after fix | Real |
| `sessionStart` opens a session with `client: cursor`, `cursor-<host>` agent | Pass after fix | Real |
| `citadel-cli kg` runs with no refusal, as the session | Pass after fix, sandbox on and off | Real. Without the rule: `Rejected:` |
| Push lands a `git.push` row naming agent, session and owner (HTTP and SSH) | Pass | Replay |
| Push to another owner's repo refused, row `reason: permission_denied` | Pass | Replay |
| Push with a read-only session refused, row `reason: scope_denied` | Pass | Replay |
| CLI call outside scopes refused, `agent.scope_refused` names the session | Pass | Replay |
| `sessionEnd` ends the session and revokes its token, repo config and key removed | Pass | Real; token rows checked in the database |
| Next call with that token refused (API 401, git `authentication required`) | Pass | Replay |
| Two parallel sessions in one repo stay separately attributed | Pass after fix | Replay |

## SSH and HTTPS

Replay shows no difference between them, but a replay shell has no Cursor sandbox. `docs/cli.md` records
a real observation on this Cursor version: under the sandbox, an SSH `git push`
chained after other commands on a non-22 port went through Cursor's relay and
failed, while the same push as its own command worked. That was not re-run here,
because it needs a real push session.

## Not exercised

- `git push` from a real Cursor session, on either transport.
- TLS: HTTPS remotes are the same helper path, tested over loopback HTTP.
- The Cursor IDE: only `cursor-agent`.
- Commit trailers with parallel sessions: the shared `prepare-commit-msg` hook names the last session started.
