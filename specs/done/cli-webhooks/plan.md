# Plan — cli-webhooks

Q-table ratified 071744ZMAY26 after surveying `/usr/local/src/com.github/Rethunk-Tech/citadel`:

- command shape = nested `repo webhook` and `namespace webhook`
- server API = namespace-scoped list/create/update/delete under `/api/namespaces/{slug}/webhooks`
- repo support = repo namespaces via multi-segment namespace paths (`acme/demo`)
- secrets = server-generated and returned once as `cleartext_secret`
- no server-side test ping route exists yet

## Proposed file layout

```
cmd/webhook.go          — repo/namespace webhook cobra commands + helpers
cmd/webhook_test.go     — handler tests
```
