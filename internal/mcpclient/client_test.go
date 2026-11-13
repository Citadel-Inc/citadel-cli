package mcpclient

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/Citadel-Inc/citadel-cli/internal/mcptest"
)

func TestMain(m *testing.M) {
	waitRetry = func(ctx context.Context, _ int) error {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			return nil
		}
	}
	os.Exit(m.Run())
}

// failMethod makes the first n POSTs carrying the JSON-RPC method answer with
// status instead of reaching the MCP handler, and counts every POST for it.
func failMethod(method string, status, n int, seen *atomic.Int32) func(http.Handler) http.Handler {
	var failed atomic.Int32
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method == http.MethodPost {
				body, _ := io.ReadAll(r.Body)
				r.Body = io.NopCloser(bytes.NewReader(body))
				if strings.Contains(string(body), `"method":"`+method+`"`) {
					seen.Add(1)
					if int(failed.Add(1)) <= n {
						w.WriteHeader(status)
						return
					}
				}
			}
			next.ServeHTTP(w, r)
		})
	}
}

func textResult(s string) *mcp.CallToolResult {
	return &mcp.CallToolResult{Content: []mcp.Content{&mcp.TextContent{Text: s}}}
}

func dial(t *testing.T, url, token string) *Client {
	t.Helper()
	c := New(url+"/mcp", token, 5*time.Second, Options{})
	if err := c.Initialize(t.Context()); err != nil {
		t.Fatalf("Initialize: %v", err)
	}
	t.Cleanup(func() { _ = c.Close() })
	return c
}

func TestInitializeAndToolsList_SendBearerToken(t *testing.T) {
	t.Parallel()
	s := mcptest.NewServer()
	mcptest.Tool(s, "get_namespace", "Look up a namespace", func(map[string]any) (*mcp.CallToolResult, error) {
		return textResult("ok"), nil
	})
	var sawAuth atomic.Int32
	srv := mcptest.Serve(t, s, func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Header.Get("Authorization") != "Bearer tok-123" {
				t.Errorf("Authorization = %q on %s", r.Header.Get("Authorization"), r.Method)
			}
			sawAuth.Add(1)
			next.ServeHTTP(w, r)
		})
	})
	c := dial(t, srv.URL, "tok-123")
	tools, err := c.ToolsList(t.Context())
	if err != nil {
		t.Fatalf("ToolsList: %v", err)
	}
	if len(tools) != 1 || tools[0].Name != "get_namespace" || tools[0].Description != "Look up a namespace" {
		t.Fatalf("tools = %+v", tools)
	}
	if sawAuth.Load() < 2 {
		t.Fatalf("expected authenticated requests, saw %d", sawAuth.Load())
	}
}

func TestToolsList_RetriesTransientStatus(t *testing.T) {
	t.Parallel()
	for _, status := range []int{http.StatusServiceUnavailable, http.StatusInternalServerError} {
		t.Run(http.StatusText(status), func(t *testing.T) {
			t.Parallel()
			s := mcptest.NewServer()
			mcptest.Tool(s, "t1", "", func(map[string]any) (*mcp.CallToolResult, error) { return textResult("x"), nil })
			var seen atomic.Int32
			srv := mcptest.Serve(t, s, failMethod("tools/list", status, 1, &seen))
			c := dial(t, srv.URL, "t")
			tools, err := c.ToolsList(t.Context())
			if err != nil {
				t.Fatalf("ToolsList: %v", err)
			}
			if len(tools) != 1 || seen.Load() != 2 {
				t.Fatalf("tools=%d list posts=%d, want 1 tool and 2 posts", len(tools), seen.Load())
			}
		})
	}
}

func TestToolsList_GivesUpAfterMaxAttempts(t *testing.T) {
	t.Parallel()
	var seen atomic.Int32
	srv := mcptest.Serve(t, mcptest.NewServer(), failMethod("tools/list", http.StatusServiceUnavailable, 100, &seen))
	c := dial(t, srv.URL, "t")
	if _, err := c.ToolsList(t.Context()); err == nil {
		t.Fatal("want error after exhausted retries")
	}
	if seen.Load() != maxAttempts {
		t.Fatalf("list posts = %d, want %d", seen.Load(), maxAttempts)
	}
}

func TestToolsCall_ResultAndRaw(t *testing.T) {
	t.Parallel()
	s := mcptest.NewServer()
	var got map[string]any
	mcptest.Tool(s, "echo", "", func(args map[string]any) (*mcp.CallToolResult, error) {
		got = args
		r := textResult("boom")
		r.IsError = true
		return r, nil
	})
	srv := mcptest.Serve(t, s, nil)
	c := dial(t, srv.URL, "t")
	res, err := c.ToolsCall(t.Context(), "echo", map[string]any{"n": 7, "s": "v"})
	if err != nil {
		t.Fatalf("ToolsCall: %v", err)
	}
	if !res.IsError || len(res.Content) != 1 || res.Content[0]["type"] != "text" || res.Content[0]["text"] != "boom" {
		t.Fatalf("result = %+v", res)
	}
	if got["n"] != float64(7) || got["s"] != "v" {
		t.Fatalf("server saw args %v", got)
	}
	var raw struct {
		IsError bool `json:"isError"`
	}
	if err := json.Unmarshal(res.Raw, &raw); err != nil || !raw.IsError {
		t.Fatalf("raw = %s (%v)", res.Raw, err)
	}
}

func TestToolsCall_NeverRetried(t *testing.T) {
	t.Parallel()
	s := mcptest.NewServer()
	mcptest.Tool(s, "t1", "", func(map[string]any) (*mcp.CallToolResult, error) { return textResult("x"), nil })
	var seen atomic.Int32
	srv := mcptest.Serve(t, s, failMethod("tools/call", http.StatusInternalServerError, 1, &seen))
	c := dial(t, srv.URL, "t")
	if _, err := c.ToolsCall(t.Context(), "t1", nil); err == nil {
		t.Fatal("want error from HTTP 500")
	}
	if seen.Load() != 1 {
		t.Fatalf("tools/call posts = %d, want 1", seen.Load())
	}
}

func TestInitialize_NotRetried(t *testing.T) {
	t.Parallel()
	var seen atomic.Int32
	srv := mcptest.Serve(t, mcptest.NewServer(), failMethod("initialize", http.StatusServiceUnavailable, 1, &seen))
	c := New(srv.URL+"/mcp", "t", 5*time.Second, Options{})
	if err := c.Initialize(t.Context()); err == nil {
		t.Fatal("want initialize error")
	}
	if seen.Load() != 1 {
		t.Fatalf("initialize posts = %d, want 1", seen.Load())
	}
}

func TestUnauthorizedHTTP401(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	}))
	t.Cleanup(srv.Close)
	c := New(srv.URL, "bad", 5*time.Second, Options{})
	if err := c.Initialize(t.Context()); !IsUnauthorized(err) {
		t.Fatalf("want unauthorized, got %v", err)
	}
}

func TestUnauthorizedMidSession(t *testing.T) {
	t.Parallel()
	s := mcptest.NewServer()
	var revoke atomic.Bool
	srv := mcptest.Serve(t, s, func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if revoke.Load() && r.Method == http.MethodPost {
				w.WriteHeader(http.StatusUnauthorized)
				return
			}
			next.ServeHTTP(w, r)
		})
	})
	c := dial(t, srv.URL, "t")
	revoke.Store(true)
	if _, err := c.ToolsList(t.Context()); !IsUnauthorized(err) {
		t.Fatalf("want unauthorized, got %v", err)
	}
}

// TestIsUnauthorized_UnwrapsWrapped guards the errors.As migration: callers
// that wrap a *mcpclient.Error with fmt.Errorf("...: %w", err) must still
// classify as unauthorized.
func TestIsUnauthorized_UnwrapsWrapped(t *testing.T) {
	t.Parallel()
	inner := &Error{Kind: KindUnauthorized, Message: "bad token"}
	wrapped := fmt.Errorf("mcp call failed: %w", inner)
	if !IsUnauthorized(wrapped) {
		t.Fatal("wrapped *Error must classify as unauthorized")
	}
	if IsUnauthorized(errors.New("plain error, not unauthorized")) {
		t.Fatal("plain error must not classify as unauthorized")
	}
}

func TestJSONRPCErrorFromServerIsClassified(t *testing.T) {
	t.Parallel()
	srv := mcptest.Serve(t, mcptest.NewServer(), nil)
	c := dial(t, srv.URL, "t")
	_, err := c.ResourcesRead(t.Context(), "citadel://missing")
	var e *Error
	if !errors.As(err, &e) || e.JSONRPCCode == 0 || e.Message == "" {
		t.Fatalf("want typed Error carrying the JSON-RPC code, got %v", err)
	}
	_, err = c.ToolsCall(t.Context(), "nope", nil)
	if !errors.As(err, &e) || e.JSONRPCCode == 0 || !strings.Contains(e.Message, "nope") {
		t.Fatalf("want typed Error naming the tool, got %v", err)
	}
}

// rewriteProtocol swaps the version in the server's initialize response.
func rewriteProtocol(to string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			rec := httptest.NewRecorder()
			next.ServeHTTP(rec, r)
			body := strings.Replace(rec.Body.String(), `"protocolVersion":"`+ProtocolVersion+`"`, `"protocolVersion":"`+to+`"`, 1)
			for k, v := range rec.Header() {
				if k != "Content-Length" {
					w.Header()[k] = v
				}
			}
			w.WriteHeader(rec.Code)
			_, _ = io.WriteString(w, body)
		})
	}
}

func TestVersionMismatch(t *testing.T) {
	t.Parallel()
	for _, server := range []string{"2025-06-18", "1999-01-01"} {
		t.Run(server, func(t *testing.T) {
			t.Parallel()
			srv := mcptest.Serve(t, mcptest.NewServer(), rewriteProtocol(server))
			c := New(srv.URL+"/mcp", "t", 5*time.Second, Options{})
			err := c.Initialize(t.Context())
			var e *Error
			if !errors.As(err, &e) || e.Kind != KindVersionMismatch || !strings.Contains(e.Message, server) {
				t.Fatalf("want KindVersionMismatch naming %s, got %v", server, err)
			}
		})
	}
}

func TestResourcesListAndRead(t *testing.T) {
	t.Parallel()
	s := mcptest.NewServer()
	s.AddResource(&mcp.Resource{URI: "citadel://ns/x", Name: "x", Description: "desc", MIMEType: "text/plain"},
		func(_ context.Context, req *mcp.ReadResourceRequest) (*mcp.ReadResourceResult, error) {
			return &mcp.ReadResourceResult{Contents: []*mcp.ResourceContents{{URI: req.Params.URI, Text: "body"}}}, nil
		})
	srv := mcptest.Serve(t, s, nil)
	c := dial(t, srv.URL, "t")
	rows, err := c.ResourcesList(t.Context())
	if err != nil || len(rows) != 1 || rows[0].URI != "citadel://ns/x" || rows[0].Description != "desc" {
		t.Fatalf("rows=%+v err=%v", rows, err)
	}
	raw, err := c.ResourcesRead(t.Context(), "citadel://ns/x")
	if err != nil {
		t.Fatal(err)
	}
	var parsed struct {
		Contents []map[string]any `json:"contents"`
	}
	if err := json.Unmarshal(raw, &parsed); err != nil || len(parsed.Contents) != 1 || parsed.Contents[0]["text"] != "body" {
		t.Fatalf("raw=%s err=%v", raw, err)
	}
}

func TestPromptsListAndGet(t *testing.T) {
	t.Parallel()
	s := mcptest.NewServer()
	var gotArgs map[string]string
	s.AddPrompt(&mcp.Prompt{Name: "issue_template", Description: "Open an issue", Arguments: []*mcp.PromptArgument{{Name: "topic"}}},
		func(_ context.Context, req *mcp.GetPromptRequest) (*mcp.GetPromptResult, error) {
			gotArgs = req.Params.Arguments
			return &mcp.GetPromptResult{
				Description: "Open an issue",
				Messages:    []*mcp.PromptMessage{{Role: "user", Content: &mcp.TextContent{Text: "Title?"}}},
			}, nil
		})
	srv := mcptest.Serve(t, s, nil)
	c := dial(t, srv.URL, "t")
	rows, err := c.PromptsList(t.Context())
	if err != nil || len(rows) != 1 || rows[0].Name != "issue_template" {
		t.Fatalf("rows=%+v err=%v", rows, err)
	}
	raw, err := c.PromptsGet(t.Context(), "issue_template", map[string]any{"topic": "auth", "n": 3})
	if err != nil {
		t.Fatal(err)
	}
	if gotArgs["topic"] != "auth" || gotArgs["n"] != "3" {
		t.Fatalf("server saw args %v", gotArgs)
	}
	var parsed struct {
		Messages []map[string]any `json:"messages"`
	}
	if err := json.Unmarshal(raw, &parsed); err != nil || len(parsed.Messages) != 1 || parsed.Messages[0]["role"] != "user" {
		t.Fatalf("raw=%s err=%v", raw, err)
	}
}

func TestNotInitialized(t *testing.T) {
	t.Parallel()
	c := New("http://nope", "t", time.Second, Options{})
	checks := map[string]func() error{
		"ToolsList":     func() error { _, err := c.ToolsList(t.Context()); return err },
		"ToolsCall":     func() error { _, err := c.ToolsCall(t.Context(), "x", nil); return err },
		"ResourcesList": func() error { _, err := c.ResourcesList(t.Context()); return err },
		"ResourcesRead": func() error { _, err := c.ResourcesRead(t.Context(), "x"); return err },
		"PromptsList":   func() error { _, err := c.PromptsList(t.Context()); return err },
		"PromptsGet":    func() error { _, err := c.PromptsGet(t.Context(), "x", nil); return err },
	}
	for name, fn := range checks {
		if err := fn(); err == nil || !strings.Contains(err.Error(), "not initialized") {
			t.Errorf("%s: want not-initialized, got %v", name, err)
		}
	}
}

func TestClassifyJSONRPCError_AllKinds(t *testing.T) {
	t.Parallel()
	cases := []struct {
		code int
		want Kind
	}{
		{-32601, KindMethodNotFound},
		{-32602, KindInvalidParams},
		{-32600, KindInvalidRequest},
		{-32700, KindParseError},
		{-32001, KindUnauthorized},
		{-99999, KindUnknown},
	}
	for _, tc := range cases {
		got := classifyJSONRPCError(tc.code, "msg")
		if got.Kind != tc.want {
			t.Errorf("code %d: want kind %d, got %d", tc.code, tc.want, got.Kind)
		}
	}
}

func TestNew_DefaultTimeout(t *testing.T) {
	t.Parallel()
	c := New("http://example.test", "t", 0, Options{})
	if c.http.Timeout != 60*time.Second {
		t.Fatalf("default timeout: got %v", c.http.Timeout)
	}
}

func TestTimeoutSurfacesAsError(t *testing.T) {
	t.Parallel()
	block := make(chan struct{})
	s := mcptest.NewServer()
	mcptest.Tool(s, "slow", "", func(map[string]any) (*mcp.CallToolResult, error) {
		<-block
		return textResult("late"), nil
	})
	srv := mcptest.Serve(t, s, nil)
	c := New(srv.URL+"/mcp", "t", 100*time.Millisecond, Options{})
	if err := c.Initialize(t.Context()); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = c.Close() })
	_, err := c.ToolsCall(t.Context(), "slow", nil)
	close(block)
	if err == nil {
		t.Fatal("want timeout error")
	}
}

func TestAuthTransport_CapsResponseBody(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = io.Copy(w, io.LimitReader(neverEnding('x'), maxResponseBody+1))
	}))
	t.Cleanup(srv.Close)
	u, err := url.Parse(srv.URL)
	if err != nil {
		t.Fatal(err)
	}
	tap := &authTransport{base: http.DefaultTransport, token: "t", scheme: u.Scheme, host: u.Host}
	req, err := http.NewRequestWithContext(t.Context(), http.MethodPost, srv.URL, nil)
	if err != nil {
		t.Fatal(err)
	}
	resp, err := tap.RoundTrip(req)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = resp.Body.Close() }()
	_, err = io.ReadAll(resp.Body)
	if err == nil || !strings.Contains(err.Error(), "32 MiB") {
		t.Fatalf("want 32 MiB cap error, got %v", err)
	}
}

type neverEnding byte

func (b neverEnding) Read(p []byte) (int, error) {
	for i := range p {
		p[i] = byte(b)
	}
	return len(p), nil
}

func TestAuthTransportStripsSessionHeaderOffOrigin(t *testing.T) {
	var leaked atomic.Bool
	other := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "" || r.Header.Get("Mcp-Session-Id") != "" {
			leaked.Store(true)
		}
		w.WriteHeader(http.StatusOK)
	}))
	t.Cleanup(other.Close)

	tap := &authTransport{base: http.DefaultTransport, token: "secret-token", scheme: "http", host: "origin.test"}
	req, err := http.NewRequestWithContext(t.Context(), http.MethodPost, other.URL+"/mcp", nil)
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Mcp-Session-Id", "sess-1")
	resp, err := tap.RoundTrip(req)
	if err != nil {
		t.Fatal(err)
	}
	_ = resp.Body.Close()
	if leaked.Load() {
		t.Fatal("session header or bearer token was sent off origin")
	}
}

func TestRedirectToOtherHostDoesNotCarryToken(t *testing.T) {
	var leaked atomic.Bool
	other := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "" {
			leaked.Store(true)
		}
		w.WriteHeader(http.StatusInternalServerError)
	}))
	t.Cleanup(other.Close)
	// 127.0.0.1 and localhost are different hosts to the client.
	target := strings.Replace(other.URL, "127.0.0.1", "localhost", 1)
	origin := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, target+"/mcp", http.StatusTemporaryRedirect)
	}))
	t.Cleanup(origin.Close)

	c := New(origin.URL+"/mcp", "secret-token", 5*time.Second, Options{})
	_ = c.Initialize(t.Context())
	if leaked.Load() {
		t.Fatal("bearer token was sent to the redirect target on another host")
	}
}
