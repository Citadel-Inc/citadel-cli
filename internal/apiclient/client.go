// Package apiclient is the Bearer-authenticated HTTP wrapper used by every
// citadel-cli subcommand handler against the Citadel JSON API. It centralises
// the load-config / require-token / build-request / decode-or-error chain.
package apiclient

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/Rethunk-Tech/citadel-cli/internal/clicfg"
)

// Client is a thin Citadel-API client: server URL + bearer token + http.Client.
type Client struct {
	server string
	token  string
	http   *http.Client
}

// New builds a Client from a loaded clicfg.Config and the optional --server
// flag override. Returns the canonical "not authenticated" error if no token
// is configured.
func New(cfg clicfg.Config, flagServer string) (*Client, error) {
	if cfg.AccessToken == "" {
		return nil, errors.New("not authenticated; run 'citadel-cli auth login' first")
	}
	return &Client{
		server: strings.TrimRight(cfg.ResolveServerURL(flagServer), "/"),
		token:  cfg.AccessToken,
		http:   http.DefaultClient,
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
		return fmt.Errorf("server error %d: %s", resp.StatusCode, strings.TrimSpace(string(b)))
	}
	if out == nil || resp.StatusCode == http.StatusNoContent {
		return nil
	}
	if err := json.NewDecoder(resp.Body).Decode(out); err != nil {
		return fmt.Errorf("decode response: %w", err)
	}
	return nil
}
