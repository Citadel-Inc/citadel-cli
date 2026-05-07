## Approach

Reuse the CLI's existing request, output, browser-opening, and confirmation helpers instead of introducing a new auth client layer. Keep the surface small and daemon-aligned: provider discovery is public, while link and unlink use the same saved-session auth path as the rest of the authenticated CLI.

## Implementation notes

- Add a new `cmd/auth_provider.go` command tree
- Reuse standard list-output plumbing for provider list
- Reuse browser launch helpers for `link`
- Reuse typed/yes-style confirmation helpers for `unlink`
- Add provider completions backed by `GET /api/auth/providers`
- Cover request shapes, output handling, and error passthrough in tests

## Risks

- The daemon does not expose a linked-identities listing route, so `unlink` cannot offer "only linked providers" completion; completion should therefore use the configured provider registry, not a guessed per-user state view.
- Live smoke can verify list and link initiation without requiring the browser callback to complete inside the CLI.
