package cmd

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"github.com/Rethunk-Tech/citadel/internal/clicfg"
)

// McpCmd is the parent for `citadel mcp ...`. Speaks the MCP Streamable
// HTTP protocol (POST /mcp with `tools/list` or `tools/call` JSON-RPC).
//
// Authentication: an agent-token Bearer is required. The CLI takes the
// token via `--token` (one-shot) OR by setting CITADEL_AGENT_TOKEN. We
// deliberately do NOT reuse `cfg.AccessToken` (the Supabase user JWT) for
// MCP because oauth_jwt has a different surface (waitlist-gated, OIDC).
// Operators run `citadel token issue --agent X` to mint an agent token,
// then export it for `citadel mcp ...` calls.
var McpCmd = &cobra.Command{
	Use:   "mcp",
	Short: "Interact with MCP tools",
	Long: `Commands for listing and calling MCP tools via the server.

Authentication uses an AGENT TOKEN (issued via 'citadel token issue'), not
the Supabase user JWT. Pass via --token or CITADEL_AGENT_TOKEN env var.`,
}

var toolsCmd = &cobra.Command{
	Use:   "tools",
	Short: "List available MCP tools",
	Long: `Retrieves and displays all available MCP tools from the configured server,
including tool name and description.`,
	RunE: runMcpTools,
}

var callCmd = &cobra.Command{
	Use:   "call <tool>",
	Short: "Call an MCP tool with arguments",
	Long: `Invokes a named MCP tool with optional --arg key=value pairs.
Results are pretty-printed by default; use --json for raw output.`,
	Args: cobra.ExactArgs(1),
	RunE: runMcpCall,
}

func runMcpTools(cmd *cobra.Command, args []string) error {
	cfg, err := clicfg.Load()
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}
	flagServer, _ := cmd.Flags().GetString("server")
	flagToken, _ := cmd.Flags().GetString("token")
	mcpURL, agentToken, err := mcpDestination(cfg, flagServer, flagToken)
	if err != nil {
		return err
	}

	resp, err := mcpJSONRPC(mcpURL, agentToken, "tools/list", nil)
	if err != nil {
		return err
	}
	tools, ok := resp["tools"].([]any)
	if !ok {
		return fmt.Errorf("unexpected response shape: missing tools array")
	}
	for _, t := range tools {
		tm, ok := t.(map[string]any)
		if !ok {
			continue
		}
		name, _ := tm["name"].(string)
		desc, _ := tm["description"].(string)
		if desc == "" {
			fmt.Println(name)
		} else {
			fmt.Printf("%s\t%s\n", name, desc)
		}
	}
	return nil
}

func runMcpCall(cmd *cobra.Command, args []string) error {
	toolName := args[0]
	cfg, err := clicfg.Load()
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}
	flagServer, _ := cmd.Flags().GetString("server")
	flagToken, _ := cmd.Flags().GetString("token")
	rawJSON, _ := cmd.Flags().GetBool("json")
	argPairs, _ := cmd.Flags().GetStringSlice("arg")

	mcpURL, agentToken, err := mcpDestination(cfg, flagServer, flagToken)
	if err != nil {
		return err
	}

	toolArgs := map[string]any{}
	for _, p := range argPairs {
		k, v, ok := strings.Cut(p, "=")
		if !ok {
			return fmt.Errorf("bad --arg %q (expected key=value)", p)
		}
		toolArgs[k] = v
	}

	resp, err := mcpJSONRPC(mcpURL, agentToken, "tools/call", map[string]any{
		"name":      toolName,
		"arguments": toolArgs,
	})
	if err != nil {
		return err
	}
	if rawJSON {
		out, _ := json.MarshalIndent(resp, "", "  ")
		fmt.Println(string(out))
		return nil
	}
	// Pretty-print: surface text content blocks first, fall back to raw JSON.
	if content, ok := resp["content"].([]any); ok {
		for _, c := range content {
			cm, ok := c.(map[string]any)
			if !ok {
				continue
			}
			if cm["type"] == "text" {
				if text, ok := cm["text"].(string); ok {
					fmt.Println(text)
				}
			}
		}
		if isErr, _ := resp["isError"].(bool); isErr {
			os.Exit(2)
		}
		return nil
	}
	out, _ := json.MarshalIndent(resp, "", "  ")
	fmt.Println(string(out))
	return nil
}

// mcpDestination resolves the /mcp URL + bearer token from cfg + flags +
// env. Server URL precedence is the standard one (cfg.ResolveServerURL);
// the agent token has its own --token / CITADEL_AGENT_TOKEN precedence
// distinct from cfg.AccessToken (which is the Supabase JWT and the wrong
// credential for /mcp).
func mcpDestination(cfg clicfg.Config, flagServer, flagToken string) (mcpURL, token string, err error) {
	server := cfg.ResolveServerURL(flagServer)
	// /mcp lives on the mcp.src.land subdomain by default; if the operator
	// pointed --server at api.src.land, the mcp endpoint path is the same
	// shape (POST /mcp on whichever host they pointed at).
	mcpURL = strings.TrimRight(server, "/") + "/mcp"
	// Heuristic: if server is api.src.land, swap to mcp.src.land for /mcp.
	mcpURL = strings.Replace(mcpURL, "api.src.land/mcp", "mcp.src.land/mcp", 1)

	token = flagToken
	if token == "" {
		token = os.Getenv("CITADEL_AGENT_TOKEN")
	}
	if token == "" {
		return "", "", fmt.Errorf("agent token required: pass --token or set CITADEL_AGENT_TOKEN (issue one with 'citadel token issue --agent <name>')")
	}
	return mcpURL, token, nil
}

// mcpJSONRPC issues a single JSON-RPC 2.0 request to /mcp and returns the
// `result` object (or an error built from `error`). v1 uses an integer
// request id of 1 and does not multiplex.
func mcpJSONRPC(url, token, method string, params any) (map[string]any, error) {
	reqBody := map[string]any{
		"jsonrpc": "2.0",
		"id":      1,
		"method":  method,
	}
	if params != nil {
		reqBody["params"] = params
	}
	bodyBytes, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}
	httpReq, err := http.NewRequest("POST", url, bytes.NewReader(bodyBytes))
	if err != nil {
		return nil, fmt.Errorf("new request: %w", err)
	}
	httpReq.Header.Set("Authorization", "Bearer "+token)
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Accept", "application/json")

	resp, err := http.DefaultClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()
	respBytes, _ := io.ReadAll(resp.Body)
	if resp.StatusCode == http.StatusUnauthorized {
		return nil, fmt.Errorf("unauthorized — check CITADEL_AGENT_TOKEN / --token")
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP %d: %s", resp.StatusCode, strings.TrimSpace(string(respBytes)))
	}

	var rpc struct {
		Result map[string]any `json:"result"`
		Error  *struct {
			Code    int    `json:"code"`
			Message string `json:"message"`
		} `json:"error"`
	}
	if err := json.Unmarshal(respBytes, &rpc); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}
	if rpc.Error != nil {
		return nil, fmt.Errorf("MCP error %d: %s", rpc.Error.Code, rpc.Error.Message)
	}
	return rpc.Result, nil
}

func init() {
	McpCmd.AddCommand(toolsCmd)
	McpCmd.AddCommand(callCmd)

	// --token applies to both subcommands.
	McpCmd.PersistentFlags().String("token", "", "Agent token (overrides CITADEL_AGENT_TOKEN env var)")

	callCmd.Flags().StringSlice("arg", []string{}, "Tool arguments as key=value pairs (repeatable)")
	callCmd.Flags().Bool("json", false, "Output raw JSON-RPC result")
}
