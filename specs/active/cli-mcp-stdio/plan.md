# Plan — cli-mcp-stdio

`citadel mcp serve --stdio` constructs the same MCP server used in HTTP mode (`internal/mcp/server.go`) but mounts it on a stdio transport (`mark3labs/mcp-go` already provides `server.NewStdioServer`). Token lookup via `os.Getenv("CITADEL_TOKEN")` — stamps the server's auth context.

Logging via `log/slog` with the handler bound to `os.Stderr`. Default level `warn` (Q2) — parent clients (Claude Desktop) surface stderr in their UI and noisy logs are a UX problem.

EOF on stdin triggers `srv.Shutdown(ctx)` with a 5s timeout (Q3); in-flight tool calls drain or are cancelled at deadline.

Conformance test fires the standard MCP init handshake + a tools/list + a tools/call against the process via stdin/stdout pipes.
