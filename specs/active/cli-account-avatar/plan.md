## Approach

Extend the existing `account` command tree rather than creating a parallel top-level avatar noun. Reuse the CLI's standard mutation-output handling so `import` and `sync` behave like other account-facing verbs.

## Implementation notes

- Add an `account avatar` subtree under `cmd/account.go`
- Reuse shared output helpers for human summary + `--json`
- Validate `--source` client-side before sending requests
- Keep the CLI thin: the daemon remains the source of truth for provider-link and availability checks

## Risks

- Provider-backed smoke may depend on linked identities or live provider config, so `gravatar` is the safest likely live-smoke path if the environment lacks GitHub/Google links.
