package httpx

import (
	"bytes"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
)

// User runs with -v: one METHOD URL → STATUS line per successful call.
func TestVerboseTransport_LogsLineOnSuccess(t *testing.T) {
	pr, pw, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	old := os.Stderr
	os.Stderr = pw
	var buf bytes.Buffer
	done := make(chan struct{})
	go func() {
		_, _ = io.Copy(&buf, pr)
		close(done)
	}()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusTeapot)
	}))
	t.Cleanup(srv.Close)

	c := &http.Client{Transport: Stack(nil, Options{Verbose: true})}
	req, err := http.NewRequest(http.MethodGet, srv.URL+"/x", nil)
	if err != nil {
		t.Fatal(err)
	}
	resp, err := c.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	_, _ = io.Copy(io.Discard, resp.Body)
	_ = resp.Body.Close()

	os.Stderr = old
	_ = pw.Close()
	<-done
	_ = pr.Close()

	out := buf.String()
	if !strings.Contains(out, "GET") || !strings.Contains(out, "418") {
		t.Fatalf("expected verbose line with GET and 418, got: %q", out)
	}
}

// User runs with --debug-http: full dump must not leak raw bearer tokens.
func TestDebugHTTP_RedactsAuthorization(t *testing.T) {
	pr, pw, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	old := os.Stderr
	os.Stderr = pw
	var buf bytes.Buffer
	done := make(chan struct{})
	go func() {
		_, _ = io.Copy(&buf, pr)
		close(done)
	}()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("{}"))
	}))
	t.Cleanup(srv.Close)

	c := &http.Client{Transport: Stack(nil, Options{DebugHTTP: true})}
	req, err := http.NewRequest(http.MethodGet, srv.URL+"/secret", nil)
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Authorization", "Bearer super-secret-token")
	resp, err := c.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	_, _ = io.Copy(io.Discard, resp.Body)
	_ = resp.Body.Close()

	os.Stderr = old
	_ = pw.Close()
	<-done
	_ = pr.Close()

	out := buf.String()
	if strings.Contains(out, "super-secret-token") {
		t.Fatal("debug dump leaked bearer token")
	}
	if !strings.Contains(out, "Authorization") {
		t.Fatalf("expected Authorization header in dump, got: %q", out)
	}
}

// Transient 429 then success mirrors rate-limit then happy path.
func TestRetryOn429ThenSuccess(t *testing.T) {
	var calls int
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		if calls == 1 {
			w.Header().Set("Retry-After", "0")
			w.WriteHeader(http.StatusTooManyRequests)
			return
		}
		w.WriteHeader(http.StatusOK)
	}))
	t.Cleanup(srv.Close)

	c := &http.Client{Transport: Stack(nil, Options{})}
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
	if calls < 2 {
		t.Fatalf("expected retry after 429, calls=%d", calls)
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("want 200, got %d", resp.StatusCode)
	}
}
