package cmd

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/Rethunk-Tech/citadel/cmd/citadel-cli/internal/mcpclient"
	"github.com/Rethunk-Tech/citadel/internal/clicfg"
)

// McpCmd is the parent for `citadel-cli mcp ...`. Speaks the MCP Streamable
// HTTP protocol against /mcp.
//
// Authentication: defaults to cfg.AccessToken (Supabase JWT from
// `citadel-cli auth login`). Override with --token or CITADEL_AGENT_TOKEN
// for agent / CI use. The MCP server's verifyBearer (per go-mcp-oauth
// A2) tries JWT first then falls through to agent_tokens, so either
// works at the resource-server boundary.
var McpCmd = &cobra.Command{
	Use:   "mcp",
	Short: "Interact with MCP tools",
	Long: `Commands for listing and calling MCP tools via the Citadel MCP server.

Authentication defaults to your Supabase JWT from 'citadel-cli auth login'.
Override with --token or CITADEL_AGENT_TOKEN for agent / CI workflows.`,
}

var toolsCmd = &cobra.Command{
	Use:   "tools",
	Short: "List available MCP tools",
	RunE:  runMcpTools,
}

var callCmd = &cobra.Command{
	Use:   "call <tool>",
	Short: "Call an MCP tool with arguments",
	Long: `Invokes a named MCP tool with optional --arg key=value pairs.
Results are pretty-printed by default; use --json for the raw JSON-RPC
response. Args coerce by default: digits→number, CSV→array, else string.
Use --arg-string key=value to force string for a single arg.`,
	Args: cobra.ExactArgs(1),
	RunE: runMcpCall,
}

func runMcpTools(cmd *cobra.Command, _ []string) error {
	c, err := dialMCP(cmd)
	if err != nil {
		return err
	}
	tools, err := c.ToolsList(cmdContext(cmd))
	if err != nil {
		return surfaceErr(err)
	}
	for _, t := range tools {
		if t.Description == "" {
			fmt.Println(t.Name)
		} else {
			fmt.Printf("%s\t%s\n", t.Name, t.Description)
		}
	}
	return nil
}

func runMcpCall(cmd *cobra.Command, args []string) error {
	toolName := args[0]
	rawJSON, _ := cmd.Flags().GetBool("json")
	argPairs, _ := cmd.Flags().GetStringSlice("arg")
	stringArgPairs, _ := cmd.Flags().GetStringSlice("arg-string")

	toolArgs := map[string]any{}
	for _, p := range argPairs {
		k, v, ok := strings.Cut(p, "=")
		if !ok {
			return fmt.Errorf("bad --arg %q (expected key=value)", p)
		}
		toolArgs[k] = coerceArg(v)
	}
	for _, p := range stringArgPairs {
		k, v, ok := strings.Cut(p, "=")
		if !ok {
			return fmt.Errorf("bad --arg-string %q (expected key=value)", p)
		}
		toolArgs[k] = v
	}

	c, err := dialMCP(cmd)
	if err != nil {
		return err
	}
	res, err := c.ToolsCall(cmdContext(cmd), toolName, toolArgs)
	if err != nil {
		return surfaceErr(err)
	}
	if rawJSON {
		var pretty any
		_ = json.Unmarshal(res.Raw, &pretty)
		out, _ := json.MarshalIndent(pretty, "", "  ")
		fmt.Println(string(out))
	} else {
		printToolResult(res)
	}
	if res.IsError {
		os.Exit(2)
	}
	return nil
}

// dialMCP loads config + flags + env into a connected (Initialize-d)
// mcpclient. Token precedence: --token > CITADEL_AGENT_TOKEN >
// cfg.AccessToken.
func dialMCP(cmd *cobra.Command) (*mcpclient.Client, error) {
	cfg, err := clicfg.Load()
	if err != nil {
		return nil, fmt.Errorf("load config: %w", err)
	}
	flagServer, _ := cmd.Flags().GetString("server")
	flagToken, _ := cmd.Flags().GetString("token")
	timeoutSecs, _ := cmd.Flags().GetInt("timeout")

	mcpURL := resolveMCPURL(cfg.ResolveServerURL(flagServer))
	token := pickToken(flagToken, cfg.AccessToken)
	if token == "" {
		return nil, errors.New("no auth token: run `citadel-cli auth login` (or pass --token / set CITADEL_AGENT_TOKEN)")
	}

	c := mcpclient.New(mcpURL, token, time.Duration(timeoutSecs)*time.Second)
	if err := c.Initialize(cmdContext(cmd)); err != nil {
		return nil, surfaceErr(err)
	}
	return c, nil
}

// resolveMCPURL turns the resolved server base URL into the /mcp path.
// If the operator pointed --server at api.src.land, swap to mcp.src.land
// since the production MCP endpoint lives on its own subdomain.
func resolveMCPURL(server string) string {
	url := strings.TrimRight(server, "/") + "/mcp"
	return strings.Replace(url, "api.src.land/mcp", "mcp.src.land/mcp", 1)
}

// pickToken applies the token-precedence chain: --token > env > JWT.
func pickToken(flagToken, jwt string) string {
	if flagToken != "" {
		return flagToken
	}
	if env := os.Getenv("CITADEL_AGENT_TOKEN"); env != "" {
		return env
	}
	return jwt
}

// surfaceErr maps mcpclient errors to user copy. Auth failures point at
// `citadel-cli auth login` per spec §Auth; everything else passes through.
func surfaceErr(err error) error {
	if mcpclient.IsUnauthorized(err) {
		return errors.New("unauthorized: run `citadel-cli auth login` to refresh your session, or pass --token / set CITADEL_AGENT_TOKEN")
	}
	return err
}

// cmdContext returns the cobra command's context (Go 1.21+). Falls back
// to context.Background for the (unreachable) nil case.
func cmdContext(cmd *cobra.Command) context.Context {
	if ctx := cmd.Context(); ctx != nil {
		return ctx
	}
	return context.Background()
}

// printToolResult pretty-prints a tools/call result. Text content blocks
// emit one per line; non-text content falls through to JSON.
func printToolResult(res *mcpclient.ToolCallResult) {
	if len(res.Content) == 0 {
		return
	}
	for _, c := range res.Content {
		if c["type"] == "text" {
			if text, ok := c["text"].(string); ok {
				fmt.Println(text)
				continue
			}
		}
		out, _ := json.MarshalIndent(c, "", "  ")
		fmt.Println(string(out))
	}
}

// coerceArg implements the spec A6 / Q2 coercion: digit-only → number,
// CSV → array (with each element coerced recursively), bare boolean
// literals → bool, everything else → string. --arg-string opts out.
func coerceArg(v string) any {
	if v == "true" {
		return true
	}
	if v == "false" {
		return false
	}
	if n, ok := parseInt(v); ok {
		return n
	}
	if f, ok := parseFloat(v); ok {
		return f
	}
	if strings.Contains(v, ",") {
		parts := strings.Split(v, ",")
		out := make([]any, 0, len(parts))
		for _, p := range parts {
			out = append(out, coerceArg(p))
		}
		return out
	}
	return v
}

// parseInt accepts only the canonical integer form: optional leading
// minus, then digits with no leading zero (except literal "0"). This
// avoids treating zip codes / IDs like "07823" as numbers and clobbering
// their leading zeros.
func parseInt(v string) (int64, bool) {
	if v == "" {
		return 0, false
	}
	s := strings.TrimPrefix(v, "-")
	if s == "" {
		return 0, false
	}
	if len(s) > 1 && s[0] == '0' {
		return 0, false
	}
	for _, r := range s {
		if r < '0' || r > '9' {
			return 0, false
		}
	}
	n, err := strconv.ParseInt(v, 10, 64)
	if err != nil {
		return 0, false
	}
	return n, true
}

// parseFloat accepts a decimal with at least one dot and digits on both
// sides; rejects scientific notation, hex, and ambiguous forms.
func parseFloat(v string) (float64, bool) {
	if !strings.Contains(v, ".") {
		return 0, false
	}
	s := strings.TrimPrefix(v, "-")
	dot := strings.Index(s, ".")
	if dot <= 0 || dot == len(s)-1 {
		return 0, false
	}
	for _, r := range s {
		if r != '.' && (r < '0' || r > '9') {
			return 0, false
		}
	}
	f, err := strconv.ParseFloat(v, 64)
	if err != nil {
		return 0, false
	}
	return f, true
}

func init() {
	McpCmd.AddCommand(toolsCmd)
	McpCmd.AddCommand(callCmd)

	McpCmd.PersistentFlags().String("token", "", "Auth token (overrides CITADEL_AGENT_TOKEN env var; defaults to your `citadel-cli auth login` session JWT)")
	McpCmd.PersistentFlags().Int("timeout", 60, "Per-call HTTP timeout in seconds")

	callCmd.Flags().StringSlice("arg", []string{}, "Tool arguments as key=value pairs (digits→number, CSV→array, else string)")
	callCmd.Flags().StringSlice("arg-string", []string{}, "Tool arguments forced to string (no coercion)")
	callCmd.Flags().Bool("json", false, "Output raw JSON-RPC tools/call result")
}
