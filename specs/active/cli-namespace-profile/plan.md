## Approach

Extend the existing `namespace` command tree with a `profile` subtree. Reuse the CLI's existing namespace-path handling and single-resource output helpers so profile reads and edits feel like the rest of the namespace surface.

## Implementation notes

- Add `namespace profile get` and `namespace profile edit`
- Keep scalar-field edits flag-driven
- Preserve daemon field names in JSON/YAML output
- Surface visibility/ownership failures directly rather than masking them

## Risks

- `social_links` is the only structured field, so the implementation may need a small helper for repeated `--social key=url` flags or a JSON input flag; that design can be finalized during implementation without changing the overall spec shape.
