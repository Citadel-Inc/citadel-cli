# Plan — cli-oauth-clients

Each verb thin-wraps the existing OAuth client REST surface. `rotate-secret` prints the new secret once to stdout (with `--copy-to-clipboard` if available), then exits — matches the dashboard one-time reveal UX.

`--org <slug>` flag forwards to the API as a namespace filter; absent flag defaults to the actor's account-scoped clients.

Reuses the destructive-confirm helper from `go-citadel-cli-repo` (typed slug match) for revoke. Both commands share the same UX so muscle memory transfers.
