# Plan — cli-mcp-resources

Resource registry is a `map[uri]Resource` populated at server init. `Resource` carries a `Read(ctx) ([]byte, mime, error)` closure; the handler dispatches by URI prefix (`citadel://doc/...` reads from the docs tree; `citadel://spec/...` reads from `specs/`).

Prompt registry is similarly a `map[name]Prompt`. `Prompt` carries `Get(args) ([]Message, error)` and a JSON-Schema arguments declaration. v1 prompts are static templates with light arg interpolation.

CLI verbs hit the running MCP server (HTTP or stdio depending on context); render `resources list` / `prompts list` as a table; `read` / `get` dump the result body or message list.

Auth (Q2): all-or-nothing tied to the MCP session token. If a session can call tools, it can read resources + prompts. Per-resource ACL is a follow-on if a multi-tenant exposure ever lands.
