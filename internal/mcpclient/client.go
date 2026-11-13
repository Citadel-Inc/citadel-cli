// Package mcpclient is the CLI's MCP client, a thin layer over the official
// go-sdk Streamable HTTP transport.
//
// One Client corresponds to one CLI invocation: Initialize before any other
// call. List/read/get methods are retried (each retry on a fresh session,
// because the SDK closes a session after a non-transient server failure);
// tools/call and initialize are single-shot. The SDK's own SSE resumption is
// disabled so a tool call is never replayed, and the standalone server-push
// stream is not opened because no CLI command consumes server messages.
package mcpclient

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"
	"sync/atomic"
	"syscall"
	"time"

	"github.com/modelcontextprotocol/go-sdk/jsonrpc"
	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/Citadel-Inc/citadel-cli/internal/httpx"
)

// ProtocolVersion is the MCP version this client speaks. A server that
// negotiates a different version is surfaced as KindVersionMismatch.
const ProtocolVersion = "2025-11-25"

const (
	clientName      = "citadel-cli"
	clientVersion   = "1"
	maxAttempts     = 3
	maxResponseBody = 32 << 20
)

// Options are the per-invocation knobs surfaced from the CLI persistent flags.
type Options struct {
	Verbose   bool
	DebugHTTP bool
}

// Client holds the per-invocation MCP session.
type Client struct {
	url     string
	http    *http.Client
	tap     *authTransport
	sdk     *mcp.Client
	session *mcp.ClientSession
}

// ToolCallResult is the shape returned by tools/call: a content array +
// optional isError flag. Callers pretty-print text blocks; --json emits
// the raw JSON.
type ToolCallResult struct {
	Content []map[string]any `json:"content"`
	IsError bool             `json:"isError,omitempty"`
	Raw     json.RawMessage  `json:"-"`
}

// New constructs a Client with the given timeout and observability options.
// Pass 0 timeout for the 60s default (per spec R2).
func New(serverURL, token string, timeout time.Duration, opts Options) *Client {
	if timeout <= 0 {
		timeout = 60 * time.Second
	}
	tap := &authTransport{
		base:  httpx.Stack(nil, httpx.Options{Verbose: opts.Verbose, DebugHTTP: opts.DebugHTTP}),
		token: token,
	}
	if u, err := url.Parse(serverURL); err == nil {
		tap.scheme, tap.host = u.Scheme, u.Host
	}
	return &Client{
		url:  serverURL,
		http: &http.Client{Timeout: timeout, Transport: tap},
		tap:  tap,
		sdk: mcp.NewClient(
			&mcp.Implementation{Name: clientName, Version: clientVersion},
			&mcp.ClientOptions{Capabilities: &mcp.ClientCapabilities{}},
		),
	}
}

// authTransport adds the bearer token and records the last HTTP status, which
// is the only reliable signal for 401 and for retryable 5xx: the SDK folds
// both into opaque errors.
type authTransport struct {
	base   http.RoundTripper
	token  string
	status atomic.Int64
	// Redirects reach RoundTrip too; the token and session id go only to the configured origin.
	scheme, host string
}

func (a *authTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	req = req.Clone(req.Context())
	if req.URL.Scheme == a.scheme && req.URL.Host == a.host {
		req.Header.Set("Authorization", "Bearer "+a.token)
	} else {
		req.Header.Del("Authorization")
		req.Header.Del("Mcp-Session-Id")
	}
	resp, err := a.base.RoundTrip(req)
	if resp != nil && resp.Body != nil {
		resp.Body = capResponseBody(resp.Body)
	}
	if req.Method == http.MethodDelete {
		// The SDK's session teardown follows a failed call; its status would mask the failure.
		return resp, err
	}
	if err != nil {
		a.status.Store(0)
		return nil, err
	}
	a.status.Store(int64(resp.StatusCode))
	return resp, nil
}

func capResponseBody(rc io.ReadCloser) io.ReadCloser {
	return &cappedBody{r: io.LimitReader(rc, maxResponseBody+1), c: rc, max: maxResponseBody}
}

type cappedBody struct {
	r   io.Reader
	c   io.Closer
	n   int64
	max int64
}

func (c *cappedBody) Read(p []byte) (int, error) {
	n, err := c.r.Read(p)
	c.n += int64(n)
	if c.n > c.max {
		return n, fmt.Errorf("streamable HTTP response body exceeds the %d MiB limit", c.max>>20)
	}
	return n, err
}

func (c *cappedBody) Close() error { return c.c.Close() }

// Initialize performs the MCP initialize handshake. A server that settles on a
// protocol version other than ProtocolVersion produces KindVersionMismatch.
func (c *Client) Initialize(ctx context.Context) error {
	c.tap.status.Store(0)
	session, err := c.sdk.Connect(ctx, &mcp.StreamableClientTransport{
		Endpoint:             c.url,
		HTTPClient:           c.http,
		DisableStandaloneSSE: true,
		MaxRetries:           -1,
	}, &mcp.ClientSessionOptions{ProtocolVersion: ProtocolVersion})
	if err != nil {
		return c.classify(err)
	}
	if v := session.InitializeResult().ProtocolVersion; v != "" && v != ProtocolVersion {
		_ = session.Close()
		return versionMismatch(v)
	}
	c.session = session
	return nil
}

// Close ends the session (best effort; the server may already have dropped it).
func (c *Client) Close() error {
	if c.session == nil {
		return nil
	}
	s := c.session
	c.session = nil
	return s.Close()
}

// ToolsList calls tools/list (first page).
func (c *Client) ToolsList(ctx context.Context) ([]*mcp.Tool, error) {
	res, err := retry(ctx, c, func() (*mcp.ListToolsResult, error) {
		return c.session.ListTools(ctx, nil)
	})
	if err != nil {
		return nil, err
	}
	return res.Tools, nil
}

// ToolsCall calls tools/call exactly once.
func (c *Client) ToolsCall(ctx context.Context, name string, args map[string]any) (*ToolCallResult, error) {
	if c.session == nil {
		return nil, errors.New("ToolsCall: client not initialized")
	}
	params := &mcp.CallToolParams{Name: name}
	if args != nil {
		params.Arguments = args
	}
	res, err := c.session.CallTool(ctx, params)
	if err != nil {
		return nil, c.classify(err)
	}
	raw, err := json.Marshal(res)
	if err != nil {
		return nil, fmt.Errorf("encode tools/call result: %w", err)
	}
	var parsed ToolCallResult
	if err := json.Unmarshal(raw, &parsed); err != nil {
		return nil, fmt.Errorf("decode tools/call result: %w", err)
	}
	parsed.Raw = raw
	return &parsed, nil
}

// ResourcesList calls resources/list (first page).
func (c *Client) ResourcesList(ctx context.Context) ([]*mcp.Resource, error) {
	res, err := retry(ctx, c, func() (*mcp.ListResourcesResult, error) {
		return c.session.ListResources(ctx, nil)
	})
	if err != nil {
		return nil, err
	}
	return res.Resources, nil
}

// ResourcesRead calls resources/read for a URI (citadel:// or repo://).
func (c *Client) ResourcesRead(ctx context.Context, uri string) (json.RawMessage, error) {
	res, err := retry(ctx, c, func() (*mcp.ReadResourceResult, error) {
		return c.session.ReadResource(ctx, &mcp.ReadResourceParams{URI: uri})
	})
	if err != nil {
		return nil, err
	}
	return json.Marshal(res)
}

// PromptsList calls prompts/list (first page).
func (c *Client) PromptsList(ctx context.Context) ([]*mcp.Prompt, error) {
	res, err := retry(ctx, c, func() (*mcp.ListPromptsResult, error) {
		return c.session.ListPrompts(ctx, nil)
	})
	if err != nil {
		return nil, err
	}
	return res.Prompts, nil
}

// PromptsGet calls prompts/get. The protocol carries prompt arguments as
// strings, so non-string values are rendered with their JSON encoding.
func (c *Client) PromptsGet(ctx context.Context, name string, arguments map[string]any) (json.RawMessage, error) {
	var args map[string]string
	if len(arguments) > 0 {
		args = make(map[string]string, len(arguments))
		for k, v := range arguments {
			if s, ok := v.(string); ok {
				args[k] = s
				continue
			}
			b, err := json.Marshal(v)
			if err != nil {
				return nil, fmt.Errorf("encode prompt argument %q: %w", k, err)
			}
			args[k] = string(b)
		}
	}
	res, err := retry(ctx, c, func() (*mcp.GetPromptResult, error) {
		return c.session.GetPrompt(ctx, &mcp.GetPromptParams{Name: name, Arguments: args})
	})
	if err != nil {
		return nil, err
	}
	return json.Marshal(res)
}

// retry runs an idempotent call, repeating it on a fresh session after a
// transient failure.
func retry[T any](ctx context.Context, c *Client, fn func() (T, error)) (T, error) {
	var zero T
	if c.session == nil {
		return zero, errors.New("client not initialized")
	}
	for attempt := range maxAttempts {
		c.tap.status.Store(0)
		out, err := fn()
		if err == nil {
			return out, nil
		}
		if attempt == maxAttempts-1 || !c.transient(ctx, err) {
			return zero, c.classify(err)
		}
		if err := waitRetry(ctx, attempt); err != nil {
			return zero, err
		}
		_ = c.session.Close()
		c.session = nil
		if err := c.Initialize(ctx); err != nil {
			return zero, err
		}
	}
	return zero, errors.New("retry attempts exhausted")
}

func (c *Client) transient(ctx context.Context, err error) bool {
	if ctx.Err() != nil || c.tap.status.Load() == http.StatusUnauthorized {
		return false
	}
	switch c.tap.status.Load() {
	case 408, 425, 429, 500, 502, 503, 504:
		return true
	}
	var netErr net.Error
	if errors.As(err, &netErr) && netErr.Timeout() {
		return true
	}
	return errors.Is(err, syscall.ECONNRESET) ||
		errors.Is(err, syscall.ECONNREFUSED) ||
		errors.Is(err, syscall.EPIPE) ||
		errors.Is(err, io.EOF) ||
		errors.Is(err, io.ErrUnexpectedEOF)
}

var waitRetry = waitRetryPause

func waitRetryPause(ctx context.Context, attempt int) error {
	timer := time.NewTimer(100 * time.Millisecond * time.Duration(1<<attempt))
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}

// classify maps SDK and transport failures onto the typed Error surface.
func (c *Client) classify(err error) error {
	if c.tap.status.Load() == http.StatusUnauthorized {
		return &Error{Kind: KindUnauthorized, Message: "unauthorized"}
	}
	if je, ok := errors.AsType[*jsonrpc.Error](err); ok {
		return classifyJSONRPCError(int(je.Code), je.Message)
	}
	// The SDK reports a server's unsupported version only as an unexported
	// error type, so its message is the one carrier of the offending version.
	if _, v, ok := strings.Cut(err.Error(), "unsupported protocol version: "); ok {
		return versionMismatch(strings.Trim(v, `"`))
	}
	return err
}

func versionMismatch(server string) *Error {
	return &Error{
		Kind:    KindVersionMismatch,
		Message: fmt.Sprintf("MCP protocol mismatch: client speaks %s, server speaks %s", ProtocolVersion, server),
	}
}
