// Package apiclient is the Bearer-authenticated HTTP wrapper used by every
// citadel-cli subcommand handler against the Citadel JSON API. It centralizes
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
	"net"
	"net/http"
	"net/url"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/Citadel-Inc/citadel-cli/internal/clicfg"
	"github.com/Citadel-Inc/citadel-cli/internal/httpx"
	"github.com/Rethunk-Tech/go-httpkit"
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

// TokenIgnoredHeader names why the server answered an optional-auth route as if
// no token had been sent: "confined", or the scope the route needs, or "refused".
const TokenIgnoredHeader = "X-Citadel-Token-Ignored" //nolint:gosec // a header name, not a credential

// TokenIgnoredError is an answer that only describes an anonymous caller, so it
// would mislead if shown as the answer for this token.
type TokenIgnoredError struct {
	Reason string
}

func (e *TokenIgnoredError) Error() string {
	switch e.Reason {
	case "confined":
		return "this token is confined to one namespace and cannot use this route, so the server treated the request as anonymous"
	case "refused":
		return "this token cannot use this route, so the server treated the request as anonymous"
	}
	return "this token lacks the " + e.Reason + " scope, so the server treated the request as anonymous and may be hiding the resource"
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

// ParseRetryAfterSeconds interprets a Retry-After header value: positive
// integer seconds, or HTTP-date; returns 0 when absent, unparseable or past.
func ParseRetryAfterSeconds(raw string) int {
	at, ok := httpkit.RetryAfter(strings.TrimSpace(raw))
	if !ok {
		return 0
	}
	return max(0, int(math.Round(time.Until(at).Seconds())))
}

// maxErrorBody bounds an error answer, which is only ever shown to the user.
const maxErrorBody = 1 << 20

// ReadErrorBody reads at most maxErrorBody bytes of a failed response's body.
func ReadErrorBody(r io.Reader) []byte {
	b, _ := io.ReadAll(io.LimitReader(r, maxErrorBody))
	return b
}

// Client is a thin Citadel-API client: server URL + bearer token + http.Client.
type Client struct {
	server     string
	token      string
	userAgent  string
	http       *http.Client
	streamHTTP *http.Client
	retryOn401 func(context.Context) (newAccessToken string, err error)
	writeToken func(context.Context) (string, error)
}

// Options are the per-invocation knobs surfaced by root persistent flags.
// Server is the resolved --server flag value; Verbose / DebugHTTP wire the
// trace transport; retry/backoff is always on for idempotent verbs.
//
// RetryOn401, when set, is invoked after a 401 response to obtain a new
// bearer token; the original request is retried exactly once with the new
// token. Used for agent-token rotation (see cmd rotate-on-401 wiring).
//
// UserAgent, when non-empty, is sent as the User-Agent header on every
// request. Defaults to "citadel-cli" when empty.
type Options struct {
	Server     string
	Verbose    bool
	DebugHTTP  bool
	UserAgent  string
	RetryOn401 func(ctx context.Context) (newAccessToken string, err error)
	// WriteToken, when set, supplies the bearer for requests other than GET and
	// HEAD. On error the client's own token is used, so a token that can still
	// write keeps working.
	WriteToken func(ctx context.Context) (string, error)
}

// New builds a Client from a loaded clicfg.Config and the resolved CLI
// options. Returns the canonical "not authenticated" error if no token is
// configured.
func New(cfg clicfg.Config, opts Options) (*Client, error) {
	if cfg.AccessToken == "" {
		return nil, errors.New("not authenticated; run 'citadel-cli auth login' first")
	}
	ua := opts.UserAgent
	if ua == "" {
		ua = "citadel-cli"
	}
	rt := httpx.Stack(nil, httpx.Options{Verbose: opts.Verbose, DebugHTTP: opts.DebugHTTP})
	return &Client{
		server:     ResolveRESTServerURL(cfg.ResolveRawServerURL(opts.Server)),
		token:      cfg.AccessToken,
		userAgent:  ua,
		http:       &http.Client{Timeout: defaultTimeout, Transport: rt},
		streamHTTP: &http.Client{Transport: rt},
		retryOn401: opts.RetryOn401,
		writeToken: opts.WriteToken,
	}, nil
}

// bearer picks the token for one request.
func (c *Client) bearer(ctx context.Context, method string) string {
	if c.writeToken == nil || method == http.MethodGet || method == http.MethodHead {
		return c.token
	}
	token, err := c.writeToken(ctx)
	if err != nil || token == "" {
		return c.token
	}
	return token
}

// ResolveRESTServerURL maps the configured server URL to the base REST paths
// are appended to. The src.land API lives at the root of api.src.land (the
// MCP host is coerced there), while the apex and every self-hosted server serve
// it under /api on the web origin. Loopback hosts are used as written.
func ResolveRESTServerURL(server string) string {
	base := strings.TrimRight(strings.TrimSpace(server), "/")
	u, err := url.Parse(base)
	if err != nil {
		return base
	}
	switch u.Host {
	case "mcp.src.land", "api.src.land":
		u.Host = "api.src.land"
		u.Path = strings.TrimSuffix(u.Path, "/api")
	default:
		// Every documented self-host layout (single-binary, Helm, air-gap)
		// serves the API under /api on the web origin, loopback included
		// (the air-gap default is http://localhost:8000). Only a loopback
		// server that does not answer /api/healthz is used as written, which
		// is how a stub or a bare dev backend behaves.
		if isLoopbackHost(u.Hostname()) && (!ProbeLoopback || !loopbackServesAPI(base)) {
			return base
		}
		if !strings.HasSuffix(u.Path, "/api") {
			u.Path += "/api"
		}
	}
	return strings.TrimRight(u.String(), "/")
}

// endpoint joins path to the REST base. Paths written with a leading /api work
// against a base that already ends in /api.
func (c *Client) endpoint(path string) string {
	if strings.HasSuffix(c.server, "/api") && strings.HasPrefix(path, "/api/") {
		path = strings.TrimPrefix(path, "/api")
	}
	return c.server + path
}

// ProbeLoopback turns the loopback front-door check on; test binaries that
// stub the API at the server root switch it off.
var ProbeLoopback = true

var loopbackProbe sync.Map

// loopbackServesAPI is false only when the server answers /api/healthz with
// 404, i.e. it has no /api mount. A refused or failed probe (a stack still
// settling) keeps the documented /api layout instead of sending every request
// to the web origin's HTML.
func loopbackServesAPI(base string) bool {
	if v, ok := loopbackProbe.Load(base); ok {
		if serves, isBool := v.(bool); isBool {
			return serves
		}
	}
	hc := &http.Client{Timeout: 2 * time.Second}
	resp, err := hc.Get(base + "/api/healthz") //nolint:noctx // the URL is resolved before any request context exists
	if err != nil {
		return true
	}
	_ = resp.Body.Close()
	ok := resp.StatusCode != http.StatusNotFound
	loopbackProbe.Store(base, ok)
	return ok
}

func isLoopbackHost(h string) bool {
	if strings.EqualFold(h, "localhost") {
		return true
	}
	ip := net.ParseIP(h)
	return ip != nil && ip.IsLoopback()
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

// Patch sends PATCH path with JSON-encoded body; decodes response into out.
func (c *Client) Patch(ctx context.Context, path string, body, out any) error {
	return c.do(ctx, http.MethodPatch, path, body, out)
}

// Delete sends DELETE path; ignores any response body.
func (c *Client) Delete(ctx context.Context, path string) error {
	return c.do(ctx, http.MethodDelete, path, nil, nil)
}

// GetEventStream issues a GET with Accept: text/event-stream for SSE consumers.
// The caller must close resp.Body. Non-success statuses return *HTTPError with
// the body drained.
func (c *Client) GetEventStream(ctx context.Context, path, lastEventID string) (*http.Response, error) {
	for attempt := range 2 {
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.endpoint(path), nil)
		if err != nil {
			return nil, fmt.Errorf("build request: %w", err)
		}
		req.Header.Set("Authorization", "Bearer "+c.token)
		req.Header.Set("User-Agent", c.userAgent)
		req.Header.Set("Accept", "text/event-stream")
		req.Header.Set("Cache-Control", "no-cache")
		if lastEventID != "" {
			req.Header.Set("Last-Event-ID", lastEventID)
		}
		resp, err := c.streamHTTP.Do(req)
		if err != nil {
			return nil, fmt.Errorf("request failed: %w", err)
		}
		if resp.StatusCode >= 200 && resp.StatusCode < 300 {
			return resp, nil
		}
		b := ReadErrorBody(resp.Body)
		_ = resp.Body.Close()
		if retry, rerr := c.refreshOn401(ctx, resp.StatusCode, attempt); rerr != nil {
			return nil, rerr
		} else if retry {
			continue
		}
		return nil, &HTTPError{
			StatusCode: resp.StatusCode,
			Body:       strings.TrimSpace(string(b)),
			RetryAfter: ParseRetryAfterSeconds(resp.Header.Get("Retry-After")),
		}
	}
	return nil, errors.New("event stream: exhausted 401 retries")
}

func (c *Client) do(ctx context.Context, method, path string, body, out any) error {
	var bodyBytes []byte
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return fmt.Errorf("marshal request: %w", err)
		}
		bodyBytes = b
	}
	for attempt := range 2 {
		var rdr io.Reader
		if bodyBytes != nil {
			rdr = bytes.NewReader(bodyBytes)
		}
		req, err := http.NewRequestWithContext(ctx, method, c.endpoint(path), rdr)
		if err != nil {
			return fmt.Errorf("build request: %w", err)
		}
		req.Header.Set("Authorization", "Bearer "+c.bearer(ctx, method))
		req.Header.Set("User-Agent", c.userAgent)
		if body != nil {
			req.Header.Set("Content-Type", "application/json")
		}
		resp, err := c.http.Do(req)
		if err != nil {
			return fmt.Errorf("request failed: %w", err)
		}
		ignored := resp.Header.Get(TokenIgnoredHeader)
		if ignored != "" && (ignored == "confined" || resp.StatusCode == http.StatusNotFound) {
			_ = resp.Body.Close()
			return &TokenIgnoredError{Reason: ignored}
		}
		if resp.StatusCode >= 200 && resp.StatusCode < 300 {
			defer func() { _ = resp.Body.Close() }()
			if ignored != "" {
				_, _ = fmt.Fprintf(os.Stderr, "warning: %s\n", &TokenIgnoredError{Reason: ignored})
			}
			return decodeResponse(resp, out)
		}
		b := ReadErrorBody(resp.Body)
		_ = resp.Body.Close()
		if retry, rerr := c.refreshOn401(ctx, resp.StatusCode, attempt); rerr != nil {
			return rerr
		} else if retry {
			continue
		}
		return &HTTPError{
			StatusCode: resp.StatusCode,
			Body:       strings.TrimSpace(string(b)),
			RetryAfter: ParseRetryAfterSeconds(resp.Header.Get("Retry-After")),
		}
	}
	return errors.New("api client: exhausted 401 retries")
}

// refreshOn401 asks the retry hook for a new token after the first 401 and
// reports whether the request should be sent again with it.
func (c *Client) refreshOn401(ctx context.Context, status, attempt int) (bool, error) {
	if status != http.StatusUnauthorized || attempt != 0 || c.retryOn401 == nil {
		return false, nil
	}
	newTok, err := c.retryOn401(ctx)
	if err != nil {
		return false, err
	}
	if newTok == "" {
		return false, nil
	}
	c.token = newTok
	return true, nil
}

func decodeResponse(resp *http.Response, out any) error {
	if out == nil || resp.StatusCode == http.StatusNoContent {
		return nil
	}
	if raw, ok := out.(*json.RawMessage); ok {
		b, err := io.ReadAll(resp.Body)
		if err != nil {
			return fmt.Errorf("read response: %w", err)
		}
		*raw = b
		return nil
	}
	if err := json.NewDecoder(resp.Body).Decode(out); err != nil {
		return fmt.Errorf("decode response: %w", err)
	}
	return nil
}
