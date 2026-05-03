# Plan — cli-mcp-stream

HTTP handler branches on `Accept: text/event-stream`: if present + tool flagged `streaming`, hijack the response writer for SSE framing (`data: <json>\n\n`). Tool runtime exposes `ctx.Emit(name, payload)` that the dispatcher serialises onto the SSE stream.

Heartbeat is a 15s ticker (Q1) emitting a no-op `: ping\n\n` comment line — keeps proxies from severing the connection.

Client side: when CLI tool registry advertises `streaming: true` for a verb, the HTTP request adds `Accept: text/event-stream` and demuxes events. Progress events render as a terminal status line (overwrite via `\r`); terminal `result` event prints the final response.

Throttle (Q3) caps tool-emitted progress events at 10/sec via a token bucket in the dispatcher; over-cap events are dropped silently with a counter.
