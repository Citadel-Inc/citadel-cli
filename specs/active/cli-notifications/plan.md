## Approach

Add a top-level `notification` command tree for inbox operations and a nested `prefs` subtree for preference inspection/mutation. Reuse the existing pagination helpers and output modes already used by repo, issue, and audit lists.

## Implementation notes

- Add list/read/read-all/unread-count plus prefs get/set
- Reuse standard list pagination flags and machine outputs
- Keep mutation defaults human-friendly, with `--json` where appropriate
- Mirror the daemon's preference shapes instead of inventing CLI-only abstractions

## Risks

- The notification-prefs response shape is richer than a simple boolean map, so the spec owner should inspect the exact handler payload before implementation and preserve field naming verbatim.
