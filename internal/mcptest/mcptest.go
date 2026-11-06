// Package mcptest serves a real go-sdk MCP server over HTTP for tests, so
// client code is exercised against the actual protocol rather than a canned
// JSON-RPC fake.
package mcptest

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// NewServer returns an MCP server identified as a test server.
func NewServer() *mcp.Server {
	return mcp.NewServer(&mcp.Implementation{Name: "citadel-mcp-test", Version: "1"}, nil)
}

// Tool registers a tool that accepts any arguments and returns h's result.
func Tool(s *mcp.Server, name, description string, h func(args map[string]any) (*mcp.CallToolResult, error)) {
	s.AddTool(
		&mcp.Tool{Name: name, Description: description, InputSchema: map[string]any{"type": "object"}},
		func(_ context.Context, req *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			var args map[string]any
			if len(req.Params.Arguments) > 0 {
				if err := json.Unmarshal(req.Params.Arguments, &args); err != nil {
					return nil, err
				}
			}
			return h(args)
		},
	)
}

// Handler returns the Streamable HTTP handler for s.
func Handler(s *mcp.Server) http.Handler {
	return mcp.NewStreamableHTTPHandler(func(*http.Request) *mcp.Server { return s }, nil)
}

// Serve mounts s at /mcp on an httptest server. wrap, when non-nil, sits in
// front of the MCP handler for header checks and fault injection.
func Serve(t *testing.T, s *mcp.Server, wrap func(http.Handler) http.Handler) *httptest.Server {
	t.Helper()
	h := Handler(s)
	if wrap != nil {
		h = wrap(h)
	}
	mux := http.NewServeMux()
	mux.Handle("/mcp", h)
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)
	return srv
}
