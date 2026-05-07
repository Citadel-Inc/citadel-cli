## Approach

Extend the existing `account` command tree with a small privacy subtree. Reuse the CLI's standard mutation-output helpers so reads and writes behave like other account-facing commands.

## Implementation notes

- Add `account privacy get` and `account privacy set`
- Reuse `--json` on mutation and `--output` on read
- Only send fields the user explicitly set
- Keep the CLI thin: validation is limited to "at least one field changed" plus flag parsing

## Risks

- The daemon treats omitted fields as "leave unchanged", so the CLI must preserve that distinction and avoid sending zero-values for untouched flags.
