package apiclient

import (
	"bytes"
	"fmt"
	"io"
	"math/rand/v2"
	"net/http"
	"net/http/httputil"
	"os"
	"strconv"
	"time"
)

// Default retry policy for idempotent verbs against transient failures.
const (
	retryAttempts    = 3
	retryBaseBackoff = 250 * time.Millisecond
	retryMaxBackoff  = 4 * time.Second
)

// retryableStatus reports whether a response status code is worth retrying on
// an idempotent verb. 408, 425, 429, and 5xx (except 501 Not Implemented).
func retryableStatus(code int) bool {
	switch code {
	case 408, 425, 429, 500, 502, 503, 504:
		return true
	}
	return false
}

// idempotentMethod reports whether a method is safe to retry transparently.
func idempotentMethod(m string) bool {
	switch m {
	case http.MethodGet, http.MethodHead, http.MethodOptions, http.MethodDelete, http.MethodPut:
		return true
	}
	return false
}

// retryAfterDelay parses a Retry-After header (seconds-only form). Returns
// zero if absent or malformed; caller falls back to expo backoff.
func retryAfterDelay(h http.Header) time.Duration {
	v := h.Get("Retry-After")
	if v == "" {
		return 0
	}
	secs, err := strconv.Atoi(v)
	if err != nil || secs < 0 {
		return 0
	}
	return time.Duration(secs) * time.Second
}

// backoff returns the expo-jittered delay for attempt n (0-indexed).
func backoff(n int) time.Duration {
	d := min(retryBaseBackoff<<n, retryMaxBackoff)
	// full jitter: random in [d/2, d]
	half := d / 2
	return half + time.Duration(rand.Int64N(int64(half)+1))
}

// retryTransport wraps a base RoundTripper with idempotent-verb retry on
// transient errors and 5xx/429/408/425 responses.
type retryTransport struct {
	base http.RoundTripper
}

func (rt *retryTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	if !idempotentMethod(req.Method) {
		return rt.base.RoundTrip(req)
	}
	// Buffer the body so we can replay on retry. Empty body == nil-safe.
	var body []byte
	if req.Body != nil {
		b, err := io.ReadAll(req.Body)
		_ = req.Body.Close()
		if err != nil {
			return nil, fmt.Errorf("read request body: %w", err)
		}
		body = b
	}

	var lastErr error
	for attempt := range retryAttempts {
		if body != nil {
			req.Body = io.NopCloser(bytes.NewReader(body))
		}
		resp, err := rt.base.RoundTrip(req)
		if err == nil && !retryableStatus(resp.StatusCode) {
			return resp, nil
		}
		if err == nil {
			// Non-retryable HTTP failure on the last attempt: surface it.
			if attempt == retryAttempts-1 {
				return resp, nil
			}
			delay := retryAfterDelay(resp.Header)
			_ = resp.Body.Close()
			if delay == 0 {
				delay = backoff(attempt)
			}
			if err := sleepCtx(req, delay); err != nil {
				return nil, err
			}
			continue
		}
		lastErr = err
		if attempt == retryAttempts-1 {
			return nil, err
		}
		if err := sleepCtx(req, backoff(attempt)); err != nil {
			return nil, err
		}
	}
	return nil, lastErr
}

// sleepCtx waits d, or returns ctx.Err() if the request context fires first.
func sleepCtx(req *http.Request, d time.Duration) error {
	t := time.NewTimer(d)
	defer t.Stop()
	select {
	case <-t.C:
		return nil
	case <-req.Context().Done():
		return req.Context().Err()
	}
}

// traceTransport dumps redacted requests/responses to stderr.
//
// Verbose: one METHOD URL → STATUS line per call.
// DebugHTTP: full headers + body, with Authorization scrubbed.
type traceTransport struct {
	base      http.RoundTripper
	verbose   bool
	debugHTTP bool
}

func (tt *traceTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	start := time.Now()
	if tt.debugHTTP {
		dump, _ := httputil.DumpRequestOut(redactAuth(req), true)
		fmt.Fprintf(os.Stderr, "--- HTTP request ---\n%s\n", dump)
	}
	resp, err := tt.base.RoundTrip(req)
	dur := time.Since(start)
	if err != nil {
		if tt.verbose || tt.debugHTTP {
			fmt.Fprintf(os.Stderr, "%s %s → error after %s: %v\n", req.Method, req.URL, dur.Round(time.Millisecond), err)
		}
		return resp, err
	}
	if tt.debugHTTP {
		dump, _ := httputil.DumpResponse(resp, true)
		fmt.Fprintf(os.Stderr, "--- HTTP response (%s) ---\n%s\n", dur.Round(time.Millisecond), dump)
	} else if tt.verbose {
		fmt.Fprintf(os.Stderr, "%s %s → %d in %s\n", req.Method, req.URL, resp.StatusCode, dur.Round(time.Millisecond))
	}
	return resp, nil
}

// redactAuth shallow-clones req with the Authorization header masked, so
// httputil.DumpRequestOut never leaks the bearer token to stderr.
func redactAuth(req *http.Request) *http.Request {
	clone := req.Clone(req.Context())
	if clone.Header.Get("Authorization") != "" {
		clone.Header.Set("Authorization", "Bearer <redacted>")
	}
	return clone
}
