# Plan — go-citadel-cli-repo

Each verb is a thin wrapper over the existing HTTP API: parse flags, build request, call API, format output. Reuse the existing `citadel-cli/internal/api` client + auth bearer handling.

Destructive-confirm helper prompts for typed slug match and bails on mismatch; `--yes` skips. Matches dashboard danger-zone UX so muscle-memory transfers between web + CLI.

`--output json` emitter shared across verbs via a `cmd.PrintRecord(any)` helper that branches on `viper.GetString("output")`.

Integration test boots an ephemeral droplet fixture (or hits a designated test substrate), runs each verb in sequence, asserts response + side-effects via the API client.
