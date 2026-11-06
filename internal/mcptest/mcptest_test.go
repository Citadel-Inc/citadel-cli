package mcptest

import (
	"fmt"
	"net/http"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func TestServe_RoundTripsToolArguments(t *testing.T) {
	s := NewServer()
	Tool(s, "echo", "echoes", func(args map[string]any) (*mcp.CallToolResult, error) {
		return &mcp.CallToolResult{Content: []mcp.Content{&mcp.TextContent{Text: fmt.Sprint(args["v"])}}}, nil
	})
	var wrapped bool
	srv := Serve(t, s, func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			wrapped = true
			next.ServeHTTP(w, r)
		})
	})

	client := mcp.NewClient(&mcp.Implementation{Name: "t", Version: "1"}, nil)
	session, err := client.Connect(t.Context(), &mcp.StreamableClientTransport{Endpoint: srv.URL + "/mcp", DisableStandaloneSSE: true}, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = session.Close() }()
	res, err := session.CallTool(t.Context(), &mcp.CallToolParams{Name: "echo", Arguments: map[string]any{"v": "hi"}})
	if err != nil {
		t.Fatal(err)
	}
	if tc, ok := res.Content[0].(*mcp.TextContent); !ok || tc.Text != "hi" {
		t.Fatalf("content = %+v", res.Content)
	}
	if !wrapped {
		t.Fatal("wrap middleware not applied")
	}
}
