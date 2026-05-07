# Plan — cli-repo-topics

## Implementation approach

Three new subcommands under `repoTopicCmd`:

1. `repo topic list` — calls `GET /api/namespaces/{ns}/repos/{repo}/topics`
   - Returns `{"topics":["go","cli",...]}`
   - Human: one topic per line
   - JSON: emit full response

2. `repo topic set <topic>...` — calls `PUT /api/namespaces/{ns}/repos/{repo}/topics`
   - Body: `{"topics":["a","b",...]}`
   - Human: prints the new topic list (same as list output)
   - JSON: emit full response

3. `repo topic popular` — calls `GET /api/topics/popular?limit=N`
   - Returns `[{"topic":"go","count":42},...]` (array, not object!)
   - Human: tab-writer table with topic and count columns
   - JSON: emit array as-is

## Notes

- `popular` has no namespace/repo scope — uses global API path.
- `set` with zero args clears all topics (calls PUT with `{"topics":[]}`).
- Error shape from server: `{"error":"topic_too_long"}` etc — pass through via `handleAPIError`.
