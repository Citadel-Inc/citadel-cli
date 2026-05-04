// Package mcpclient is a tiny Streamable-HTTP client for MCP servers.
//
// One Client instance corresponds to one logical CLI invocation: callers
// must Initialize() before issuing ToolsList / ToolsCall / ResourcesList /
// ResourcesRead / PromptsList / PromptsGet (the server rejects non-initialize
// requests without an Mcp-Session-Id header that is only minted on initialize).
package mcpclient

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// ProtocolVersion is the MCP version this client speaks. Must match the
// server's accepted version per the handshake contract; mismatch is
// surfaced as a typed Error from Initialize.
const ProtocolVersion = "2025-11-25"

const (
	clientName    = "citadel-cli"
	clientVersion = "1"
	sessionHeader = "Mcp-Session-Id"
)

// Client holds the per-invocation MCP session.
type Client struct {
	ServerURL string
	Token     string
	HTTP      *http.Client

	sessionID   string
	serverProto string
	serverInfo  ServerInfo
	nextID      int
}

// ServerInfo mirrors the server's `serverInfo` block.
type ServerInfo struct {
	Name    string `json:"name"`
	Version string `json:"version"`
}

// Tool mirrors a single entry in tools/list.
type Tool struct {
	Name        string         `json:"name"`
	Description string         `json:"description,omitempty"`
	InputSchema map[string]any `json:"inputSchema,omitempty"`
}

// MCPResource is one row from resources/list.
type MCPResource struct {
	URI         string `json:"uri"`
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	MimeType    string `json:"mimeType,omitempty"`
}

// MCPPrompt is one row from prompts/list.
type MCPPrompt struct {
	Name        string              `json:"name"`
	Description string              `json:"description,omitempty"`
	Arguments   []MCPPromptArgument `json:"arguments,omitempty"`
}

// MCPPromptArgument mirrors prompts/list argument metadata.
type MCPPromptArgument struct {
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	Required    *bool  `json:"required,omitempty"`
}

// New constructs a Client with the given timeout. Pass 0 for no timeout
// override (uses 60s default per spec R2).
func New(serverURL, token string, timeout time.Duration) *Client {
	if timeout <= 0 {
		timeout = 60 * time.Second
	}
	return &Client{
		ServerURL: serverURL,
		Token:     token,
		HTTP:      &http.Client{Timeout: timeout},
	}
}

// Initialize performs the MCP initialize handshake. On success the client
// captures Mcp-Session-Id + the server's protocolVersion. Mismatch with
// our advertised ProtocolVersion produces an ErrVersionMismatch.
func (c *Client) Initialize(ctx context.Context) error {
	params := map[string]any{
		"protocolVersion": ProtocolVersion,
		"capabilities":    map[string]any{},
		"clientInfo":      map[string]any{"name": clientName, "version": clientVersion},
	}
	var out struct {
		ProtocolVersion string     `json:"protocolVersion"`
		ServerInfo      ServerInfo `json:"serverInfo"`
	}
	sessionID, err := c.call(ctx, "initialize", params, &out)
	if err != nil {
		return err
	}
	if sessionID == "" {
		return fmt.Errorf("server did not return %s header on initialize", sessionHeader)
	}
	c.sessionID = sessionID
	c.serverProto = out.ProtocolVersion
	c.serverInfo = out.ServerInfo
	if out.ProtocolVersion != "" && out.ProtocolVersion != ProtocolVersion {
		return &Error{
			Kind:    KindVersionMismatch,
			Message: fmt.Sprintf("MCP protocol mismatch: client speaks %s, server speaks %s", ProtocolVersion, out.ProtocolVersion),
		}
	}
	return nil
}

// ServerInfo returns the negotiated server identity (post-Initialize).
func (c *Client) ServerInfoValue() ServerInfo { return c.serverInfo }

// ToolsList calls tools/list. Initialize must have run first.
func (c *Client) ToolsList(ctx context.Context) ([]Tool, error) {
	if c.sessionID == "" {
		return nil, fmt.Errorf("ToolsList: client not initialized")
	}
	var out struct {
		Tools []Tool `json:"tools"`
	}
	if _, err := c.call(ctx, "tools/list", map[string]any{}, &out); err != nil {
		return nil, err
	}
	return out.Tools, nil
}

// ToolCallResult is the shape returned by tools/call: a content array +
// optional isError flag. Callers pretty-print text blocks; --json emits
// the raw JSON.
type ToolCallResult struct {
	Content []map[string]any `json:"content"`
	IsError bool             `json:"isError,omitempty"`
	Raw     json.RawMessage  `json:"-"`
}

// ToolsCall calls tools/call with the given tool name + arguments.
// Initialize must have run first.
func (c *Client) ToolsCall(ctx context.Context, name string, args map[string]any) (*ToolCallResult, error) {
	if c.sessionID == "" {
		return nil, fmt.Errorf("ToolsCall: client not initialized")
	}
	params := map[string]any{"name": name, "arguments": args}
	var raw json.RawMessage
	if _, err := c.call(ctx, "tools/call", params, &raw); err != nil {
		return nil, err
	}
	var parsed ToolCallResult
	if err := json.Unmarshal(raw, &parsed); err != nil {
		return nil, fmt.Errorf("decode tools/call result: %w", err)
	}
	parsed.Raw = raw
	return &parsed, nil
}

// ResourcesList calls resources/list.
func (c *Client) ResourcesList(ctx context.Context) ([]MCPResource, error) {
	if c.sessionID == "" {
		return nil, fmt.Errorf("ResourcesList: client not initialized")
	}
	var out struct {
		Resources []MCPResource `json:"resources"`
	}
	if _, err := c.call(ctx, "resources/list", map[string]any{}, &out); err != nil {
		return nil, err
	}
	return out.Resources, nil
}

// ResourcesRead calls resources/read for a URI (citadel:// or repo://).
func (c *Client) ResourcesRead(ctx context.Context, uri string) (json.RawMessage, error) {
	if c.sessionID == "" {
		return nil, fmt.Errorf("ResourcesRead: client not initialized")
	}
	var raw json.RawMessage
	if _, err := c.call(ctx, "resources/read", map[string]any{"uri": uri}, &raw); err != nil {
		return nil, err
	}
	return raw, nil
}

// PromptsList calls prompts/list.
func (c *Client) PromptsList(ctx context.Context) ([]MCPPrompt, error) {
	if c.sessionID == "" {
		return nil, fmt.Errorf("PromptsList: client not initialized")
	}
	var out struct {
		Prompts []MCPPrompt `json:"prompts"`
	}
	if _, err := c.call(ctx, "prompts/list", map[string]any{}, &out); err != nil {
		return nil, err
	}
	return out.Prompts, nil
}

// PromptsGet calls prompts/get.
func (c *Client) PromptsGet(ctx context.Context, name string, arguments map[string]any) (json.RawMessage, error) {
	if c.sessionID == "" {
		return nil, fmt.Errorf("PromptsGet: client not initialized")
	}
	params := map[string]any{"name": name}
	if len(arguments) > 0 {
		params["arguments"] = arguments
	}
	var raw json.RawMessage
	if _, err := c.call(ctx, "prompts/get", params, &raw); err != nil {
		return nil, err
	}
	return raw, nil
}

// call is the JSON-RPC round-trip primitive. Returns the session id from
// the response (only meaningful on initialize). The result is decoded
// into out (any encoding/json target).
func (c *Client) call(ctx context.Context, method string, params any, out any) (string, error) {
	c.nextID++
	reqBody := map[string]any{
		"jsonrpc": "2.0",
		"id":      c.nextID,
		"method":  method,
	}
	if params != nil {
		reqBody["params"] = params
	}
	bodyBytes, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("marshal %s: %w", method, err)
	}
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, c.ServerURL, bytes.NewReader(bodyBytes))
	if err != nil {
		return "", fmt.Errorf("new request: %w", err)
	}
	httpReq.Header.Set("Authorization", "Bearer "+c.Token)
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Accept", "application/json")
	if c.sessionID != "" {
		httpReq.Header.Set(sessionHeader, c.sessionID)
	}

	resp, err := c.HTTP.Do(httpReq)
	if err != nil {
		return "", fmt.Errorf("%s: %w", method, err)
	}
	defer func() { _ = resp.Body.Close() }()
	respBytes, _ := io.ReadAll(resp.Body)

	if resp.StatusCode == http.StatusUnauthorized {
		return "", &Error{Kind: KindUnauthorized, Message: "unauthorized"}
	}

	var rpc struct {
		Result json.RawMessage `json:"result"`
		Error  *struct {
			Code    int    `json:"code"`
			Message string `json:"message"`
		} `json:"error"`
	}
	if len(respBytes) > 0 {
		if err := json.Unmarshal(respBytes, &rpc); err != nil {
			return "", fmt.Errorf("%s: decode response (HTTP %d): %w", method, resp.StatusCode, err)
		}
	}
	if rpc.Error != nil {
		return "", classifyJSONRPCError(rpc.Error.Code, rpc.Error.Message)
	}
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("%s: HTTP %d", method, resp.StatusCode)
	}
	if out != nil && len(rpc.Result) > 0 {
		if err := json.Unmarshal(rpc.Result, out); err != nil {
			return "", fmt.Errorf("%s: decode result: %w", method, err)
		}
	}
	return resp.Header.Get(sessionHeader), nil
}
