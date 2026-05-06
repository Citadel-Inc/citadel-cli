package httpx

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
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

type boomReader struct{}

func (boomReader) Read([]byte) (int, error) { return 0, errors.New("read body") }

// A GET with a body is unusual but legal; replayable transports must still
// surface body read failures instead of sending a corrupt stream.
func TestRetryTransport_ReadBodyError(t *testing.T) {
	req, err := http.NewRequestWithContext(t.Context(), http.MethodGet, "http://example.test/", io.NopCloser(boomReader{}))
	if err != nil {
		t.Fatal(err)
	}
	rt := &RetryTransport{Base: http.DefaultTransport}
	_, err = rt.RoundTrip(req)
	if err == nil || !strings.Contains(err.Error(), "read request body") {
		t.Fatalf("want body read error, got %v", err)
	}
}

type seqRoundTripper struct {
	n   int
	ok  http.RoundTripper
	err error
}

func (s *seqRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	s.n++
	if s.n == 1 {
		return nil, s.err
	}
	return s.ok.RoundTrip(req)
}

func TestRetryTransport_RetriesAfterBaseTransportError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	t.Cleanup(srv.Close)

	base := &seqRoundTripper{ok: srv.Client().Transport}
	base.err = fmt.Errorf("transient: %w", io.ErrUnexpectedEOF)

	c := &http.Client{Transport: &RetryTransport{Base: base}}
	req, err := http.NewRequestWithContext(t.Context(), http.MethodGet, srv.URL, nil)
	if err != nil {
		t.Fatal(err)
	}
	resp, err := c.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	_, _ = io.Copy(io.Discard, resp.Body)
	_ = resp.Body.Close()
	if base.n < 2 {
		t.Fatalf("expected retry after transport error, got %d calls", base.n)
	}
}

func TestSleepCtx_Canceled(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "http://example.test/", nil)
	if err != nil {
		t.Fatal(err)
	}
	if err := sleepCtx(req, time.Hour); !errors.Is(err, context.Canceled) {
		t.Fatalf("want context.Canceled, got %v", err)
	}
}
