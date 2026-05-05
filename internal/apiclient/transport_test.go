package apiclient

import (
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"

	"github.com/Rethunk-Tech/citadel-cli/internal/clicfg"
)

// TestRetryOn503 asserts an idempotent GET retries through a 503 and
// succeeds on the next attempt.
func TestRetryOn503(t *testing.T) {
	var calls atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if calls.Add(1) == 1 {
			w.WriteHeader(503)
			return
		}
		w.WriteHeader(200)
		_, _ = w.Write([]byte(`{"ok":true}`))
	}))
	defer srv.Close()

	c, err := New(clicfg.Config{ServerURL: srv.URL, AccessToken: "tok"}, Options{})
	if err != nil {
		t.Fatal(err)
	}
	var out struct {
		OK bool `json:"ok"`
	}
	if err := c.Get(t.Context(), "/x", &out); err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got := calls.Load(); got != 2 {
		t.Fatalf("expected 2 calls (1 retry), got %d", got)
	}
	if !out.OK {
		t.Fatal("decoded body lost")
	}
}

// TestNoRetryOnPOST asserts a non-idempotent POST does NOT retry on 503.
func TestNoRetryOnPOST(t *testing.T) {
	var calls atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		w.WriteHeader(503)
	}))
	defer srv.Close()

	c, _ := New(clicfg.Config{ServerURL: srv.URL, AccessToken: "tok"}, Options{})
	_ = c.Post(t.Context(), "/x", map[string]string{"a": "b"}, nil)
	if got := calls.Load(); got != 1 {
		t.Fatalf("POST must not retry; got %d calls", got)
	}
}
