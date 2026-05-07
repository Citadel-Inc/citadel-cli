# Plan — cli-deploy-tokens

`cli-deploy-tokens` is now actionable: the user already chose a nested command
shape, and the server survey confirmed the missing piece is backend CRUD in
`citadel`, not a blocked prerequisite.

## Approach

- Add namespace-scoped deploy-token REST routes in `citadel`; both repo and
  namespace CLI parents will call the same backend because repos are
  namespace-backed resources.
- Extend the `deploy_tokens` schema with a persisted display label so the spec's
  `--name` flag has first-class server support.
- Reuse existing `tokens:read` / `tokens:write` permissions and existing list /
  one-time-secret response patterns from the token and oauth-client surfaces.
- Mirror existing CLI list output, dry-run revoke behavior, and completion
  conventions instead of inventing a custom UX.

## Proposed server routes

- `GET /api/namespaces/{slug}/deploy-tokens`
- `POST /api/namespaces/{slug}/deploy-tokens`
- `DELETE /api/namespaces/{slug}/deploy-tokens/{id}`

`{slug}` is a namespace path, so org namespaces and repo namespaces share one
route family.

## Proposed file layout

```text
citadel/internal/api/deploytokensapi/handler.go        — list/create/revoke
citadel/internal/api/deploytokensapi/pure_test.go      — route/validation tests
citadel/cmd/citadel/main.go                            — route wiring
citadel/supabase/migrations/*_deploy_tokens_name.sql   — add persisted token label

citadel-cli/cmd/deploy_token.go                        — shared nested cobra commands
citadel-cli/cmd/deploy_token_test.go                   — CLI handler tests
citadel-cli/internal/completion/keys.go                — deploy-token completion key(s)
```
