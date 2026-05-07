package apiclient

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/Rethunk-Tech/citadel-cli/internal/clicfg"
)

func TestNew_RequiresToken(t *testing.T) {
	if _, err := New(clicfg.Config{}, Options{}); err == nil || !strings.Contains(err.Error(), "not authenticated") {
		t.Fatalf("expected not-authenticated error, got %v", err)
	}
}

func TestNew_CoercesProductionRESTHost(t *testing.T) {
	c, err := New(clicfg.Config{ServerURL: "https://mcp.src.land", AccessToken: "tok"}, Options{})
	if err != nil {
		t.Fatal(err)
	}
	if got := c.Server(); got != "https://api.src.land" {
		t.Fatalf("server = %q, want https://api.src.land", got)
	}
}

func TestNew_LeavesCustomHostUntouched(t *testing.T) {
	c, err := New(clicfg.Config{ServerURL: "http://127.0.0.1:7777", AccessToken: "tok"}, Options{})
	if err != nil {
		t.Fatal(err)
	}
	if got := c.Server(); got != "http://127.0.0.1:7777" {
		t.Fatalf("server = %q", got)
	}
}

func TestClient_GetDecodes(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("Authorization"); got != "Bearer tok" {
			t.Errorf("missing/incorrect bearer header: %q", got)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"name":"alice"}`))
	}))
	defer srv.Close()

	c, err := New(clicfg.Config{ServerURL: srv.URL, AccessToken: "tok"}, Options{})
	if err != nil {
		t.Fatal(err)
	}
	var out struct{ Name string }
	if err := c.Get(context.Background(), "/whoami", &out); err != nil {
		t.Fatal(err)
	}
	if out.Name != "alice" {
		t.Fatalf("got %q", out.Name)
	}
}

func TestClient_Get401RetryAfterHook(t *testing.T) {
	var n int
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		auth := r.Header.Get("Authorization")
		n++
		switch {
		case n == 1 && auth == "Bearer old":
			w.WriteHeader(http.StatusUnauthorized)
			_, _ = w.Write([]byte(`{"error":"nope"}`))
		case n == 2 && auth == "Bearer new":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"name":"alice"}`))
		default:
			t.Errorf("unexpected request #%d auth %q", n, auth)
			w.WriteHeader(http.StatusTeapot)
		}
	}))
	defer srv.Close()

	hookCalls := 0
	c, err := New(clicfg.Config{ServerURL: srv.URL, AccessToken: "old"}, Options{
		RetryOn401: func(ctx context.Context) (string, error) {
			hookCalls++
			return "new", nil
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	var out struct{ Name string }
	if err := c.Get(context.Background(), "/v", &out); err != nil {
		t.Fatal(err)
	}
	if out.Name != "alice" {
		t.Fatalf("got %+v", out)
	}
	if hookCalls != 1 {
		t.Fatalf("hook calls = %d", hookCalls)
	}
	if n != 2 {
		t.Fatalf("server requests = %d", n)
	}
}

func TestClient_Get401TwiceAfterHookReturnsHTTPError(t *testing.T) {
	var n int
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		n++
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte("no"))
	}))
	defer srv.Close()

	c, err := New(clicfg.Config{ServerURL: srv.URL, AccessToken: "old"}, Options{
		RetryOn401: func(ctx context.Context) (string, error) {
			return "new", nil
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	err = c.Get(context.Background(), "/v", nil)
	if err == nil {
		t.Fatal("expected error")
	}
	if !IsStatus(err, http.StatusUnauthorized) {
		t.Fatalf("got %v", err)
	}
	if n != 2 {
		t.Fatalf("want 2 server hits, got %d", n)
	}
}

func TestClient_PostSendsJSON(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Content-Type") != "application/json" {
			t.Errorf("missing content-type")
		}
		var body struct{ X int }
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatal(err)
		}
		if body.X != 42 {
			t.Errorf("got %d", body.X)
		}
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte(`{"id":"abc"}`))
	}))
	defer srv.Close()

	c, _ := New(clicfg.Config{ServerURL: srv.URL, AccessToken: "tok"}, Options{})
	var out struct{ ID string }
	if err := c.Post(context.Background(), "/things", map[string]int{"x": 42}, &out); err != nil {
		t.Fatal(err)
	}
	if out.ID != "abc" {
		t.Fatalf("got %q", out.ID)
	}
}

func TestClient_DeleteAcceptsNoContent(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))
	defer srv.Close()

	c, _ := New(clicfg.Config{ServerURL: srv.URL, AccessToken: "tok"}, Options{})
	if err := c.Delete(context.Background(), "/things/1"); err != nil {
		t.Fatal(err)
	}
}

func TestClient_NonSuccessReturnsHTTPError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusForbidden)
		_, _ = w.Write([]byte("forbidden body"))
	}))
	defer srv.Close()

	c, _ := New(clicfg.Config{ServerURL: srv.URL, AccessToken: "tok"}, Options{})
	err := c.Get(context.Background(), "/x", nil)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	var he *HTTPError
	if !errors.As(err, &he) {
		t.Fatalf("expected *HTTPError, got %T", err)
	}
	if he.StatusCode != http.StatusForbidden {
		t.Errorf("got status %d", he.StatusCode)
	}
	if he.Body != "forbidden body" {
		t.Errorf("got body %q", he.Body)
	}
}

func TestClient_NonSuccessCapturesRetryAfter(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Retry-After", "45")
		// 403 is not retried on GET; 429 would loop in RetryTransport.
		w.WriteHeader(http.StatusForbidden)
		_, _ = w.Write([]byte(`{"error":"nope"}`))
	}))
	defer srv.Close()

	c, _ := New(clicfg.Config{ServerURL: srv.URL, AccessToken: "tok"}, Options{})
	err := c.Get(context.Background(), "/x", nil)
	if err == nil {
		t.Fatal("expected error")
	}
	var he *HTTPError
	if !errors.As(err, &he) {
		t.Fatalf("got %T: %v", err, err)
	}
	if he.StatusCode != http.StatusForbidden || he.RetryAfter != 45 {
		t.Fatalf("HTTPError = %#v", he)
	}
}

// POST is not auto-retried by the transport; Retry-After can use a large value.
func TestClient_Post429CapturesRetryAfterSeconds(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Retry-After", "45")
		w.WriteHeader(http.StatusTooManyRequests)
		_, _ = w.Write([]byte(`{"error":"slow"}`))
	}))
	defer srv.Close()

	c, _ := New(clicfg.Config{ServerURL: srv.URL, AccessToken: "tok"}, Options{})
	err := c.Post(context.Background(), "/x", map[string]string{"k": "v"}, nil)
	if err == nil {
		t.Fatal("expected error")
	}
	var he *HTTPError
	if !errors.As(err, &he) {
		t.Fatalf("got %T", err)
	}
	if he.StatusCode != http.StatusTooManyRequests || he.RetryAfter != 45 {
		t.Fatalf("HTTPError = %#v", he)
	}
}

func TestHTTPError_ErrorAndDecodeBody(t *testing.T) {
	he := &HTTPError{StatusCode: 409, Body: `{"error":"has_repos","detail":"x"}`}
	if got := he.Error(); !strings.Contains(got, "409") || !strings.Contains(got, "has_repos") {
		t.Fatalf("Error() = %q", got)
	}
	var body struct {
		Error  string `json:"error"`
		Detail string `json:"detail"`
	}
	if err := he.DecodeBody(&body); err != nil {
		t.Fatalf("DecodeBody: %v", err)
	}
	if body.Error != "has_repos" || body.Detail != "x" {
		t.Fatalf("decoded %+v", body)
	}
	// nil target and empty body each short-circuit.
	if err := he.DecodeBody(nil); err != nil {
		t.Fatalf("DecodeBody(nil): %v", err)
	}
	if err := (&HTTPError{Body: ""}).DecodeBody(&body); err != nil {
		t.Fatalf("DecodeBody empty: %v", err)
	}
}

func TestIsStatus(t *testing.T) {
	err := &HTTPError{StatusCode: 404}
	if !IsStatus(err, 404) {
		t.Fatal("expected match on direct *HTTPError")
	}
	wrapped := errors.Join(errors.New("ctx"), err)
	if !IsStatus(wrapped, 404) {
		t.Fatal("expected match through errors.Join")
	}
	if IsStatus(err, 500) {
		t.Fatal("did not expect match for different code")
	}
	if IsStatus(errors.New("plain"), 404) {
		t.Fatal("plain errors must not match")
	}
	if IsStatus(nil, 404) {
		t.Fatal("nil must not match")
	}
}

func TestClient_ServerAndToken(t *testing.T) {
	c, err := New(clicfg.Config{ServerURL: "https://x.test/", AccessToken: "tok"}, Options{})
	if err != nil {
		t.Fatal(err)
	}
	if got := c.Server(); got != "https://x.test" {
		t.Fatalf("Server() = %q", got)
	}
	if got := c.Token(); got != "tok" {
		t.Fatalf("Token() = %q", got)
	}
}

func TestClient_PatchSendsJSON(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPatch {
			t.Errorf("method = %s", r.Method)
		}
		var body struct{ Name string }
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatal(err)
		}
		if body.Name != "x" {
			t.Errorf("got %q", body.Name)
		}
		w.WriteHeader(http.StatusNoContent)
	}))
	defer srv.Close()

	c, _ := New(clicfg.Config{ServerURL: srv.URL, AccessToken: "tok"}, Options{})
	if err := c.Patch(context.Background(), "/things/1", map[string]string{"name": "x"}, nil); err != nil {
		t.Fatal(err)
	}
}

func TestClient_PutSendsJSON(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPut {
			t.Errorf("method = %s", r.Method)
		}
		var body struct{ Y int }
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatal(err)
		}
		if body.Y != 7 {
			t.Errorf("got %d", body.Y)
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"ok":true}`))
	}))
	defer srv.Close()

	c, _ := New(clicfg.Config{ServerURL: srv.URL, AccessToken: "tok"}, Options{})
	var out struct{ OK bool }
	if err := c.Put(context.Background(), "/things/1", map[string]int{"y": 7}, &out); err != nil {
		t.Fatal(err)
	}
	if !out.OK {
		t.Fatalf("got %+v", out)
	}
}

func TestClient_ContextCancellation(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	c, _ := New(clicfg.Config{ServerURL: srv.URL, AccessToken: "tok"}, Options{})
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if err := c.Get(ctx, "/x", nil); err == nil || !errors.Is(err, context.Canceled) {
		t.Fatalf("expected ctx.Canceled, got %v", err)
	}
}

func TestClient_GetEventStream_success(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer tok" {
			t.Errorf("Authorization = %q", r.Header.Get("Authorization"))
		}
		if !strings.Contains(r.Header.Get("Accept"), "text/event-stream") {
			t.Errorf("Accept = %q", r.Header.Get("Accept"))
		}
		if got := r.Header.Get("Last-Event-ID"); got != "42" {
			t.Errorf("Last-Event-ID = %q", got)
		}
		w.Header().Set("Content-Type", "text/event-stream")
		_, _ = w.Write([]byte("data: ok\n\n"))
	}))
	defer srv.Close()

	c, err := New(clicfg.Config{ServerURL: srv.URL, AccessToken: "tok"}, Options{})
	if err != nil {
		t.Fatal(err)
	}
	resp, err := c.GetEventStream(context.Background(), "/stream", "42")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = resp.Body.Close() }()
	b, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(b), "data: ok") {
		t.Fatalf("body = %q", string(b))
	}
}

func TestClient_GetEventStream_IgnoresRequestTimeoutWhileBodyStreams(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		fl, ok := w.(http.Flusher)
		if !ok {
			t.Fatal("responseWriter does not support Flush")
		}
		w.Header().Set("Content-Type", "text/event-stream")
		fl.Flush()
		time.Sleep(60 * time.Millisecond)
		_, _ = w.Write([]byte("data: ok\n\n"))
		fl.Flush()
	}))
	defer srv.Close()

	c, err := New(clicfg.Config{ServerURL: srv.URL, AccessToken: "tok"}, Options{})
	if err != nil {
		t.Fatal(err)
	}
	c.http.Timeout = 25 * time.Millisecond

	resp, err := c.GetEventStream(context.Background(), "/stream", "")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = resp.Body.Close() }()

	b, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("stream body read failed: %v", err)
	}
	if !strings.Contains(string(b), "data: ok") {
		t.Fatalf("body = %q", string(b))
	}
}

func TestClient_GetEventStream_nonSuccessDrainsBody(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte(`missing`))
	}))
	defer srv.Close()

	c, _ := New(clicfg.Config{ServerURL: srv.URL, AccessToken: "tok"}, Options{})
	resp, err := c.GetEventStream(context.Background(), "/nope", "")
	if err == nil {
		if resp != nil {
			_ = resp.Body.Close()
		}
		t.Fatal("expected error")
	}
	var he *HTTPError
	if !errors.As(err, &he) || he.StatusCode != http.StatusNotFound || he.Body != "missing" {
		t.Fatalf("err = %v", err)
	}
}

func TestClient_GetEventStream_retryAfterOnError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Retry-After", "12")
		w.WriteHeader(http.StatusGone)
		_, _ = w.Write([]byte("gone"))
	}))
	defer srv.Close()

	c, _ := New(clicfg.Config{ServerURL: srv.URL, AccessToken: "tok"}, Options{})
	_, err := c.GetEventStream(context.Background(), "/gone", "")
	var he *HTTPError
	if !errors.As(err, &he) || he.StatusCode != http.StatusGone || he.RetryAfter != 12 {
		t.Fatalf("HTTPError = %#v", he)
	}
}
