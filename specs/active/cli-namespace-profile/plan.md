## Approach

Add a `namespace profile` subtree under the existing `namespace` command. Read-only; no mutation verbs. Reuse the existing namespace-path handling and single-resource output helpers.

## Implementation notes

- Add `namespace profile get` only
- Render the key identity fields: display_name, bio, location, social_links, etc.
- Preserve daemon field names in JSON/YAML output
- Surface visibility/ownership 404 failures clearly

## Risks

- `social_links` is a `map[string]string`; ensure the table renderer flattens it gracefully rather than printing a raw Go map string.
