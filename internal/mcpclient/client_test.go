package mcpclient

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

type rpcReq struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      int             `json:"id"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params"`
}

func newRPCServer(t *testing.T, handler func(w http.ResponseWriter, r *http.Request, req rpcReq)) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req rpcReq
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatalf("server: decode req: %v", err)
		}
		handler(w, r, req)
	}))
}

func writeResult(w http.ResponseWriter, sessionID string, id int, result any) {
	if sessionID != "" {
		w.Header().Set("Mcp-Session-Id", sessionID)
	}
	w.Header().Set("Content-Type", "application/json")
	body, _ := json.Marshal(map[string]any{"jsonrpc": "2.0", "id": id, "result": result})
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(body)
}

func writeError(w http.ResponseWriter, status, id, code int, message string) {
	w.Header().Set("Content-Type", "application/json")
	body, _ := json.Marshal(map[string]any{
		"jsonrpc": "2.0",
		"id":      id,
		"error":   map[string]any{"code": code, "message": message},
	})
	w.WriteHeader(status)
	_, _ = w.Write(body)
}

func TestInitializeAndToolsList(t *testing.T) {
	srv := newRPCServer(t, func(w http.ResponseWriter, r *http.Request, req rpcReq) {
		if got := r.Header.Get("Authorization"); got != "Bearer tok-123" {
			t.Errorf("Authorization header = %q want Bearer tok-123", got)
		}
		switch req.Method {
		case "initialize":
			writeResult(w, "sess-abc", req.ID, map[string]any{
				"protocolVersion": ProtocolVersion,
				"serverInfo":      map[string]any{"name": "citadel-mcp", "version": "1"},
			})
		case "tools/list":
			if r.Header.Get("Mcp-Session-Id") != "sess-abc" {
				t.Errorf("session header missing on tools/list")
			}
			writeResult(w, "sess-abc", req.ID, map[string]any{
				"tools": []map[string]any{
					{"name": "get_namespace", "description": "Look up a namespace"},
				},
			})
		default:
			t.Fatalf("unexpected method %q", req.Method)
		}
	})
	defer srv.Close()

	c := New(srv.URL, "tok-123", 5*time.Second)
	if err := c.Initialize(context.Background()); err != nil {
		t.Fatalf("Initialize: %v", err)
	}
	if c.ServerInfoValue().Name != "citadel-mcp" {
		t.Errorf("ServerInfo.Name = %q", c.ServerInfoValue().Name)
	}
	tools, err := c.ToolsList(context.Background())
	if err != nil {
		t.Fatalf("ToolsList: %v", err)
	}
	if len(tools) != 1 || tools[0].Name != "get_namespace" {
		t.Fatalf("tools = %+v", tools)
	}
}

func TestToolsCall(t *testing.T) {
	srv := newRPCServer(t, func(w http.ResponseWriter, r *http.Request, req rpcReq) {
		switch req.Method {
		case "initialize":
			writeResult(w, "s1", req.ID, map[string]any{"protocolVersion": ProtocolVersion})
		case "tools/call":
			var p struct {
				Name      string         `json:"name"`
				Arguments map[string]any `json:"arguments"`
			}
			_ = json.Unmarshal(req.Params, &p)
			if p.Name != "get_namespace" || p.Arguments["path"] != "damon" {
				t.Errorf("bad params: %+v", p)
			}
			writeResult(w, "s1", req.ID, map[string]any{
				"content": []map[string]any{{"type": "text", "text": "hello"}},
			})
		}
	})
	defer srv.Close()
	c := New(srv.URL, "t", 5*time.Second)
	if err := c.Initialize(context.Background()); err != nil {
		t.Fatal(err)
	}
	res, err := c.ToolsCall(context.Background(), "get_namespace", map[string]any{"path": "damon"})
	if err != nil {
		t.Fatalf("ToolsCall: %v", err)
	}
	if len(res.Content) != 1 || res.Content[0]["text"] != "hello" {
		t.Errorf("content = %+v", res.Content)
	}
}

func TestUnauthorizedHTTP401(t *testing.T) {
	srv := newRPCServer(t, func(w http.ResponseWriter, r *http.Request, req rpcReq) {
		w.WriteHeader(http.StatusUnauthorized)
	})
	defer srv.Close()
	c := New(srv.URL, "bad", time.Second)
	err := c.Initialize(context.Background())
	if !IsUnauthorized(err) {
		t.Fatalf("want unauthorized, got %v", err)
	}
}

// TestIsUnauthorized_UnwrapsWrapped guards the errors.As migration: callers
// that wrap a *mcpclient.Error with fmt.Errorf("...: %w", err) must still
// classify as unauthorized.
func TestIsUnauthorized_UnwrapsWrapped(t *testing.T) {
	inner := &Error{Kind: KindUnauthorized, Message: "bad token"}
	wrapped := fmt.Errorf("mcp call failed: %w", inner)
	if !IsUnauthorized(wrapped) {
		t.Fatal("wrapped *Error must classify as unauthorized")
	}
	if IsUnauthorized(fmt.Errorf("plain error, not unauthorized")) {
		t.Fatal("plain error must not classify as unauthorized")
	}
}

func TestMethodNotFound(t *testing.T) {
	srv := newRPCServer(t, func(w http.ResponseWriter, r *http.Request, req rpcReq) {
		switch req.Method {
		case "initialize":
			writeResult(w, "s", req.ID, map[string]any{"protocolVersion": ProtocolVersion})
		case "tools/call":
			writeError(w, http.StatusBadRequest, req.ID, -32601, "tool not found: nope")
		}
	})
	defer srv.Close()
	c := New(srv.URL, "t", time.Second)
	if err := c.Initialize(context.Background()); err != nil {
		t.Fatal(err)
	}
	_, err := c.ToolsCall(context.Background(), "nope", nil)
	e, ok := err.(*Error)
	if !ok || e.Kind != KindMethodNotFound || !strings.Contains(e.Message, "nope") {
		t.Fatalf("want KindMethodNotFound w/ name, got %v", err)
	}
}

func TestVersionMismatch(t *testing.T) {
	srv := newRPCServer(t, func(w http.ResponseWriter, r *http.Request, req rpcReq) {
		writeResult(w, "s", req.ID, map[string]any{"protocolVersion": "1999-01-01"})
	})
	defer srv.Close()
	c := New(srv.URL, "t", time.Second)
	err := c.Initialize(context.Background())
	e, ok := err.(*Error)
	if !ok || e.Kind != KindVersionMismatch {
		t.Fatalf("want KindVersionMismatch, got %v", err)
	}
}

func TestNoSessionHeader(t *testing.T) {
	srv := newRPCServer(t, func(w http.ResponseWriter, r *http.Request, req rpcReq) {
		// Forget to set Mcp-Session-Id.
		w.Header().Set("Content-Type", "application/json")
		body, _ := json.Marshal(map[string]any{"jsonrpc": "2.0", "id": req.ID, "result": map[string]any{"protocolVersion": ProtocolVersion}})
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(body)
	})
	defer srv.Close()
	c := New(srv.URL, "t", time.Second)
	if err := c.Initialize(context.Background()); err == nil || !strings.Contains(err.Error(), "Mcp-Session-Id") {
		t.Fatalf("want session-header error, got %v", err)
	}
}

func TestResourcesList(t *testing.T) {
	srv := newRPCServer(t, func(w http.ResponseWriter, r *http.Request, req rpcReq) {
		switch req.Method {
		case "initialize":
			writeResult(w, "s", req.ID, map[string]any{"protocolVersion": ProtocolVersion})
		case "resources/list":
			writeResult(w, "s", req.ID, map[string]any{"resources": []map[string]any{
				{"uri": "citadel://ns/x", "name": "x"},
			}})
		}
	})
	defer srv.Close()
	c := New(srv.URL, "t", time.Second)
	if err := c.Initialize(context.Background()); err != nil {
		t.Fatal(err)
	}
	rs, err := c.ResourcesList(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(rs) != 1 || rs[0].URI != "citadel://ns/x" {
		t.Fatalf("got %+v", rs)
	}
}

func TestResourcesRead(t *testing.T) {
	srv := newRPCServer(t, func(w http.ResponseWriter, r *http.Request, req rpcReq) {
		switch req.Method {
		case "initialize":
			writeResult(w, "s", req.ID, map[string]any{"protocolVersion": ProtocolVersion})
		case "resources/read":
			writeResult(w, "s", req.ID, map[string]any{"contents": []map[string]any{
				{"uri": "citadel://ns/x", "mimeType": "application/json", "text": "{}"},
			}})
		}
	})
	defer srv.Close()
	c := New(srv.URL, "t", time.Second)
	if err := c.Initialize(context.Background()); err != nil {
		t.Fatal(err)
	}
	raw, err := c.ResourcesRead(context.Background(), "citadel://ns/x")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(raw), "citadel://ns/x") {
		t.Fatalf("raw missing uri: %s", raw)
	}
}

func TestPromptsList(t *testing.T) {
	srv := newRPCServer(t, func(w http.ResponseWriter, r *http.Request, req rpcReq) {
		switch req.Method {
		case "initialize":
			writeResult(w, "s", req.ID, map[string]any{"protocolVersion": ProtocolVersion})
		case "prompts/list":
			writeResult(w, "s", req.ID, map[string]any{"prompts": []map[string]any{
				{"name": "issue_template", "description": "Open an issue"},
			}})
		}
	})
	defer srv.Close()
	c := New(srv.URL, "t", time.Second)
	if err := c.Initialize(context.Background()); err != nil {
		t.Fatal(err)
	}
	ps, err := c.PromptsList(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(ps) != 1 || ps[0].Name != "issue_template" {
		t.Fatalf("got %+v", ps)
	}
}

func TestPromptsGet(t *testing.T) {
	srv := newRPCServer(t, func(w http.ResponseWriter, r *http.Request, req rpcReq) {
		switch req.Method {
		case "initialize":
			writeResult(w, "s", req.ID, map[string]any{"protocolVersion": ProtocolVersion})
		case "prompts/get":
			writeResult(w, "s", req.ID, map[string]any{
				"description": "Open an issue",
				"messages": []map[string]any{
					{"role": "user", "content": map[string]any{"type": "text", "text": "Title?"}},
				},
			})
		}
	})
	defer srv.Close()
	c := New(srv.URL, "t", time.Second)
	if err := c.Initialize(context.Background()); err != nil {
		t.Fatal(err)
	}
	raw, err := c.PromptsGet(context.Background(), "issue_template", map[string]any{"k": "v"})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(raw), "Title?") {
		t.Fatalf("raw missing content: %s", raw)
	}
}

func TestResourcesList_NotInitialized(t *testing.T) {
	c := New("http://nope", "t", time.Second)
	if _, err := c.ResourcesList(context.Background()); err == nil || !strings.Contains(err.Error(), "not initialized") {
		t.Fatalf("want not-initialized, got %v", err)
	}
	if _, err := c.ResourcesRead(context.Background(), "x"); err == nil || !strings.Contains(err.Error(), "not initialized") {
		t.Fatalf("want not-initialized, got %v", err)
	}
	if _, err := c.PromptsList(context.Background()); err == nil || !strings.Contains(err.Error(), "not initialized") {
		t.Fatalf("want not-initialized, got %v", err)
	}
	if _, err := c.PromptsGet(context.Background(), "x", nil); err == nil || !strings.Contains(err.Error(), "not initialized") {
		t.Fatalf("want not-initialized, got %v", err)
	}
}

func TestClassifyJSONRPCError_AllKinds(t *testing.T) {
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
