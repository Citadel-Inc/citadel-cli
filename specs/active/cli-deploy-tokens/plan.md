# Plan — cli-deploy-tokens

Blocked on Q-table ratification (Q1–Q3) and server-side API survey.
No implementation work should begin until P0 human tasks are checked.

## Assumptions (to be validated in Q-table)

- Deploy tokens are managed at `/api/deploy-tokens` or similar route
- Response envelope matches the `{items, next_cursor}` pagination shape
- Cleartext token is returned only at creation time (`cleartext_token` field)

## Proposed file layout

```
cmd/deploy_token.go         — cobra commands (list, create, revoke)
cmd/deploy_token_test.go    — handler tests
internal/completion/keys.go — add KeyDeployTokens constant
```
