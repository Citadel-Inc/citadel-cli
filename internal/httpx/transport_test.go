package httpx

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"
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
	}))
	defer srv.Close()

	c := &http.Client{Transport: Stack(nil, Options{})}
	req, _ := http.NewRequestWithContext(t.Context(), http.MethodGet, srv.URL, nil)
	resp, err := c.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	_, _ = io.Copy(io.Discard, resp.Body)
	_ = resp.Body.Close()
	if got := calls.Load(); got != 2 {
		t.Fatalf("expected 2 calls (1 retry), got %d", got)
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

	c := &http.Client{Transport: Stack(nil, Options{})}
	req, _ := http.NewRequestWithContext(t.Context(), http.MethodPost, srv.URL, nil)
	resp, _ := c.Do(req)
	if resp != nil {
		_ = resp.Body.Close()
	}
	if got := calls.Load(); got != 1 {
		t.Fatalf("POST must not retry; got %d calls", got)
	}
}

// TestStackBaseDefault ensures Stack(nil, _) wraps DefaultTransport without
// panicking.
func TestStackBaseDefault(t *testing.T) {
	rt := Stack(nil, Options{})
	if rt == nil {
		t.Fatal("Stack returned nil")
	}
	_ = context.Background()
}

func TestRetryAfterDelay_ParseSeconds(t *testing.T) {
	h := http.Header{}
	if got := retryAfterDelay(h); got != 0 {
		t.Fatalf("empty: %v", got)
	}
	h.Set("Retry-After", "3")
	if got := retryAfterDelay(h); got != 3*time.Second {
		t.Fatalf("got %v", got)
	}
	h.Set("Retry-After", "not-a-number")
	if got := retryAfterDelay(h); got != 0 {
		t.Fatalf("malformed: %v", got)
	}
	h.Set("Retry-After", "-1")
	if got := retryAfterDelay(h); got != 0 {
		t.Fatalf("negative: %v", got)
	}
}
