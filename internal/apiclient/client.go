// Package apiclient is the Bearer-authenticated HTTP wrapper used by every
// citadel-cli subcommand handler against the Citadel JSON API. It centralises
// the load-config / require-token / build-request / decode-or-error chain.
//
// Non-2xx responses surface as *HTTPError; handlers that need to branch on
// status (e.g. 412 mfa_required) use errors.As to recover it.
package apiclient

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/Rethunk-Tech/citadel-cli/internal/clicfg"
	"github.com/Rethunk-Tech/citadel-cli/internal/httpx"
)

// defaultTimeout is the per-request timeout applied to the api client's
// underlying http.Client. Per-request context deadlines may override.
const defaultTimeout = 30 * time.Second

// HTTPError is returned for any non-2xx response. Body is the trimmed
// response payload (best-effort read; may be empty).
type HTTPError struct {
	StatusCode int
	Body       string
	// RetryAfter is seconds from the Retry-After header when parseable (RFC 7231
	// delta-seconds or HTTP-date); zero when absent or malformed.
	RetryAfter int
}

func (e *HTTPError) Error() string {
	return fmt.Sprintf("server error %d: %s", e.StatusCode, e.Body)
}

// DecodeBody decodes the (best-effort-trimmed) response body as JSON into v.
// Returns nil if Body is empty or v is nil.
func (e *HTTPError) DecodeBody(v any) error {
	if v == nil || e.Body == "" {
		return nil
	}
	return json.Unmarshal([]byte(e.Body), v)
}

// IsStatus reports whether err wraps an *HTTPError with the given status code.
func IsStatus(err error, code int) bool {
	var he *HTTPError
	return errors.As(err, &he) && he.StatusCode == code
}

// ParseRetryAfterSeconds interprets a Retry-After header value: non-negative
// integer seconds, or HTTP-date; returns 0 when absent or unparseable.
func ParseRetryAfterSeconds(raw string) int {
	v := strings.TrimSpace(raw)
	if v == "" {
		return 0
	}
	if secs, err := strconv.Atoi(v); err == nil && secs >= 0 {
		return secs
	}
	if t, err := http.ParseTime(v); err == nil {
		d := time.Until(t)
		if d < 0 {
			return 0
		}
		return int(math.Round(d.Seconds()))
	}
	return 0
}

// Client is a thin Citadel-API client: server URL + bearer token + http.Client.
type Client struct {
	server string
	token  string
	http   *http.Client
}

// Options are the per-invocation knobs surfaced by root persistent flags.
// Server is the resolved --server flag value; Verbose / DebugHTTP wire the
// trace transport; retry/backoff is always on for idempotent verbs.
type Options struct {
	Server    string
	Verbose   bool
	DebugHTTP bool
}

// New builds a Client from a loaded clicfg.Config and the resolved CLI
// options. Returns the canonical "not authenticated" error if no token is
// configured.
func New(cfg clicfg.Config, opts Options) (*Client, error) {
	if cfg.AccessToken == "" {
		return nil, errors.New("not authenticated; run 'citadel-cli auth login' first")
	}
	rt := httpx.Stack(nil, httpx.Options{Verbose: opts.Verbose, DebugHTTP: opts.DebugHTTP})
	return &Client{
		server: strings.TrimRight(cfg.ResolveServerURL(opts.Server), "/"),
		token:  cfg.AccessToken,
		http:   &http.Client{Timeout: defaultTimeout, Transport: rt},
	}, nil
}

// Server returns the resolved base URL with no trailing slash.
func (c *Client) Server() string { return c.server }

// Token returns the bearer token (for handlers that still need to build a
// custom request, e.g. streaming endpoints).
func (c *Client) Token() string { return c.token }

// Get sends GET path; decodes JSON body into out (pass nil to discard).
func (c *Client) Get(ctx context.Context, path string, out any) error {
	return c.do(ctx, http.MethodGet, path, nil, out)
}

// Post sends POST path with JSON-encoded body; decodes response into out.
func (c *Client) Post(ctx context.Context, path string, body, out any) error {
	return c.do(ctx, http.MethodPost, path, body, out)
}

// Put sends PUT path with JSON-encoded body; decodes response into out.
func (c *Client) Put(ctx context.Context, path string, body, out any) error {
	return c.do(ctx, http.MethodPut, path, body, out)
}

// Delete sends DELETE path; ignores any response body.
func (c *Client) Delete(ctx context.Context, path string) error {
	return c.do(ctx, http.MethodDelete, path, nil, nil)
}

func (c *Client) do(ctx context.Context, method, path string, body, out any) error {
	var rdr io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return fmt.Errorf("marshal request: %w", err)
		}
		rdr = bytes.NewReader(b)
	}
	req, err := http.NewRequestWithContext(ctx, method, c.server+path, rdr)
	if err != nil {
		return fmt.Errorf("build request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+c.token)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	resp, err := c.http.Do(req)
	if err != nil {
		return fmt.Errorf("request failed: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		b, _ := io.ReadAll(resp.Body)
		return &HTTPError{
			StatusCode: resp.StatusCode,
			Body:       strings.TrimSpace(string(b)),
			RetryAfter: ParseRetryAfterSeconds(resp.Header.Get("Retry-After")),
		}
	}
	if out == nil || resp.StatusCode == http.StatusNoContent {
		return nil
	}
	if err := json.NewDecoder(resp.Body).Decode(out); err != nil {
		return fmt.Errorf("decode response: %w", err)
	}
	return nil
}
